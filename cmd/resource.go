package cmd

import (
	"fmt"
	"os"
)

// runResource implements the `resource` command (read-only).
//
//	get <res://path>   dump a resource's class, name, and properties
func runResource(args []string) int {
	params, err := parseResourceArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "resource: %v\n", err)
		return 2
	}
	return dialPostPrint("resource", params, "resource")
}

func parseResourceArgs(args []string) (map[string]any, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("usage: resource get <res://path>")
	}
	sub, rest := args[0], args[1:]
	switch sub {
	case "get":
		if len(rest) != 1 {
			return nil, fmt.Errorf("usage: resource get <res://path>")
		}
		return map[string]any{"action": "get", "path": rest[0]}, nil
	default:
		return nil, fmt.Errorf("unknown resource subcommand %q (want get)", sub)
	}
}
