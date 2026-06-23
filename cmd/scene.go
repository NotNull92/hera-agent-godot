package cmd

import (
	"fmt"
	"os"
)

// runScene implements `hera-agent-godot scene <tree|list|open|save>`.
//
//	tree              describe the edited scene's node tree
//	list              list open scenes and the current one
//	open <res://...>  open a scene in the editor
//	save              save the edited scene
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
	case "open", "save":
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
	case "save":
		if len(args) > 1 {
			return nil, fmt.Errorf("scene save does not accept arguments")
		}
		return map[string]any{"action": "save"}, nil
	default:
		return nil, fmt.Errorf("unknown scene subcommand %q (want tree|list|open|save)", args[0])
	}
}
