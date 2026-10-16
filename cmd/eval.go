package cmd

import (
	"fmt"
	"os"
	"strings"
)

// runEval implements `hera-godot eval <expression>`.
//
// Joins its args into one expression in the selected language.
func runEval(args []string) int {
	params, err := parseEvalArgs(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, "eval:", err)
		return 2
	}
	return dialMutationPostPrint("eval", params, "eval")
}

func evalLanguage(explicit string) (string, error) {
	if explicit == "" {
		explicit = os.Getenv("HERA_GODOT_EVAL_LANG")
	}
	if explicit == "" {
		explicit = "csharp"
	}
	if explicit != "csharp" && explicit != "gdscript" {
		return "", fmt.Errorf("--lang / HERA_GODOT_EVAL_LANG must be csharp or gdscript")
	}
	return explicit, nil
}

func parseEvalArgs(args []string) (map[string]any, error) {
	lang := ""
	expr := []string{}
	for i := 0; i < len(args); i++ {
		if args[i] == "--" {
			expr = append(expr, args[i+1:]...)
			break
		}
		if isFlag(args[i], "--lang") {
			value, next, err := takeFlagValue(args, i, "--lang")
			if err != nil {
				return nil, err
			}
			lang = value
			i = next - 1
			continue
		}
		expr = append(expr, args[i])
	}
	if len(expr) == 0 {
		return nil, fmt.Errorf("requires an expression")
	}
	lang, err := evalLanguage(lang)
	if err != nil {
		return nil, err
	}
	return map[string]any{"expr": strings.Join(expr, " "), "lang": lang}, nil
}
