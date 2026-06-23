package cmd

import (
	"fmt"
	"os"
	"strings"
)

// runNode implements `hera-agent-godot node <find [query] [--type Class] | get <path>>`.
//
//	find  match nodes by name substring and/or class
//	get   dump a node's editor-visible properties
func runNode(args []string) int {
	params, err := parseNodeArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "node: %v\n", err)
		return 2
	}
	return dialPostPrint("node", params, "node")
}

func parseNodeArgs(args []string) (map[string]any, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("usage: node <find|get> ...")
	}
	sub, rest := args[0], args[1:]
	switch sub {
	case "find":
		params := map[string]any{"action": "find"}
		for i := 0; i < len(rest); i++ {
			switch {
			case rest[i] == "--type":
				if i+1 >= len(rest) {
					return nil, fmt.Errorf("--type requires a value")
				}
				i++
				params["type"] = rest[i]
			case strings.HasPrefix(rest[i], "--"):
				return nil, fmt.Errorf("unknown flag %q", rest[i])
			default:
				if _, ok := params["query"]; ok {
					return nil, fmt.Errorf("unexpected argument %q", rest[i])
				}
				params["query"] = rest[i]
			}
		}
		return params, nil
	case "get":
		if len(rest) != 1 {
			return nil, fmt.Errorf("usage: node get <path>")
		}
		return map[string]any{"action": "get", "path": rest[0]}, nil
	default:
		return nil, fmt.Errorf("unknown node subcommand %q (want find|get)", sub)
	}
}
