package cmd

import (
	"fmt"
	"os"
	"strings"
)

// runNode implements the `node` command (read + write).
//
//	find [query] [--type Class]              match nodes by name and/or class
//	get <path>                               dump a node's properties
//	add <type> [--parent <path>] [--name n]  add a node under a parent
//	set <path> --prop <name> --value <v>     set a node property
//	remove <path>                            remove a node
func runNode(args []string) int {
	params, err := parseNodeArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "node: %v\n", err)
		return 2
	}
	if nodeActionMutates(params["action"]) {
		return dialMutationPostPrint("node", params, "node")
	}
	return dialPostPrint("node", params, "node")
}

func nodeActionMutates(action any) bool {
	switch action {
	case "add", "instance", "set", "set_resource", "remove", "attach_script", "detach_script":
		return true
	default:
		return false
	}
}

func parseNodeArgs(args []string) (map[string]any, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("usage: node <find|get|add|instance|set|remove> ...")
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

	case "add":
		if len(rest) == 0 {
			return nil, fmt.Errorf("usage: node add <type> [--parent <path>] [--name <name>]")
		}
		params := map[string]any{"action": "add", "type": rest[0]}
		for i := 1; i < len(rest); i++ {
			switch rest[i] {
			case "--parent":
				if i+1 >= len(rest) {
					return nil, fmt.Errorf("--parent requires a value")
				}
				i++
				params["parent"] = rest[i]
			case "--name":
				if i+1 >= len(rest) {
					return nil, fmt.Errorf("--name requires a value")
				}
				i++
				params["name"] = rest[i]
			default:
				return nil, fmt.Errorf("unknown flag %q", rest[i])
			}
		}
		return params, nil

	case "instance":
		if len(rest) == 0 {
			return nil, fmt.Errorf("usage: node instance <res://scene.tscn> [--parent <path>] [--name <name>]")
		}
		params := map[string]any{"action": "instance", "scene": rest[0]}
		for i := 1; i < len(rest); i++ {
			switch rest[i] {
			case "--parent":
				if i+1 >= len(rest) {
					return nil, fmt.Errorf("--parent requires a value")
				}
				i++
				params["parent"] = rest[i]
			case "--name":
				if i+1 >= len(rest) {
					return nil, fmt.Errorf("--name requires a value")
				}
				i++
				params["name"] = rest[i]
			default:
				return nil, fmt.Errorf("unknown flag %q", rest[i])
			}
		}
		return params, nil

	case "set":
		if len(rest) == 0 {
			return nil, fmt.Errorf("usage: node set <path> --prop <name> --value <value>")
		}
		params := map[string]any{"action": "set", "path": rest[0]}
		for i := 1; i < len(rest); i++ {
			switch rest[i] {
			case "--prop":
				if i+1 >= len(rest) {
					return nil, fmt.Errorf("--prop requires a value")
				}
				i++
				params["prop"] = rest[i]
			case "--value":
				if i+1 >= len(rest) {
					return nil, fmt.Errorf("--value requires a value")
				}
				i++
				params["value"] = rest[i]
			default:
				return nil, fmt.Errorf("unknown flag %q", rest[i])
			}
		}
		if _, ok := params["prop"]; !ok {
			return nil, fmt.Errorf("node set requires --prop")
		}
		if _, ok := params["value"]; !ok {
			return nil, fmt.Errorf("node set requires --value")
		}
		return params, nil

	case "set-resource":
		if len(rest) == 0 {
			return nil, fmt.Errorf("usage: node set-resource <path> --prop <name> --resource <res://path>")
		}
		params := map[string]any{"action": "set_resource", "path": rest[0]}
		for i := 1; i < len(rest); i++ {
			switch rest[i] {
			case "--prop":
				if i+1 >= len(rest) {
					return nil, fmt.Errorf("--prop requires a value")
				}
				i++
				params["prop"] = rest[i]
			case "--resource":
				if i+1 >= len(rest) {
					return nil, fmt.Errorf("--resource requires a value")
				}
				i++
				params["resource"] = rest[i]
			default:
				return nil, fmt.Errorf("unknown flag %q", rest[i])
			}
		}
		if _, ok := params["prop"]; !ok {
			return nil, fmt.Errorf("node set-resource requires --prop")
		}
		if _, ok := params["resource"]; !ok {
			return nil, fmt.Errorf("node set-resource requires --resource")
		}
		return params, nil

	case "remove":
		if len(rest) != 1 {
			return nil, fmt.Errorf("usage: node remove <path>")
		}
		return map[string]any{"action": "remove", "path": rest[0]}, nil

	case "attach-script":
		if len(rest) != 2 {
			return nil, fmt.Errorf("usage: node attach-script <path> <res://script.gd>")
		}
		return map[string]any{"action": "attach_script", "path": rest[0], "script": rest[1]}, nil

	case "detach-script":
		if len(rest) != 1 {
			return nil, fmt.Errorf("usage: node detach-script <path>")
		}
		return map[string]any{"action": "detach_script", "path": rest[0]}, nil

	default:
		return nil, fmt.Errorf("unknown node subcommand %q (want find|get|add|instance|set|set-resource|remove|attach-script|detach-script)", sub)
	}
}
