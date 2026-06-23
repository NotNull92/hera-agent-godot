package cmd

import "fmt"

// runOutput implements `hera-agent-godot output [--type log|error|warning]`.
//
// Maps to the addon `output` tool (editor output log + error/warning entries).
//
// TODO(phase3).
func runOutput(args []string) int {
	_ = args
	notImplemented("output")
	return 1
}

// notImplemented is a shared placeholder for skeleton command handlers.
func notImplemented(name string) {
	fmt.Printf("hera-agent-godot: %q is not implemented yet (skeleton). See docs/ROADMAP.md\n", name)
}
