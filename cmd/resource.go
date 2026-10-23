package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"
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
	case "set", "create", "resave", "update_uids", "export_mesh_library":
		return true
	default:
		return false
	}
}

func parseResourceArgs(args []string) (map[string]any, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("usage: resource <get|uid|list|set|create|resave|update-uids|export-mesh-library> ...")
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
	case "list":
		return parseResourceListArgs(rest)
	case "set":
		return parseResourceSetArgs(rest)
	case "resave":
		if len(rest) != 1 {
			return nil, fmt.Errorf("usage: resource resave <res://path>")
		}
		return map[string]any{"action": "resave", "path": rest[0]}, nil
	case "create":
		return parseResourceCreateArgs(rest)
	case "update-uids":
		if len(rest) != 0 {
			return nil, fmt.Errorf("usage: resource update-uids")
		}
		return map[string]any{"action": "update_uids"}, nil
	case "export-mesh-library":
		return parseExportMeshLibraryArgs(rest)
	default:
		return nil, fmt.Errorf("unknown resource subcommand %q (want get|uid|list|set|create|resave|update-uids|export-mesh-library)", sub)
	}
}

func parseResourceListArgs(args []string) (map[string]any, error) {
	params := map[string]any{"action": "list", "path": "res://"}
	start := 0
	if len(args) > 0 && !strings.HasPrefix(args[0], "--") {
		params["path"] = args[0]
		start = 1
	}
	for i := start; i < len(args); i++ {
		switch {
		case isFlag(args[i], "--type"):
			raw, next, err := takeFlagValue(args, i, "--type")
			if err != nil {
				return nil, fmt.Errorf("--type requires a class name")
			}
			i = next - 1
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
				return nil, fmt.Errorf("--limit requires a number")
			}
			i = next - 1
			limit, err := strconv.Atoi(raw)
			if err != nil || limit <= 0 {
				return nil, fmt.Errorf("--limit requires a positive number")
			}
			params["limit"] = limit
		default:
			return nil, fmt.Errorf("unknown flag %q", args[i])
		}
	}
	return params, nil
}

func parseResourceSetArgs(args []string) (map[string]any, error) {
	args, params, err := parseEvidenceOptions(args, false)
	if err != nil {
		return nil, err
	}
	if len(args) < 2 || (len(args) == 2 && !strings.HasPrefix(args[1], "--prop=")) {
		return nil, fmt.Errorf("usage: resource set <res://path> --prop <name=value> ...")
	}
	props, err := parseResourceProps(args[1:], "resource set <res://path> --prop <name=value> ...", false)
	if err != nil {
		return nil, err
	}
	if len(props) == 0 {
		return nil, fmt.Errorf("resource set requires at least one --prop")
	}
	params["action"], params["path"], params["props"] = "set", args[0], props
	return params, nil
}

func parseResourceCreateArgs(args []string) (map[string]any, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("usage: resource create <Class> <res://out.tres> [--force] [--prop <name=value> ...]")
	}
	params := map[string]any{"action": "create", "type": args[0], "path": args[1]}
	props, err := parseResourceProps(args[2:], "resource create <Class> <res://out.tres> [--force] [--prop <name=value> ...]", true)
	if err != nil {
		return nil, err
	}
	for i := 2; i < len(args); i++ {
		switch {
		case args[i] == "--force":
			params["force"] = true
		case isFlag(args[i], "--prop"):
			_, next, err := takeFlagValue(args, i, "--prop")
			if err != nil {
				return nil, fmt.Errorf("--prop requires a name=value")
			}
			i = next - 1
		default:
			return nil, fmt.Errorf("unknown flag %q", args[i])
		}
	}
	if len(props) > 0 {
		params["props"] = props
	}
	return params, nil
}

func parseResourceProps(args []string, usage string, allowForce bool) (map[string]string, error) {
	props := map[string]string{}
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--force":
			if allowForce {
				continue
			}
			return nil, fmt.Errorf("unknown flag %q", args[i])
		case isFlag(args[i], "--prop"):
			raw, next, err := takeFlagValue(args, i, "--prop")
			if err != nil {
				return nil, fmt.Errorf("--prop requires a name=value")
			}
			i = next - 1
			name, value, ok := strings.Cut(raw, "=")
			if !ok || name == "" {
				return nil, fmt.Errorf("--prop requires a name=value")
			}
			props[name] = value
		default:
			return nil, fmt.Errorf("usage: %s", usage)
		}
	}
	return props, nil
}

func parseExportMeshLibraryArgs(args []string) (map[string]any, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("usage: resource export-mesh-library <res://scene.tscn> <res://out.tres> [--item <name> ...]")
	}
	params := map[string]any{"action": "export_mesh_library", "path": args[0], "output": args[1]}
	var items []string
	for i := 2; i < len(args); i++ {
		switch {
		case isFlag(args[i], "--item"):
			raw, next, err := takeFlagValue(args, i, "--item")
			if err != nil {
				return nil, err
			}
			i = next - 1
			items = append(items, raw)
		default:
			return nil, fmt.Errorf("unknown flag %q", args[i])
		}
	}
	if len(items) > 0 {
		params["items"] = items
	}
	return params, nil
}
