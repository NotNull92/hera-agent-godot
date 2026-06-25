package cmd

import (
	"fmt"
	"os"
)

func runResource(args []string) int {
	params, err := parseResourceArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "resource: %v\n", err)
		return 2
	}
	if resourceActionMutates(params["action"]) {
		return dialMutationPostPrint("resource", params, "resource")
	}
	return dialPostPrint("resource", params, "resource")
}

func resourceActionMutates(action any) bool {
	switch action {
	case "resave", "update_uids", "export_mesh_library":
		return true
	default:
		return false
	}
}

func parseResourceArgs(args []string) (map[string]any, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("usage: resource <get|uid|resave|update-uids|export-mesh-library> ...")
	}
	sub, rest := args[0], args[1:]
	switch sub {
	case "get":
		if len(rest) != 1 {
			return nil, fmt.Errorf("usage: resource get <res://path>")
		}
		return map[string]any{"action": "get", "path": rest[0]}, nil
	case "uid":
		if len(rest) != 1 {
			return nil, fmt.Errorf("usage: resource uid <res://path>")
		}
		return map[string]any{"action": "uid", "path": rest[0]}, nil
	case "resave":
		if len(rest) != 1 {
			return nil, fmt.Errorf("usage: resource resave <res://path>")
		}
		return map[string]any{"action": "resave", "path": rest[0]}, nil
	case "update-uids":
		if len(rest) != 0 {
			return nil, fmt.Errorf("usage: resource update-uids")
		}
		return map[string]any{"action": "update_uids"}, nil
	case "export-mesh-library":
		return parseExportMeshLibraryArgs(rest)
	default:
		return nil, fmt.Errorf("unknown resource subcommand %q (want get|uid|resave|update-uids|export-mesh-library)", sub)
	}
}

func parseExportMeshLibraryArgs(args []string) (map[string]any, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("usage: resource export-mesh-library <res://scene.tscn> <res://out.tres> [--item <name> ...]")
	}
	params := map[string]any{"action": "export_mesh_library", "path": args[0], "output": args[1]}
	var items []string
	for i := 2; i < len(args); i++ {
		switch args[i] {
		case "--item":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--item requires a value")
			}
			i++
			items = append(items, args[i])
		default:
			return nil, fmt.Errorf("unknown flag %q", args[i])
		}
	}
	if len(items) > 0 {
		params["items"] = items
	}
	return params, nil
}
