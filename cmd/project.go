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
	if params["action"] == "import" {
		return runProjectImport(params)
	}
	if params["action"] == "export" {
		return runProjectExport(params)
	}
	if projectActionMutates(params["action"]) {
		return dialMutationPostPrint("project", params, "project")
	}
	return dialPostPrint("project", params, "project")
}

func projectActionMutates(action any) bool {
	switch action {
	case "mkdir", "set_main_scene", "scan", "reimport":
		return true
	default:
		return false
	}
}

func parseProjectArgs(args []string) (map[string]any, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("usage: project <info|list-files|scan|reimport|mkdir|set-main-scene|import|export> ...")
	}
	switch args[0] {
	case "import":
		return parseProjectImportArgs(args[1:])
	case "export":
		return parseProjectExportArgs(args[1:])
	case "info":
		if len(args) != 1 {
			return nil, fmt.Errorf("usage: project info")
		}
		return map[string]any{"action": "info"}, nil

	case "list-files":
		return parseProjectListFilesArgs(args[1:])

	case "scan":
		if len(args) != 1 {
			return nil, fmt.Errorf("usage: project scan")
		}
		return map[string]any{"action": "scan"}, nil
	case "reimport":
		return parseProjectReimportArgs(args[1:])
	case "mkdir":
		if len(args) != 2 {
			return nil, fmt.Errorf("usage: project mkdir <res://dir>")
		}
		return map[string]any{"action": "mkdir", "path": args[1]}, nil
	case "set-main-scene":
		if len(args) != 2 {
			return nil, fmt.Errorf("usage: project set-main-scene <res://scene.tscn>")
		}
		return map[string]any{"action": "set_main_scene", "path": args[1]}, nil
	default:
		return nil, fmt.Errorf("unknown project subcommand %q (want info|list-files|scan|reimport|mkdir|set-main-scene|import|export)", args[0])
	}
}

func parseProjectReimportArgs(args []string) (map[string]any, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("usage: project reimport <res://file> ...")
	}
	paths := make([]string, 0, len(args))
	paths = append(paths, args...)
	return map[string]any{"action": "reimport", "paths": paths}, nil
}

func parseProjectListFilesArgs(args []string) (map[string]any, error) {
	params := map[string]any{"action": "list_files"}
	for i := 0; i < len(args); i++ {
		switch {
		case isFlag(args[i], "--type"):
			raw, next, err := takeFlagValue(args, i, "--type")
			if err != nil {
				return nil, err
			}
			i = next - 1
			if !validProjectFileType(raw) {
				return nil, fmt.Errorf("--type must be one of all|scene|script|resource|asset|shader")
			}
			params["type"] = raw
		case isFlag(args[i], "--pattern"):
			raw, next, err := takeFlagValue(args, i, "--pattern")
			if err != nil {
				return nil, err
			}
			i = next - 1
			params["pattern"] = raw
		case isFlag(args[i], "--limit"):
			raw, next, err := takeFlagValue(args, i, "--limit")
			if err != nil {
				return nil, err
			}
			i = next - 1
			limit, err := strconv.Atoi(raw)
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
