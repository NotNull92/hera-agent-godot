package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// outputMode is set from leading global flags (--json / --ids) and consumed by
// printData. Empty = compact JSON (the default).
var outputMode string

// targetPID is set from the leading global flag --instance <pid>. 0 = unset, in
// which case commands target the most recent live editor. When set, every
// command targets that specific editor and the multi-editor mutation guard is
// satisfied (the user picked one explicitly). Consumed by selectEditor.
var targetPID int

// Execute is the CLI entry point. Leading --json/--ids/--instance set global
// options; the first non-flag argument is the command. Commands map 1:1 to
// addon tools (see docs/COMMANDS.md).
func Execute(args []string) int {
	outputMode = ""
	targetPID = 0
	start := 0
	for start < len(args) {
		arg := args[start]
		if arg == "--json" {
			outputMode = "json"
			start++
			continue
		}
		if arg == "--ids" {
			outputMode = "ids"
			start++
			continue
		}
		if arg == "--instance" {
			if start+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "--instance requires a pid argument")
				return 2
			}
			pid, ok := parsePID(args[start+1])
			if !ok {
				fmt.Fprintf(os.Stderr, "--instance: invalid pid %q\n", args[start+1])
				return 2
			}
			targetPID = pid
			start += 2
			continue
		}
		if v, ok := strings.CutPrefix(arg, "--instance="); ok {
			pid, ok := parsePID(v)
			if !ok {
				fmt.Fprintf(os.Stderr, "--instance: invalid pid %q\n", v)
				return 2
			}
			targetPID = pid
			start++
			continue
		}
		break
	}
	args = args[start:]

	if len(args) == 0 {
		usage()
		return 0
	}

	switch args[0] {
	case "status":
		return runStatus(args[1:])
	case "run":
		return runRun(args[1:])
	case "stop":
		return runStop(args[1:])
	case "scene":
		return runScene(args[1:])
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
	case "classdb":
		return runClassDB(args[1:])
	case "game":
		return runGame(args[1:])
	case "instances":
		return runInstances(args[1:])
	case "eval":
		return runEval(args[1:])
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
		fmt.Printf("unknown command: %q\n\n", args[0])
		usage()
		return 2
	}
}

func usage() {
	fmt.Print(`hera-agent-godot — drive a live Godot 4.x editor from the shell

usage: hera-agent-godot [--json|--ids] [--instance <pid>] <command> [flags]

commands:
  status     show the connected editor (project, version, active scene)
  run        play the main / current / a specific scene  (--scene, --current, --wait)
  stop       stop the running scene
  scene      tree | list | open <res://...> | save | create [--open] | save-as
  node       find|get|add|set|set-resource|remove|attach-script|detach-script
  script     create <res://script.gd> [--extends Class] [--class-name Name] [--force]
  project    info | list-files | mkdir <res://dir> | set-main-scene <res://scene.tscn>
  signal     list <node> | connect|disconnect <from> <sig> <to> <method>
  resource   get|uid|resave|update-uids|export-mesh-library
  classdb    info|methods|properties|inherits
  game       tree | instances | screenshot | assert | qa | node get|set|call
  instances  list live Hera-enabled Godot editors
  eval       evaluate a GDScript expression in the editor
  output     tail project log (--type log|error|warning|all, --lines N)
  diagnostics summarize project log errors and warnings (--lines N)
  screenshot render the edited scene to PNG (--path, --width, --height, --transparent, --runtime, --analyze)
  batch      run a JSON array of {tool, params} (stdin or --file; --continue)
  smoke      run a live editor smoke check [--run-game|--skip-game]
  version    print the CLI version

global flags (before the command):
  --json         pretty-print the response
  --ids          print only node paths (for scene tree / node find)
  --instance N   target the editor with pid N (required for mutations when
                 more than one editor is live; status output includes the pid)

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
