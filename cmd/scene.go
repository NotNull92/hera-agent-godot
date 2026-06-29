package cmd

import (
	"fmt"
	"os"
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
		if len(args) > 1 {
			return nil, fmt.Errorf("scene tree does not accept arguments")
		}
		return map[string]any{"action": "tree"}, nil
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
		if len(args) > 1 {
			return nil, fmt.Errorf("scene save does not accept arguments")
		}
		return map[string]any{"action": "save"}, nil
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
		switch args[i] {
		case "--root":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--root requires a value")
			}
			i++
			params["root"] = args[i]
		case "--force":
			params["force"] = true
		case "--open":
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
		switch args[i] {
		case "--force":
			params["force"] = true
		default:
			return nil, fmt.Errorf("unknown flag %q", args[i])
		}
	}
	return params, nil
}
