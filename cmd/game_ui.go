package cmd

import "fmt"

func parseGameUIArgs(args []string) (map[string]any, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("usage: game ui tree")
	}
	switch args[0] {
	case "tree":
		if len(args) != 1 {
			return nil, fmt.Errorf("game ui tree does not accept arguments")
		}
		return map[string]any{"action": "ui_tree"}, nil
	default:
		return nil, fmt.Errorf("unknown game ui subcommand %q (want tree)", args[0])
	}
}
