package cmd

import (
	"fmt"
	"os"
)

func runProject(args []string) int {
	params, err := parseProjectArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "project: %v\n", err)
		return 2
	}
	return dialMutationPostPrint("project", params, "project")
}

func parseProjectArgs(args []string) (map[string]any, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("usage: project mkdir <res://dir>")
	}
	switch args[0] {
	case "mkdir":
		if len(args) != 2 {
			return nil, fmt.Errorf("usage: project mkdir <res://dir>")
		}
		return map[string]any{"action": "mkdir", "path": args[1]}, nil
	default:
		return nil, fmt.Errorf("unknown project subcommand %q (want mkdir)", args[0])
	}
}
