package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/NotNull92/hera-agent-godot/internal/protocol"
)

// engineFlagPath is set from the leading global flag --engine <path> and may
// also be given after an engine-backed command. No-editor commands never
// require a live editor; the engine comes from this flag, HERA_GODOT_ENGINE or
// GODOT_BIN.
var engineFlagPath string

const (
	engineOutputLimit       = 64 * 1024
	engineVersionTimeout    = 10 * time.Second
	engineProjectRunTimeout = 300 * time.Second
)

// engineWaitDelay bounds both the inherited-pipe wait after the started process
// exits and Hera's own wait for a process it asked to terminate, so a surviving
// descendant or a termination that failed cannot block the caller. Tests shorten
// it instead of sleeping for the production value.
var engineWaitDelay = 5 * time.Second

// newEngineCommand builds the engine process with its hardening applied:
// bounded pipe draining and an owned process group on Unix.
func newEngineCommand(executable string, args []string) *exec.Cmd {
	command := exec.Command(executable, args...)
	command.WaitDelay = engineWaitDelay
	applyProcessGroup(command)
	return command
}

// engineRef is a resolved Godot engine executable.
type engineRef struct {
	Path   string
	Source string
}

// resolveEngine resolves the engine executable for no-editor commands.
func resolveEngine() (engineRef, error) {
	if engineFlagPath != "" {
		return validateEnginePath(engineRef{Path: engineFlagPath, Source: "flag"})
	}
	for _, name := range []string{"HERA_GODOT_ENGINE", "GODOT_BIN"} {
		if value := strings.TrimSpace(os.Getenv(name)); value != "" {
			return validateEnginePath(engineRef{Path: value, Source: "env:" + name})
		}
	}
	return engineRef{}, errors.New("no Godot engine: pass --engine <path> or set HERA_GODOT_ENGINE / GODOT_BIN")
}

func validateEnginePath(ref engineRef) (engineRef, error) {
	info, err := os.Stat(ref.Path)
	if err != nil {
		return engineRef{}, fmt.Errorf("engine %s: %w", ref.Path, err)
	}
	if info.IsDir() {
		return engineRef{}, fmt.Errorf("engine %s is a directory, not an executable", ref.Path)
	}
	if abs, err := filepath.Abs(ref.Path); err == nil {
		ref.Path = abs
	}
	ref.Path = resolveConsoleWrapper(ref.Path)
	return ref, nil
}

// resolveConsoleWrapper prefers the sibling engine executable over the Windows
// console wrapper, which starts the real engine as a child process (see
// docs/GODOT_EDITOR_ANALYSIS.md). Launching the engine itself lets the Job
// Object own the whole tree from the start.
func resolveConsoleWrapper(path string) string {
	if runtime.GOOS != "windows" || !strings.HasSuffix(strings.ToLower(path), "_console.exe") {
		return path
	}
	sibling := path[:len(path)-len("_console.exe")] + ".exe"
	if fileExists(sibling) {
		return sibling
	}
	return path
}

// boundedOutput keeps at most limit bytes and records truncation. Writes and
// snapshots are mutex-guarded, so a bounded wait that abandons a live engine can
// still read what was drained without racing the pipe copier.
type boundedOutput struct {
	mu        sync.Mutex
	text      []byte
	truncated bool
	limit     int
}

func newBoundedOutput(limit int) *boundedOutput {
	if limit <= 0 {
		limit = engineOutputLimit
	}
	return &boundedOutput{limit: limit}
}

func (o *boundedOutput) Write(p []byte) (int, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.limit <= 0 {
		o.limit = engineOutputLimit
	}
	remaining := o.limit - len(o.text)
	if len(p) > remaining {
		if remaining > 0 {
			o.text = append(o.text, p[:remaining]...)
		}
		o.truncated = true
	} else {
		o.text = append(o.text, p...)
	}
	return len(p), nil
}

// snapshot returns the output collected so far and whether it was truncated.
func (o *boundedOutput) snapshot() (string, bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	return string(o.text), o.truncated
}

// processGuarder is the ownership contract runEngineBounded relies on. Tests
// replace engineGuardFactory to inject setup, resume and termination failures
// deterministically.
type processGuarder interface {
	resume() string
	terminate() string
	close()
}

// engineGuardFactory creates the guard for a started process.
var engineGuardFactory = func(pid int, process *os.Process) processGuarder {
	return newProcessGuard(pid, process)
}

// engineRunResult is one bounded engine process run.
type engineRunResult struct {
	Started   bool
	ExitCode  int
	Output    string
	Truncated bool
	TimedOut  bool
	// Abandoned reports that the wait for the process was not observed to finish
	// (including its pipe drain) inside the bound after cancellation, so nothing
	// about its work or its exit status is proven.
	Abandoned   bool
	Duration    time.Duration
	RunErr      error
	Termination string
}

// runEngineBounded runs the engine with bounded output. Ownership of the
// process is established at launch:
//
//   - Windows: the child is assigned to an owned Job Object (kill-on-close)
//     immediately after Start, so the tree can be terminated through a handle
//     even when the started process already exited;
//   - Unix: only the started process is signalled, through Go's own Process
//     bookkeeping, which refuses to signal after the child was reaped. Unix
//     descendants are deliberately not signalled by numeric id (see the
//     platform file);
//   - the timeout path records incomplete cleanup instead of reporting success,
//     and pipe draining is bounded so a surviving descendant cannot block the
//     caller forever.
//
// An already-canceled context starts nothing.
func runEngineBounded(ctx context.Context, executable string, args []string, dir string, env []string, limit int) engineRunResult {
	started := time.Now()
	result := engineRunResult{ExitCode: -1}
	if ctx.Err() != nil {
		// Never launch work for a deadline that already passed.
		result.TimedOut = true
		result.RunErr = errors.New("context already expired before launch")
		result.Duration = time.Since(started)
		return result
	}
	output := newBoundedOutput(limit)
	command := newEngineCommand(executable, args)
	command.Dir = dir
	if len(env) > 0 {
		command.Env = env
	}
	command.Stdout = output
	command.Stderr = output
	if err := command.Start(); err != nil {
		result.RunErr = err
		result.Duration = time.Since(started)
		return result
	}
	result.Started = true
	guard := engineGuardFactory(command.Process.Pid, command.Process)
	// Ownership closes the guard on every path below: the refusal branch closes it
	// after terminating, and the normal path uses closeGuard, which hands the close
	// to a termination worker that is still running instead of racing it.
	// The child was launched suspended so ownership is established before it can
	// create descendants; resume it now. A child that could not be owned is never
	// allowed to run, and the failure path is bounded so a failed kill cannot
	// block the caller.
	if failure := guard.resume(); failure != "" {
		termination := guard.terminate()
		guard.close()
		waited := make(chan struct{})
		go func() { _ = command.Wait(); close(waited) }()
		select {
		case <-waited:
		case <-time.After(engineWaitDelay):
			termination = strings.TrimSpace(termination + "; the started process did not exit after termination")
		}
		message := "engine launch failed: " + failure
		if termination != "" {
			message += " (" + termination + ")"
		}
		result.RunErr = errors.New(message)
		result.Termination = termination
		result.Duration = time.Since(started)
		return result
	}

	var mu sync.Mutex
	finished := false
	terminating := false
	guardClosed := false
	workerOwnsClose := false
	termination := ""
	stopped := make(chan struct{})
	cleanupDone := make(chan struct{})

	// closeGuard closes the guard exactly once, and never while the termination
	// worker is using it: ownership moves to that worker, which closes it when it
	// finishes. This keeps a timed-out caller from closing a handle the worker is
	// still passing to a native call.
	closeGuard := func() {
		mu.Lock()
		if guardClosed {
			mu.Unlock()
			return
		}
		if terminating {
			workerOwnsClose = true
			mu.Unlock()
			return
		}
		guardClosed = true
		mu.Unlock()
		guard.close()
	}
	defer closeGuard()

	go func() {
		defer close(cleanupDone)
		select {
		case <-ctx.Done():
		case <-stopped:
			return
		}
		mu.Lock()
		if finished {
			mu.Unlock()
			return
		}
		terminating = true
		mu.Unlock()

		failure := guard.terminate()

		mu.Lock()
		termination = failure
		terminating = false
		ownsClose := workerOwnsClose
		if ownsClose {
			guardClosed = true
		}
		mu.Unlock()
		if ownsClose {
			guard.close()
		}
	}()

	// A healthy run is never cut short: the caller waits for the process to exit on
	// its own. The bound applies only after the command context is cancelled and
	// termination is under way, so a kill that cannot stop the engine still cannot
	// hang Hera.
	waited := make(chan error, 1)
	go func() { waited <- command.Wait() }()
	var waitErr error
	exitObserved := false
	select {
	case waitErr = <-waited:
		exitObserved = true
	case <-ctx.Done():
		select {
		case waitErr = <-waited:
			exitObserved = true
		case <-time.After(engineWaitDelay):
		}
	}
	mu.Lock()
	finished = true
	mu.Unlock()
	close(stopped)
	terminationObserved := false
	select {
	case <-cleanupDone:
		terminationObserved = true
	case <-time.After(engineWaitDelay):
	}
	mu.Lock()
	if !exitObserved {
		termination = strings.TrimSpace(termination + "; the process exit was not observed within the wait bound")
	} else if !terminationObserved {
		termination = strings.TrimSpace(termination + "; termination did not report back")
	}
	finalTermination := termination
	mu.Unlock()

	if exitObserved && command.ProcessState != nil {
		// ProcessState belongs to Wait; reading it while Wait runs would race.
		result.ExitCode = command.ProcessState.ExitCode()
	}
	result.TimedOut = ctx.Err() != nil
	result.Abandoned = !exitObserved
	result.RunErr = waitErr
	text, truncated := output.snapshot()
	result.Output = strings.TrimSpace(text)
	result.Truncated = truncated
	result.Duration = time.Since(started)
	result.Termination = finalTermination
	return result
}

// engineVersion is the parsed --version string.
type engineVersion struct {
	String string
	Status string
	Mono   bool
	Commit string
}

var engineCommitPattern = regexp.MustCompile(`^[0-9a-f]{7,40}$`)

// parseEngineVersion parses strings like
// "4.7.2.stable.mono.official.ed1daf0bf". Unknown parts stay empty instead of
// guessing.
func parseEngineVersion(raw string) engineVersion {
	version := engineVersion{String: strings.TrimSpace(raw)}
	parts := strings.Split(version.String, ".")
	if len(parts) < 3 {
		return version
	}
	for _, part := range parts[3:] {
		switch {
		case engineCommitPattern.MatchString(part):
			version.Commit = part
		case part == "mono":
			version.Mono = true
		case part == "stable" || part == "rc" || part == "beta" || part == "alpha" || part == "dev":
			version.Status = part
		}
	}
	return version
}

// readEngineVersion runs --version and parses the first non-empty line.
func readEngineVersion(ctx context.Context, executable string) (engineVersion, error) {
	run := runEngineBounded(ctx, executable, []string{"--version"}, "", nil, 4096)
	if run.ExitCode != 0 {
		return engineVersion{}, fmt.Errorf("--version exited %d: %s", run.ExitCode, firstLine(run.Output))
	}
	for _, line := range strings.Split(run.Output, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return parseEngineVersion(line), nil
		}
	}
	return engineVersion{}, errors.New("--version printed nothing")
}

func firstLine(text string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(text), "\n")
	return strings.TrimSpace(line)
}

// helpFlags are the engine flags Hera gates commands on. Presence is read from
// this build's --help output, so a false value only means "not documented here"
// (hidden flags such as --class-db-json never appear).
var helpFlags = []string{
	"--headless", "--script", "--check-only", "--import",
	"--export-pack", "--export-release", "--export-debug", "--export-patch",
	"--doctool", "--gdscript-docs", "--dump-extension-api", "--dump-extension-api-with-docs",
	"--dap-port", "--lsp-port", "--write-movie", "--benchmark-file",
	"--quit-after", "--fixed-fps", "--scene", "--log-file", "--editor", "--recovery-mode",
}

// engineHelp is the parsed --help output for one engine binary. Editor is only
// meaningful when Available is true: a failed or partial probe proves nothing
// about the build kind.
type engineHelp struct {
	Available  bool
	Editor     bool
	Present    map[string]bool
	ProbeError string
}

func readEngineHelp(ctx context.Context, executable string) engineHelp {
	run := runEngineBounded(ctx, executable, []string{"--help"}, "", nil, 256*1024)
	if run.ExitCode != 0 || !run.Started || run.TimedOut || run.Output == "" || run.Truncated ||
		errors.Is(run.RunErr, exec.ErrWaitDelay) {
		reason := "engine --help did not complete"
		switch {
		case run.TimedOut:
			reason = "engine --help timed out"
		case !run.Started:
			reason = "engine --help did not run"
		case run.RunErr != nil && !errors.Is(run.RunErr, exec.ErrWaitDelay):
			reason = "engine --help could not run: " + run.RunErr.Error()
		case run.ExitCode != 0:
			reason = "engine --help exited " + strconv.Itoa(run.ExitCode)
		case run.Truncated:
			reason = "engine --help output was truncated"
		case run.Output == "":
			reason = "engine --help printed nothing"
		case errors.Is(run.RunErr, exec.ErrWaitDelay):
			reason = "engine --help output was incomplete"
		}
		return engineHelp{Present: map[string]bool{}, ProbeError: reason}
	}
	return scanHelpFlags(run.Output)
}

// ansiEscape matches the CSI sequences the engine colors --help output with.
var ansiEscape = regexp.MustCompile("\x1b\\[[0-9;?]*[ -/]*[@-~]")

// scanHelpFlags reads the flags this build documents in --help. A missing flag
// means "not documented here"; hidden flags never appear. Color codes are
// stripped first, otherwise a colored flag token never matches.
func scanHelpFlags(text string) engineHelp {
	help := engineHelp{Available: true, Present: map[string]bool{}}
	fields := strings.FieldsFunc(ansiEscape.ReplaceAllString(text, ""), func(r rune) bool {
		return r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == ','
	})
	for _, field := range fields {
		for _, flag := range helpFlags {
			if field == flag {
				help.Present[flag] = true
			}
		}
		if field == "--editor" {
			help.Editor = true
		}
	}
	return help
}

// engineIdentity is the engine descriptor engine-backed results carry.
func engineIdentity(ref engineRef, version engineVersion) map[string]any {
	identity := map[string]any{"path": filepath.ToSlash(ref.Path), "source": ref.Source}
	if version.String != "" {
		identity["version"] = version.String
	}
	return identity
}

// runResultFields is the shared status/output block of an engine-backed result.
// Commands add their own artifact fields on top.
func runResultFields(run engineRunResult) map[string]any {
	fields := map[string]any{
		"exit_code": run.ExitCode, "timed_out": run.TimedOut,
		"duration_ms": run.Duration.Milliseconds(),
		"output":      run.Output, "output_truncated": run.Truncated,
	}
	if run.Termination != "" {
		fields["termination_error"] = run.Termination
	}
	if errors.Is(run.RunErr, exec.ErrWaitDelay) {
		// The process exited but something kept its pipes open, so the captured
		// output may be incomplete.
		fields["output_incomplete"] = true
	}
	if run.Abandoned {
		fields["abandoned"] = true
	}
	return fields
}

// mergeResult copies shared run fields into a command result.
func mergeResult(result map[string]any, run engineRunResult) map[string]any {
	for key, value := range runResultFields(run) {
		result[key] = value
	}
	return result
}

// runFailureCode classifies an engine run that must not be reported as success.
func runFailureCode(run engineRunResult) string {
	switch {
	case run.Abandoned:
		// The exit status was never observed, so no work can be claimed.
		return "engine_abandoned"
	case run.TimedOut:
		return "engine_timeout"
	case errors.Is(run.RunErr, exec.ErrWaitDelay):
		return "output_incomplete"
	case run.ExitCode > 0:
		return "engine_exit_nonzero"
	case run.ExitCode == -1 && run.RunErr != nil:
		return run.RunErr.Error()
	}
	return ""
}

// runFailed reports whether a run failed, including incomplete output.
func runFailed(run engineRunResult) bool {
	return runFailureCode(run) != ""
}

// resultError returns the recorded failure code of a command result.
func resultError(result map[string]any) string {
	code, _ := result["error"].(string)
	return code
}

// failEngineResult prints the result with the stderr diagnostic the stream
// contract requires and returns 1.
func failEngineResult(result map[string]any) int {
	if code := printData(&protocol.Response{OK: true, Data: result}); code != 0 {
		return code
	}
	reason := resultError(result)
	if reason == "" {
		reason = "failed"
	}
	fmt.Fprintf(os.Stderr, "error: %s\n", reason)
	return 1
}

// finishEngineResult applies the run verdict to a command result, prints it
// once, and returns the command exit code.
func finishEngineResult(result map[string]any, run engineRunResult) int {
	if reason := runFailureCode(run); reason != "" && resultError(result) == "" {
		result["error"] = reason
	}
	if resultError(result) != "" {
		return failEngineResult(result)
	}
	if code := printData(&protocol.Response{OK: true, Data: result}); code != 0 {
		return code
	}
	return 0
}

// engineContextTimeout bounds one engine-backed command. --timeout (global)
// overrides the per-command default.
func engineContextTimeout(fallback time.Duration) (context.Context, context.CancelFunc) {
	timeout := requestTimeout
	if timeout == 0 {
		timeout = fallback
	}
	return context.WithTimeout(context.Background(), timeout)
}

// takeFlagValue reads "--name value" or "--name=value" starting at args[i].
// It returns the value and the index after it.
func takeFlagValue(args []string, i int, name string) (string, int, error) {
	if value, ok := strings.CutPrefix(args[i], name+"="); ok {
		if value == "" {
			return "", i, fmt.Errorf("%s requires a value", name)
		}
		return value, i + 1, nil
	}
	if i+1 >= len(args) {
		return "", i, fmt.Errorf("%s requires a value", name)
	}
	return args[i+1], i + 2, nil
}

func isFlag(arg string, name string) bool {
	return arg == name || strings.HasPrefix(arg, name+"=")
}

// resolveProjectDir returns the project directory for engine-backed commands:
// the --project value when given, else the working directory when it holds a
// project.godot (Godot's own --path semantics).
func resolveProjectDir(explicit string) (string, error) {
	dir := strings.TrimSpace(explicit)
	if dir == "" {
		dir = "."
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(filepath.Join(abs, "project.godot"))
	if err != nil || info.IsDir() {
		if explicit != "" {
			return "", fmt.Errorf("no project.godot in %s", abs)
		}
		return "", fmt.Errorf("no project.godot in %s (pass --project <dir>)", abs)
	}
	return abs, nil
}

// runEngineCommand is the shared entry point for engine-backed commands: it
// resolves the engine and applies the engine timeout.
func runEngineCommand(fallback time.Duration) (engineRef, context.Context, context.CancelFunc, error) {
	ref, err := resolveEngine()
	if err != nil {
		return engineRef{}, nil, nil, err
	}
	ctx, cancel := engineContextTimeout(fallback)
	return ref, ctx, cancel, nil
}

func runEngine(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: engine <info|doctor> [--engine <path>] [--project <dir>]")
		return 2
	}
	switch args[0] {
	case "info":
		return runEngineInfo(args[1:])
	case "doctor":
		return runEngineDoctor(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown engine subcommand %q (want info|doctor)\n", args[0])
		return 2
	}
}

func parseEngineArgs(args []string) (project string, err error) {
	for i := 0; i < len(args); i++ {
		switch {
		case isFlag(args[i], "--engine"):
			value, next, err := takeFlagValue(args, i, "--engine")
			if err != nil {
				return "", err
			}
			engineFlagPath = value
			i = next - 1
		case isFlag(args[i], "--project"):
			value, next, err := takeFlagValue(args, i, "--project")
			if err != nil {
				return "", err
			}
			project = value
			i = next - 1
		default:
			return "", fmt.Errorf("unknown flag %q", args[i])
		}
	}
	return project, nil
}

func runEngineInfo(args []string) int {
	project, err := parseEngineArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "engine info: %v\n", err)
		return 2
	}
	_ = project
	ref, ctx, cancel, err := runEngineCommand(engineVersionTimeout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "engine info: %v\n", err)
		return 1
	}
	defer cancel()
	info, err := collectEngineInfo(ctx, ref, true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "engine info: %v\n", err)
		return 1
	}
	if code := printData(&protocol.Response{OK: true, Data: info}); code != 0 {
		return code
	}
	return 0
}

// collectEngineInfo gathers engine identity. withHelp=false skips the --help run
// when only the version string is needed.
func collectEngineInfo(ctx context.Context, ref engineRef, withHelp bool) (map[string]any, error) {
	stat, statErr := os.Stat(ref.Path)
	info := map[string]any{
		"engine": map[string]any{"path": ref.Path, "source": ref.Source},
	}
	if statErr == nil {
		info["engine"].(map[string]any)["bytes"] = stat.Size()
	}
	version, err := readEngineVersion(ctx, ref.Path)
	if err != nil {
		return nil, err
	}
	versionData := map[string]any{"string": version.String, "mono": version.Mono}
	if version.Status != "" {
		versionData["status"] = version.Status
	}
	if version.Commit != "" {
		versionData["commit"] = version.Commit
	}
	info["version"] = versionData
	if !withHelp {
		return info, nil
	}
	help := readEngineHelp(ctx, ref.Path)
	info["help_available"] = help.Available
	if help.Available {
		info["editor_build"] = help.Editor
		present := map[string]any{}
		for _, flag := range helpFlags {
			present[flag] = help.Present[flag]
		}
		info["help_flags"] = present
	} else if help.ProbeError != "" {
		// Absence of evidence is not proof of a template build, and no flag
		// inventory is claimed for a probe that did not complete.
		info["help_probe_error"] = help.ProbeError
	}
	return info, nil
}

// editorBuildState reports whether the build kind is known and whether it is an
// editor build. An unavailable help probe leaves it unknown.
func editorBuildState(info map[string]any) (known bool, editor bool) {
	value, ok := info["editor_build"].(bool)
	if !ok {
		return false, false
	}
	return true, value
}

func runEngineDoctor(args []string) int {
	project, err := parseEngineArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "engine doctor: %v\n", err)
		return 2
	}
	ref, ctx, cancel, err := runEngineCommand(engineVersionTimeout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "engine doctor: %v\n", err)
		return 1
	}
	defer cancel()
	result := map[string]any{}
	checks := []map[string]any{}
	warnings := []string{}
	info, err := collectEngineInfo(ctx, ref, true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "engine doctor: %v\n", err)
		return 1
	}
	for key, value := range info {
		result[key] = value
	}
	version, versionOK := info["version"].(map[string]any)
	checks = append(checks, engineCheck("engine_resolved", true,
		fmt.Sprintf("%s (%s)", ref.Path, ref.Source)))
	checks = append(checks, engineCheck("engine_version", versionOK && version["string"] != "", fmt.Sprint(version["string"])))
	editorKnown, editorBuild := editorBuildState(info)
	switch {
	case !editorKnown:
		detail := "help probe unavailable: the build kind could not be observed"
		if probe, ok := info["help_probe_error"].(string); ok && probe != "" {
			detail += " (" + probe + ")"
		}
		checks = append(checks, engineCheck("editor_build", false, detail))
		warnings = append(warnings, "editor build kind unknown: --help did not complete, so editor-only flags cannot be confirmed")
	case editorBuild:
		checks = append(checks, engineCheck("editor_build", true, editorBuildDetail(true)))
	default:
		checks = append(checks, engineCheck("editor_build", false, editorBuildDetail(false)))
		warnings = append(warnings, "not an editor build: --import, --export-*, --doctool, --dap-port and --lsp-port are unavailable")
	}

	templateRoot, templateDir, templateFiles, templateErr := engineTemplateDir(version, ref.Path)
	template := map[string]any{}
	switch {
	case templateErr != nil:
		template["version_dir"] = ""
		warnings = append(warnings, "export templates not checked: "+templateErr.Error())
	case templateDir != "":
		template["root"], template["version_dir"], template["files"] = templateRoot, templateDir, templateFiles
		template["found"] = true
	default:
		template["root"], template["version_dir"], template["found"] = templateRoot, "", false
	}
	result["export_templates"] = template
	checks = append(checks, engineCheck("export_templates", template["found"] == true,
		fmt.Sprintf("looked for %s under %s; a custom template can still be configured in the preset", template["version_dir"], templateRoot)))

	projectDir, projectErr := resolveProjectDir(project)
	if projectErr != nil {
		result["project"] = map[string]any{"found": false, "reason": projectErr.Error()}
		checks = append(checks, engineCheck("project", false, projectErr.Error()))
	} else {
		projectInfo := map[string]any{"found": true, "path": filepath.ToSlash(projectDir)}
		projectInfo["cache"] = projectCacheState(projectDir)
		projectInfo["hera_addon"] = projectHeraAddonState(projectDir)
		result["project"] = projectInfo
		checks = append(checks, engineCheck("project", true, filepath.ToSlash(projectDir)))
		cache := projectInfo["cache"].(map[string]any)
		checks = append(checks, engineCheck("project_cache", cache["uid_cache"] == true,
			"uid_cache.bin present: import ran at least once"))
		addon := projectInfo["hera_addon"].(map[string]any)
		checks = append(checks, engineCheck("hera_addon", addon["listed_in_project_file"] == true,
			fmt.Sprint(addon["detail"])))
	}

	result["checks"] = checks
	result["warnings"] = warnings
	result["ok"] = true
	if code := printData(&protocol.Response{OK: true, Data: result}); code != 0 {
		return code
	}
	return 0
}

func engineCheck(name string, ok bool, detail string) map[string]any {
	return map[string]any{"name": name, "ok": ok, "detail": detail}
}

func editorBuildDetail(editor bool) string {
	if editor {
		return "editor build: editor-only flags are available"
	}
	return "template build: editor-only flags unavailable"
}

// engineTemplateDir resolves the export-template version directory for a
// version, mirroring Godot's "<major>.<minor>.<patch>.<status>[.mono]" layout
// across the editor data directories Godot can use.
func engineTemplateDir(version map[string]any, enginePath string) (root string, dir string, files int, err error) {
	raw, _ := version["string"].(string)
	parts := strings.Split(raw, ".")
	if len(parts) < 4 {
		return "", "", 0, errors.New("engine version string was not parsed")
	}
	name := strings.Join(parts[:4], ".")
	if version["mono"] == true {
		name += ".mono"
	}
	roots := engineTemplateRoots(enginePath)
	for _, candidate := range roots {
		if candidate == "" {
			continue
		}
		path := filepath.Join(candidate, name)
		info, statErr := os.Stat(path)
		if statErr != nil || !info.IsDir() {
			continue
		}
		entries, readErr := os.ReadDir(path)
		count := 0
		if readErr == nil {
			for _, entry := range entries {
				if !entry.IsDir() {
					count++
				}
			}
		}
		return filepath.ToSlash(candidate), filepath.ToSlash(path), count, nil
	}
	return filepath.ToSlash(firstNonEmpty(roots)), "", 0, nil
}

// engineTemplateRoots lists the directories Godot may resolve export templates
// from: the platform data directory, XDG_DATA_HOME on Unix, and the
// self-contained layout next to the executable.
func engineTemplateRoots(enginePath string) []string {
	roots := []string{}
	if dir, err := os.UserConfigDir(); err == nil {
		roots = append(roots, filepath.Join(dir, "Godot", "export_templates"))
	}
	if xdg := strings.TrimSpace(os.Getenv("XDG_DATA_HOME")); xdg != "" {
		roots = append(roots, filepath.Join(xdg, "godot", "export_templates"))
	}
	if home, err := os.UserHomeDir(); err == nil {
		roots = append(roots, filepath.Join(home, ".local", "share", "godot", "export_templates"))
	}
	// Godot's self-contained mode keeps editor data next to the executable.
	if enginePath != "" {
		exeDir := filepath.Dir(enginePath)
		if fileExists(filepath.Join(exeDir, "_sc_")) || runtime.GOOS == "windows" {
			roots = append(roots, filepath.Join(exeDir, "editor_data", "export_templates"))
		}
	}
	return roots
}

func firstNonEmpty(values []string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

// projectCacheState reports which generated caches exist in a project.
func projectCacheState(projectDir string) map[string]any {
	godot := filepath.Join(projectDir, ".godot")
	return map[string]any{
		"dir":          dirExists(godot),
		"uid_cache":    fileExists(filepath.Join(godot, "uid_cache.bin")),
		"imported":     dirExists(filepath.Join(godot, "imported")),
		"editor_state": dirExists(filepath.Join(godot, "editor")),
	}
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// projectHeraAddonState reports whether the Hera addon is installed and listed
// as an enabled editor plugin. The check reads project.godot only; it says
// nothing about a running editor.
func projectHeraAddonState(projectDir string) map[string]any {
	const addonPath = "addons/hera_agent_godot/plugin.cfg"
	installed := fileExists(filepath.Join(projectDir, filepath.FromSlash(addonPath)))
	state := map[string]any{"installed": installed, "listed_in_project_file": false}
	text, err := os.ReadFile(filepath.Join(projectDir, "project.godot"))
	if err != nil {
		state["detail"] = "project.godot unreadable: " + err.Error()
		return state
	}
	enabled := strings.Contains(string(text), "addons/hera_agent_godot/plugin.cfg")
	state["listed_in_project_file"] = enabled
	switch {
	case !installed:
		state["detail"] = "addon not installed in this project"
	case enabled:
		state["detail"] = "addon installed and enabled in project.godot"
	default:
		state["detail"] = "addon installed but not enabled in project.godot"
	}
	return state
}

// sortedBenchmarkEntries returns benchmark phases ordered by descending cost.
func sortedBenchmarkEntries(values map[string]float64) []map[string]any {
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		if values[names[i]] == values[names[j]] {
			return names[i] < names[j]
		}
		return values[names[i]] > values[names[j]]
	})
	entries := make([]map[string]any, 0, len(names))
	for _, name := range names {
		entries = append(entries, map[string]any{"name": name, "seconds": values[name]})
	}
	return entries
}
