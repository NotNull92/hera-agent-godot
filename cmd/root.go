package cmd

import "fmt"

// Execute is the CLI entry point. It dispatches the first argument to the
// matching command handler. Commands map 1:1 to addon tools (see
// docs/COMMANDS.md).
//
// TODO(phase1): replace this hand-rolled dispatch with a real command framework
// once flags/subcommands grow. Kept dependency-free for the skeleton.
func Execute(args []string) int {
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

usage: hera-agent-godot <command> [flags]

commands:
  status     show the connected editor (project, version, active scene)
  run        play the main / current / a specific scene  (--scene, --current, --wait)
  stop       stop the running scene
  scene      tree | list
  node       find [query] [--type Class] | get <path>
  eval       evaluate a GDScript expression in the editor
  output     tail project log (--type log|error|warning|all, --lines N)

See docs/COMMANDS.md for details.
`)
}
