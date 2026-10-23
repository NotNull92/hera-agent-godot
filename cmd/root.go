package cmd

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

// outputMode is set from leading global flags (--json / --ids) and consumed by
// printData. Empty = compact JSON (the default).
var outputMode string

// targetPID is set from the leading global flag --instance <pid>. 0 = unset, in
// which case commands target the most recent live editor. When set, every
// command targets that specific editor and the multi-editor mutation guard is
// satisfied (the user picked one explicitly). Consumed by selectEditor.
var targetPID int

// requestTimeout is set from the leading global flag --timeout <ms>. 0 = unset,
// in which case the client's 5 s default applies. It bounds each HTTP request,
// not a whole command (--wait polls send many requests). Consumed by
// dialEditorWithMode.
var requestTimeout time.Duration

const maxDurationMilliseconds = int64(time.Duration(1<<63-1) / time.Millisecond)

// Execute is the CLI entry point. Leading --json/--ids/--instance set global
// options; the first non-flag argument is the command. Commands map 1:1 to
// addon tools (see docs/COMMANDS.md).
type commandOptions struct {
	OutputMode     string
	TargetPID      int
	RequestTimeout time.Duration
	EnginePath     string
	CSharpHostPath string
}

func parseCommandOptions(args []string) (commandOptions, []string, error) {
	var options commandOptions
	start := 0
	for start < len(args) {
		arg := args[start]
		if isFlag(arg, "--csharp-host") {
			value, next, err := takeFlagValue(args, start, "--csharp-host")
			if err != nil {
				return options, nil, err
			}
			options.CSharpHostPath = value
			start = next
			continue
		}
		if arg == "--json" {
			options.OutputMode = "json"
			start++
			continue
		}
		if arg == "--ids" {
			options.OutputMode = "ids"
			start++
			continue
		}
		if arg == "--engine" || strings.HasPrefix(arg, "--engine=") {
			value, next, err := takeFlagValue(args, start, "--engine")
			if err != nil {
				return options, nil, err
			}
			options.EnginePath = value
			start = next
			continue
		}
		if arg == "--instance" {
			if start+1 >= len(args) {
				return options, nil, fmt.Errorf("--instance requires a pid argument")
			}
			pid, ok := parsePID(args[start+1])
			if !ok {
				return options, nil, fmt.Errorf("--instance: invalid pid %q", args[start+1])
			}
			options.TargetPID = pid
			start += 2
			continue
		}
		if v, ok := strings.CutPrefix(arg, "--instance="); ok {
			pid, ok := parsePID(v)
			if !ok {
				return options, nil, fmt.Errorf("--instance: invalid pid %q", v)
			}
			options.TargetPID = pid
			start++
			continue
		}
		if arg == "--timeout" {
			if start+1 >= len(args) {
				return options, nil, fmt.Errorf("--timeout requires a milliseconds argument")
			}
			timeout, ok := parseTimeoutMS(args[start+1])
			if !ok {
				return options, nil, fmt.Errorf("--timeout: invalid milliseconds %q", args[start+1])
			}
			options.RequestTimeout = timeout
			start += 2
			continue
		}
		if v, ok := strings.CutPrefix(arg, "--timeout="); ok {
			timeout, ok := parseTimeoutMS(v)
			if !ok {
				return options, nil, fmt.Errorf("--timeout: invalid milliseconds %q", v)
			}
			options.RequestTimeout = timeout
			start++
			continue
		}
		break
	}
	return options, args[start:], nil

}

func Execute(args []string) int {
	options, rest, err := parseCommandOptions(args)
	return runCommand(rest, options, err)
}

func runCommand(args []string, options commandOptions, parseErr error) int {
	// shortcut: legacy runners still consume globals; thread options through them before parallel command execution.
	outputMode = options.OutputMode
	targetPID = options.TargetPID
	requestTimeout = options.RequestTimeout
	engineFlagPath = options.EnginePath
	csharpHostPath = options.CSharpHostPath
	if parseErr != nil {
		fmt.Fprintln(os.Stderr, parseErr)
		return 2
	}

	if len(args) == 0 {
		usage()
		return 0
	}

	switch args[0] {
	case "csharp-host":
		return runCSharpHost(args[1:])
	case "status":
		return runStatus(args[1:])
	case "operation":
		return runOperation(args[1:])
	case "run":
		return runRun(args[1:])
	case "stop":
		return runStop(args[1:])
	case "scene":
		return runScene(args[1:])
	case "editor":
		return runEditor(args[1:])
	case "node":
		return runNode(args[1:])
	case "script":
		return runScript(args[1:])
	case "project":
		return runProject(args[1:])
	case "signal":
		return runSignal(args[1:])
	case "resource":
		return runResource(args[1:])
	case "theme":
		return runTheme(args[1:])
	case "classdb":
		return runClassDB(args[1:])
	case "game":
		return runGame(args[1:])
	case "instances":
		return runInstances(args[1:])
	case "engine":
		return runEngine(args[1:])
	case "capture":
		return runCapture(args[1:])
	case "perf":
		return runPerf(args[1:])
	case "docs":
		return runDocs(args[1:])
	case "debug":
		return runDebug(args[1:])
	case "eval":
		return runEval(args[1:])
	case "guidance":
		return runGuidance(args[1:])
	case "game_feel", "game-feel":
		return runGameFeel(args[1:])
	case "output":
		return runOutput(args[1:])
	case "diagnostics":
		return runDiagnostics(args[1:])
	case "screenshot":
		return runScreenshot(args[1:])
	case "batch":
		return runBatch(args[1:])
	case "smoke":
		return runSmoke(args[1:])
	case "version", "--version":
		fmt.Println(version)
		return 0
	case "help", "-h", "--help":
		usage()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %q\n\n", args[0])
		usageTo(os.Stderr)
		return 2
	}
}

func usage() {
	usageTo(os.Stdout)
}

func usageTo(w io.Writer) {
	fmt.Fprint(w, `hera-godot — drive a live Godot 4.x editor from the shell

usage: hera-godot [--json|--ids] [--instance <pid>] [--timeout <ms>] [--engine <path>] <command> [flags]

commands:
  status     show the connected editor (project, version, scene, session); --capabilities for API detail
  operation  submit <session:unix_ms:nonce> --request JSON | status|cancel <id> [--verbose]
  run        play the main / current / a specific scene  (--scene, --current, --wait)
  stop       stop the running scene
  scene      tree [path] [--depth N] [--type Class] [--props a,b] | list | open <res://...> | reload [res://...] | save | create [--open] | save-as
  editor     state | selected | select <node> [--add] | clear-selection
  node       find|get|add|instance|set|set-resource|remove|reparent|attach-script|detach-script
             add|instance ... [--set prop=value]...; get <path> [--changed|--prop P|--props a,b]; --snapshot with --prop; set <path> --prop P --value V [--expected JSON [--verify]]
  script     current | inspect|open <res://script.gd|.cs> | create <res://script.gd|.cs> [--lang gdscript|csharp] [--tool]
             validate <res://script.gd> [--evidence]  check fresh on-disk GDScript with the editor's engine
             run <res://tool.gd> [--quit-after N] [-- <args>...]  headless SceneTree harness, no editor needed
  project    info | list-files | scan | reimport <res://file>... | mkdir | set-main-scene
             import [--project DIR]  engine import pass, no editor needed
             export <preset> <output> [--kind pack|release|debug|patch] [--patches a,b] [--verbose]
  signal     list <node> [--connected] | connect|disconnect <from> <sig> <to> <method>
  resource   get|uid|list|set|create|resave|update-uids|export-mesh-library
  theme      get <res://t.tres> [--type T] | set <res://t.tres> --type T --color|--constant|--font-size name=value
  classdb    info|methods|properties|signals|constants|enums|inherits  ([--own] for signals/constants/enums)
             dump --out <dir>  offline ClassDB JSON from the engine (no editor needed)
  game       [--pid N] tree | ui tree|audit | instances | screenshot | click | input | clock | assert | qa discover|diagnose|--file | node get|set|call
                    assert <path> <prop> <op> [expected] --within-ms N (1..30000)
             input sequence --file <events.json>  replay actions on physics frames
  engine     info | doctor [--project DIR]  engine identity, build kind, export templates, project cache
  capture    movie <out.avi|.png|.ogv> [--frames N] [--fps N] [--resolution WxH]  deterministic frames
  perf       sample [--headless] [--top N]  engine benchmark phases as JSON
  docs       dump --out <dir> [--format xml|json] [--scripts] [--force]  engine or project API reference
  debug      attach [--break res://f.gd:N ...] [--dap-port N] [--wait-ms N] [--thread ID]  attach to a running editor debug server
             session | stack | variables | evaluate <expr> | continue | next | step-in | step-out | detach
             --thread ID selects a discovered thread for stack/variables/evaluate/resume/step
  instances  list live Hera-enabled Godot editors; expired heartbeats listed as stale
  csharp-host install [--version vX.Y.Z]  install the SHA256-checked C# compiler helper
  eval       [--lang csharp|gdscript] evaluate an expression (default C#; HERA_GODOT_EVAL_LANG override)
  guidance   ui | game-feel
  game_feel  list bundled Game Feel topics, or query one topic
  output     tail logs (--type log|error|warning|all, --lines N, --source file|editor, --since cursor)
  diagnostics summarize logs (--lines N, --source file|editor, --since cursor)
  screenshot render the edited scene to PNG (--path, --width, --height, --transparent, --runtime, --analyze)
             --evidence [--operation-id ID] [--runtime-session ID]  capture identity and actual size
             diff <before.png> <after.png> [--threshold N]  compare two captures locally
  batch      run a JSON array of {tool, params} (stdin or --file; --continue)
  smoke      run a live editor smoke check [--run-game|--skip-game]
  version    print the CLI version

scene save and resource set accept --evidence [--expected-sha256 HASH].
Save-call success, observed disk contents and memory equivalence are separate.

global flags (before the command):
  --json         pretty-print the response
  --ids          print only paths (scene tree / node find / game ui tree)
  --instance N   target the editor with pid N (required for mutations when
                 more than one editor is live; status output includes the pid)
  --timeout MS   per-request HTTP timeout in milliseconds (default 5000)
                 also bounds the engine process for script validate and every
                 engine-backed command (script run, project import/export,
                 capture movie, perf sample, docs dump, classdb dump)
  --csharp-host PATH  C# compiler helper; HERA_GODOT_CSHARP_HOST override
  --engine PATH  Godot engine executable for commands that do not need a live
                 editor; also read from HERA_GODOT_ENGINE or GODOT_BIN

Engine-backed commands run the real engine in a bounded child process and report
its exit code plus observed artifacts; engine stdout/stderr text is localized and
is never parsed for status. Overlapping mutations or sensitive reads fail
editor_busy; status and buffered editor logs remain available. A client timeout
does not unlock running work.
See docs/COMMANDS.md for details.
`)
}

// parsePID parses a positive process id. Returns ok=false for non-numeric or
// non-positive input.
func parsePID(s string) (int, bool) {
	pid, err := strconv.Atoi(s)
	if err != nil || pid <= 0 {
		return 0, false
	}
	return pid, true
}

// parseTimeoutMS parses a positive millisecond count. Returns ok=false for
// non-numeric or non-positive input.
func parseTimeoutMS(s string) (time.Duration, bool) {
	ms, err := strconv.ParseInt(s, 10, 64)
	if err != nil || ms <= 0 || ms > maxDurationMilliseconds {
		return 0, false
	}
	return time.Duration(ms) * time.Millisecond, true
}
