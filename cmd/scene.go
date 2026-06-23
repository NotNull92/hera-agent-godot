package cmd

import (
	"fmt"
	"os"
)

// runScene implements `hera-agent-godot scene <tree|list>`.
//
//	tree  describe the edited scene's node tree
//	list  list open scenes and the current one
func runScene(args []string) int {
	params, err := parseSceneArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "scene: %v\n", err)
		return 2
	}
	return dialPostPrint("scene", params, "scene")
}

func parseSceneArgs(args []string) (map[string]any, error) {
	sub := "tree"
	if len(args) > 0 {
		sub = args[0]
	}
	if len(args) > 1 {
		return nil, fmt.Errorf("scene %s does not accept arguments", sub)
	}
	switch sub {
	case "tree":
		return map[string]any{"action": "tree"}, nil
	case "list":
		return map[string]any{"action": "open_scenes"}, nil
	default:
		return nil, fmt.Errorf("unknown scene subcommand %q (want tree|list)", sub)
	}
}
