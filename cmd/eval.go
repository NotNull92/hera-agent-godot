package cmd

import (
	"fmt"
	"os"
	"strings"
)

// runEval implements `hera-agent-godot eval <expression>`.
//
// Joins its args into one GDScript expression and evaluates it in the editor via
// the addon's Expression-based `eval` tool.
func runEval(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "eval: requires an expression")
		return 2
	}
	expr := strings.Join(args, " ")
	return dialMutationPostPrint("eval", map[string]any{"expr": expr}, "eval")
}
