package cmd

import (
	"fmt"
	"os"
	"strings"
)

func runScene(args []string) int {
	params, err := parseSceneArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "scene: %v\n", err)
		return 2
	}
	if sceneActionMutates(params["action"]) {
		return dialMutationPostPrint("scene", params, "scene")
	}
	return dialPostPrint("scene", params, "scene")
}

func sceneActionMutates(action any) bool {
	switch action {
	case "open", "reload", "save", "create", "save_as":
		return true
	default:
		return false
	}
}

func parseSceneArgs(args []string) (map[string]any, error) {
	if len(args) == 0 {
		return map[string]any{"action": "tree"}, nil
	}
	switch args[0] {
	case "tree":
		return parseSceneTreeArgs(args[1:])
	case "list":
		if len(args) > 1 {
			return nil, fmt.Errorf("scene list does not accept arguments")
		}
		return map[string]any{"action": "open_scenes"}, nil
	case "open":
		if len(args) != 2 {
			return nil, fmt.Errorf("usage: scene open <res://...>")
		}
		return map[string]any{"action": "open", "path": args[1]}, nil
	case "reload":
		if len(args) > 2 {
			return nil, fmt.Errorf("usage: scene reload [res://...]")
		}
		params := map[string]any{"action": "reload"}
		if len(args) == 2 {
			params["path"] = args[1]
		}
		return params, nil
	case "save":
		rest, params, err := parseEvidenceOptions(args[1:], false)
		if err != nil {
			return nil, err
		}
		if len(rest) > 0 {
			return nil, fmt.Errorf("scene save does not accept arguments")
		}
		params["action"] = "save"
		return params, nil
	case "create":
		return parseSceneCreateArgs(args[1:])
	case "save-as":
		return parseSceneSaveAsArgs(args[1:])
	default:
		return nil, fmt.Errorf("unknown scene subcommand %q (want tree|list|open|reload|save|create|save-as)", args[0])
	}
}

func parseSceneCreateArgs(args []string) (map[string]any, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("usage: scene create <res://...> [--root <type>] [--force] [--open]")
	}
	params := map[string]any{"action": "create", "path": args[0]}
	for i := 1; i < len(args); i++ {
		switch {
		case isFlag(args[i], "--root"):
			raw, next, err := takeFlagValue(args, i, "--root")
			if err != nil {
				return nil, err
			}
			i = next - 1
			params["root"] = raw
		case args[i] == "--force":
			params["force"] = true
		case args[i] == "--open":
			params["open"] = true
		default:
			return nil, fmt.Errorf("unknown flag %q", args[i])
		}
	}
	return params, nil
}

func parseSceneSaveAsArgs(args []string) (map[string]any, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("usage: scene save-as <res://...> [--force]")
	}
	params := map[string]any{"action": "save_as", "path": args[0]}
	for i := 1; i < len(args); i++ {
		switch {
		case args[i] == "--force":
			params["force"] = true
		default:
			return nil, fmt.Errorf("unknown flag %q", args[i])
		}
	}
	return params, nil
}

func parseSceneTreeArgs(args []string) (map[string]any, error) {
	params := map[string]any{"action": "tree"}
	for i := 0; i < len(args); i++ {
		switch {
		case isFlag(args[i], "--depth"):
			raw, next, err := takeFlagValue(args, i, "--depth")
			if err != nil {
				return nil, err
			}
			depth, err := parseNonNegativeInt(raw, "--depth")
			if err != nil {
				return nil, err
			}
			params["depth"] = depth
			i = next - 1
		case isFlag(args[i], "--type"):
			raw, next, err := takeFlagValue(args, i, "--type")
			if err != nil {
				return nil, err
			}
			if raw == "" {
				return nil, fmt.Errorf("--type requires a non-empty value")
			}
			params["type"] = raw
			i = next - 1
		case isFlag(args[i], "--props"):
			raw, next, err := takeFlagValue(args, i, "--props")
			if err != nil {
				return nil, err
			}
			props, err := parseCommaList(raw)
			if err != nil {
				return nil, err
			}
			params["props"] = props
			i = next - 1
		case strings.HasPrefix(args[i], "--"):
			return nil, fmt.Errorf("unknown flag %q", args[i])
		default:
			if params["path"] != nil {
				return nil, fmt.Errorf("unexpected argument %q", args[i])
			}
			params["path"] = args[i]
		}
	}
	return params, nil
}
