package cmd

import (
	"fmt"
	"os"
	"strconv"
)

func runProject(args []string) int {
	params, err := parseProjectArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "project: %v\n", err)
		return 2
	}
	if projectActionMutates(params["action"]) {
		return dialMutationPostPrint("project", params, "project")
	}
	return dialPostPrint("project", params, "project")
}

func projectActionMutates(action any) bool {
	return action == "mkdir"
}

func parseProjectArgs(args []string) (map[string]any, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("usage: project <info|list-files|mkdir> ...")
	}
	switch args[0] {
	case "info":
		if len(args) != 1 {
			return nil, fmt.Errorf("usage: project info")
		}
		return map[string]any{"action": "info"}, nil

	case "list-files":
		return parseProjectListFilesArgs(args[1:])

	case "mkdir":
		if len(args) != 2 {
			return nil, fmt.Errorf("usage: project mkdir <res://dir>")
		}
		return map[string]any{"action": "mkdir", "path": args[1]}, nil
	default:
		return nil, fmt.Errorf("unknown project subcommand %q (want info|list-files|mkdir)", args[0])
	}
}

func parseProjectListFilesArgs(args []string) (map[string]any, error) {
	params := map[string]any{"action": "list_files"}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--type":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--type requires a value")
			}
			i++
			if !validProjectFileType(args[i]) {
				return nil, fmt.Errorf("--type must be one of all|scene|script|resource|asset|shader")
			}
			params["type"] = args[i]
		case "--pattern":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--pattern requires a value")
			}
			i++
			params["pattern"] = args[i]
		case "--limit":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--limit requires a value")
			}
			i++
			limit, err := strconv.Atoi(args[i])
			if err != nil || limit <= 0 {
				return nil, fmt.Errorf("--limit must be a positive integer")
			}
			params["limit"] = limit
		default:
			return nil, fmt.Errorf("unknown flag %q", args[i])
		}
	}
	return params, nil
}

func validProjectFileType(value string) bool {
	switch value {
	case "all", "scene", "script", "resource", "asset", "shader":
		return true
	default:
		return false
	}
}
