package cmd

import (
	"fmt"
	"os"
	"strconv"
)

func runScript(args []string) int {
	params, err := parseScriptArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "script: %v\n", err)
		return 2
	}
	if scriptActionMutates(params["action"]) {
		return dialMutationPostPrint("script", params, "script")
	}
	return dialPostPrint("script", params, "script")
}

func scriptActionMutates(action any) bool {
	switch action {
	case "create", "open":
		return true
	default:
		return false
	}
}

func parseScriptArgs(args []string) (map[string]any, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("usage: script <current|inspect|open|create> ...")
	}
	sub, rest := args[0], args[1:]
	switch sub {
	case "current":
		if len(rest) != 0 {
			return nil, fmt.Errorf("usage: script current")
		}
		return map[string]any{"action": "current"}, nil
	case "inspect":
		if len(rest) != 1 {
			return nil, fmt.Errorf("usage: script inspect <res://script.gd>")
		}
		return map[string]any{"action": "inspect", "path": rest[0]}, nil
	case "open":
		return parseScriptOpenArgs(rest)
	case "create":
		return parseScriptCreateArgs(rest)
	default:
		return nil, fmt.Errorf("unknown script subcommand %q (want current|inspect|open|create)", sub)
	}
}

func parseScriptOpenArgs(args []string) (map[string]any, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("usage: script open <res://script.gd> [--line N] [--column N]")
	}
	params := map[string]any{"action": "open", "path": args[0]}
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--line":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--line requires a number")
			}
			i++
			line, err := strconv.Atoi(args[i])
			if err != nil || line <= 0 {
				return nil, fmt.Errorf("--line requires a positive number")
			}
			params["line"] = line
		case "--column":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--column requires a number")
			}
			i++
			column, err := strconv.Atoi(args[i])
			if err != nil || column <= 0 {
				return nil, fmt.Errorf("--column requires a positive number")
			}
			params["column"] = column
		default:
			return nil, fmt.Errorf("unknown flag %q", args[i])
		}
	}
	return params, nil
}

func parseScriptCreateArgs(args []string) (map[string]any, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("usage: script create <res://script.gd> [--extends <Class>] [--class-name <Name>] [--force] [--tool] [--ready] [--process] [--physics-process] [--input] [--unhandled-input] [--signal <name> ...] [--export <name:type[=value]> ...]")
	}
	params := map[string]any{"action": "create", "path": args[0]}
	var signals []string
	var exports []string
	for i := 1; i < len(args); i++ {
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
		case "--tool":
			params["tool"] = true
		case "--ready":
			params["ready"] = true
		case "--process":
			params["process"] = true
		case "--physics-process":
			params["physics_process"] = true
		case "--input":
			params["input"] = true
		case "--unhandled-input":
			params["unhandled_input"] = true
		case "--signal":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--signal requires a name")
			}
			i++
			signals = append(signals, args[i])
		case "--export":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--export requires a name:type[=value]")
			}
			i++
			exports = append(exports, args[i])
		default:
			return nil, fmt.Errorf("unknown flag %q", args[i])
		}
	}
	if len(signals) > 0 {
		params["signals"] = signals
	}
	if len(exports) > 0 {
		params["exports"] = exports
	}
	return params, nil
}
