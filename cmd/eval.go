package cmd

import "fmt"

// runEval implements `hera-agent-godot eval <gdscript-expression>`.
//
// Maps to the addon `eval` tool, which uses GDScript's Expression evaluator.
//
// TODO(phase4).
func runEval(args []string) int {
	_ = args
	notImplemented("eval")
	return 1
}

// notImplemented is a shared placeholder for not-yet-built command handlers.
func notImplemented(name string) {
	fmt.Printf("hera-agent-godot: %q is not implemented yet (skeleton). See docs/ROADMAP.md\n", name)
}
