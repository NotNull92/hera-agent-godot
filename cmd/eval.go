package cmd

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
