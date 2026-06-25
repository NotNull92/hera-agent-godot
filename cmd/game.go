package cmd

import (
	"fmt"
	"os"
)

func runGame(args []string) int {
	params, err := parseGameArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "game: %v\n", err)
		return 2
	}
	if gameActionMutates(params["action"]) {
		return dialMutationPostPrint("game", params, "game")
	}
	return dialPostPrint("game", params, "game")
}

func gameActionMutates(action any) bool {
	switch action {
	case "set", "call":
		return true
	default:
		return false
	}
}

func parseGameArgs(args []string) (map[string]any, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("usage: game <tree|node get|node set|node call> ...")
	}
	switch args[0] {
	case "tree":
		if len(args) != 1 {
			return nil, fmt.Errorf("game tree does not accept arguments")
		}
		return map[string]any{"action": "tree"}, nil
	case "node":
		return parseGameNodeArgs(args[1:])
	default:
		return nil, fmt.Errorf("unknown game subcommand %q (want tree|node get|node set|node call)", args[0])
	}
}

func parseGameNodeArgs(args []string) (map[string]any, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("usage: game node <get|set|call> ...")
	}
	switch args[0] {
	case "get":
		if len(args) != 2 {
			return nil, fmt.Errorf("usage: game node get <path>")
		}
		return map[string]any{"action": "get", "path": args[1]}, nil
	case "set":
		return parseGameNodeSetArgs(args[1:])
	case "call":
		return parseGameNodeCallArgs(args[1:])
	default:
		return nil, fmt.Errorf("unknown game node subcommand %q (want get|set|call)", args[0])
	}
}

func parseGameNodeSetArgs(args []string) (map[string]any, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("usage: game node set <path> --prop <name> --value <value>")
	}
	params := map[string]any{"action": "set", "path": args[0]}
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--prop":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--prop requires a value")
			}
			i++
			params["prop"] = args[i]
		case "--value":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--value requires a value")
			}
			i++
			params["value"] = args[i]
		default:
			return nil, fmt.Errorf("unknown flag %q", args[i])
		}
	}
	if _, ok := params["prop"]; !ok {
		return nil, fmt.Errorf("game node set requires --prop")
	}
	if _, ok := params["value"]; !ok {
		return nil, fmt.Errorf("game node set requires --value")
	}
	return params, nil
}

func parseGameNodeCallArgs(args []string) (map[string]any, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("usage: game node call <path> <method> [--arg <value> ...]")
	}
	params := map[string]any{"action": "call", "path": args[0], "method": args[1]}
	var callArgs []any
	for i := 2; i < len(args); i++ {
		switch args[i] {
		case "--arg":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--arg requires a value")
			}
			i++
			callArgs = append(callArgs, args[i])
		default:
			return nil, fmt.Errorf("unknown flag %q", args[i])
		}
	}
	if len(callArgs) > 0 {
		params["args"] = callArgs
	}
	return params, nil
}
