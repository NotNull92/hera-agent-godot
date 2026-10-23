package cmd

import (
	"context"
	"errors"
	"fmt"
	"github.com/NotNull92/hera-agent-godot/internal/engineproc"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

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
	run := engineproc.Run(ctx, executable, []string{"--version"}, "", nil, 4096)
	if failure := runFailureCode(run); failure != "" {
		return engineVersion{}, fmt.Errorf("--version failed (%s), exit %d: %s", failure, run.ExitCode, firstLine(run.Output))
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
	run := engineproc.Run(ctx, executable, []string{"--help"}, "", nil, 256*1024)
	if runFailed(run) || !run.Started || run.Output == "" || run.Truncated ||
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
