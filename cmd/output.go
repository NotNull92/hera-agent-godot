package cmd

import (
	"fmt"
	"os"
	"strconv"
)

// runOutput implements `hera output [--type log|error|warning|all] [--lines N]`.
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
		switch args[i] {
		case "--type":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--type requires a value")
			}
			i++
			switch args[i] {
			case "log", "error", "warning", "all":
			default:
				return nil, fmt.Errorf("invalid --type %q (want log|error|warning|all)", args[i])
			}
			params["type"] = args[i]
		case "--lines":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--lines requires a value")
			}
			i++
			n, err := strconv.Atoi(args[i])
			if err != nil || n <= 0 {
				return nil, fmt.Errorf("invalid --lines %q (want a positive integer)", args[i])
			}
			params["lines"] = n
		case "--source", "--since":
			flag := args[i]
			if i+1 >= len(args) || args[i+1] == "" {
				return nil, fmt.Errorf("%s requires a value", flag)
			}
			i++
			if flag == "--source" && args[i] != "file" && args[i] != "editor" {
				return nil, fmt.Errorf("invalid --source %q (want file|editor)", args[i])
			}
			params[flag[2:]] = args[i]
		default:
			return nil, fmt.Errorf("unknown flag %q", args[i])
		}
	}
	if _, hasSince := params["since"]; hasSince && params["source"] != "editor" {
		return nil, fmt.Errorf("--since requires --source editor")
	}
	return params, nil
}
