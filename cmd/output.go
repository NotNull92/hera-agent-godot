package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// runOutput implements `hera-godot output [--type log|error|warning|all] [--lines N]`.
func runOutput(args []string) int {
	params, err := parseOutputArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "output: %v\n", err)
		return 2
	}
	return dialPostPrint("output", params, "output")
}

func parseOutputArgs(args []string) (map[string]any, error) {
	params := map[string]any{}
	for i := 0; i < len(args); i++ {
		switch {
		case isFlag(args[i], "--type"):
			raw, next, err := takeFlagValue(args, i, "--type")
			if err != nil {
				return nil, err
			}
			i = next - 1
			switch raw {
			case "log", "error", "warning", "all":
			default:
				return nil, fmt.Errorf("invalid --type %q (want log|error|warning|all)", raw)
			}
			params["type"] = raw
		case isFlag(args[i], "--lines"):
			raw, next, err := takeFlagValue(args, i, "--lines")
			if err != nil {
				return nil, err
			}
			i = next - 1
			n, err := strconv.Atoi(raw)
			if err != nil || n <= 0 {
				return nil, fmt.Errorf("invalid --lines %q (want a positive integer)", raw)
			}
			params["lines"] = n
		case isFlag(args[i], "--source"), isFlag(args[i], "--since"):
			flag, _, _ := strings.Cut(args[i], "=")
			raw, next, err := takeFlagValue(args, i, flag)
			if err != nil || raw == "" {
				return nil, fmt.Errorf("%s requires a value", flag)
			}
			i = next - 1
			if flag == "--source" && raw != "file" && raw != "editor" {
				return nil, fmt.Errorf("invalid --source %q (want file|editor)", raw)
			}
			params[flag[2:]] = raw
		default:
			return nil, fmt.Errorf("unknown flag %q", args[i])
		}
	}
	if _, hasSince := params["since"]; hasSince && params["source"] != "editor" {
		return nil, fmt.Errorf("--since requires --source editor")
	}
	return params, nil
}
