package cmd

import (
	"fmt"
	"os"
)

func runScript(args []string) int {
	params, err := parseScriptArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "script: %v\n", err)
		return 2
	}
	return dialMutationPostPrint("script", params, "script")
}

func parseScriptArgs(args []string) (map[string]any, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("usage: script create <res://script.gd> [--extends <Class>] [--class-name <Name>] [--force]")
	}
	if args[0] != "create" {
		return nil, fmt.Errorf("unknown script subcommand %q (want create)", args[0])
	}
	if len(args) < 2 {
		return nil, fmt.Errorf("usage: script create <res://script.gd> [--extends <Class>] [--class-name <Name>] [--force]")
	}
	params := map[string]any{"action": "create", "path": args[1]}
	for i := 2; i < len(args); i++ {
		switch args[i] {
		case "--extends":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--extends requires a value")
			}
			i++
			params["extends"] = args[i]
		case "--class-name":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--class-name requires a value")
			}
			i++
			params["class_name"] = args[i]
		case "--force":
			params["force"] = true
		default:
			return nil, fmt.Errorf("unknown flag %q", args[i])
		}
	}
	return params, nil
}
