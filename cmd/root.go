package cmd

import "fmt"

// outputMode is set from leading global flags (--json / --ids) and consumed by
// printData. Empty = compact JSON (the default).
var outputMode string

// Execute is the CLI entry point. Leading --json/--ids set the output mode; the
// first non-flag argument is the command. Commands map 1:1 to addon tools (see
// docs/COMMANDS.md).
func Execute(args []string) int {
	outputMode = ""
	start := 0
	for start < len(args) {
		switch args[start] {
		case "--json":
			outputMode = "json"
			start++
			continue
		case "--ids":
			outputMode = "ids"
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
	case "eval":
		return runEval(args[1:])
	case "output":
		return runOutput(args[1:])
	case "screenshot":
		return runScreenshot(args[1:])
	case "batch":
		return runBatch(args[1:])
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

usage: hera-agent-godot [--json|--ids] <command> [flags]

commands:
  status     show the connected editor (project, version, active scene)
  run        play the main / current / a specific scene  (--scene, --current, --wait)
  stop       stop the running scene
  scene      tree | list | open <res://...> | save
  node       find|get|add|set|remove  (see docs/COMMANDS.md)
  eval       evaluate a GDScript expression in the editor
  output     tail project log (--type log|error|warning|all, --lines N)
  screenshot render the edited scene to PNG (--path, --width, --height, --transparent)
  batch      run a JSON array of {tool, params} (stdin or --file; --continue)

global flags (before the command):
  --json     pretty-print the response
  --ids      print only node paths (for scene tree / node find)

See docs/COMMANDS.md for details.
`)
}
