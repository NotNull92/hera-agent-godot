package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func runGame(args []string) int {
	if len(args) > 0 && args[0] == "qa" {
		if len(args) > 1 && args[1] == "discover" {
			return runGameQADiscover(args[2:])
		}
		return runGameQA(args[1:])
	}
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

func parseGameArgs(args []string) (map[string]any, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("usage: game <tree|ui tree|instances|screenshot|click|input|input-log|assert|node get|node set|node call|qa> ...")
	}
	switch args[0] {
	case "tree":
		if len(args) != 1 {
			return nil, fmt.Errorf("game tree does not accept arguments")
		}
		return map[string]any{"action": "tree"}, nil
	case "ui":
		return parseGameUIArgs(args[1:])
	case "instances":
		if len(args) != 1 {
			return nil, fmt.Errorf("game instances does not accept arguments")
		}
		return map[string]any{"action": "instances"}, nil
	case "screenshot":
		return parseGameScreenshotArgs(args[1:])
	case "click":
		return parseGameClickArgs(args[1:])
	case "input":
		return parseGameInputArgs(args[1:])
	case "input-log":
		return parseGameInputLogArgs(args[1:])
	case "assert":
		return parseGameAssertArgs(args[1:])
	case "node":
		return parseGameNodeArgs(args[1:])
	default:
		return nil, fmt.Errorf("unknown game subcommand %q (want tree|ui tree|instances|screenshot|click|input|input-log|assert|node get|node set|node call|qa)", args[0])
	}
}

func parseGameNodeArgs(args []string) (map[string]any, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("usage: game node <get|set|call> ...")
	}
	switch args[0] {
	case "get":
		return parseGameNodeGetArgs(args[1:])
	case "set":
		return parseGameNodeSetArgs(args[1:])
	case "call":
		return parseGameNodeCallArgs(args[1:])
	default:
		return nil, fmt.Errorf("unknown game node subcommand %q (want get|set|call)", args[0])
	}
}

func parseGameNodeGetArgs(args []string) (map[string]any, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("usage: game node get <path> [--prop <name>|--props <a,b>]")
	}
	params := map[string]any{"action": "get", "path": normalizeGameNodePath(args[0])}
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--prop":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--prop requires a value")
			}
			i++
			if _, ok := params["props"]; ok {
				return nil, fmt.Errorf("use either --prop or --props, not both")
			}
			params["prop"] = args[i]
		case "--props":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--props requires a value")
			}
			i++
			if _, ok := params["prop"]; ok {
				return nil, fmt.Errorf("use either --prop or --props, not both")
			}
			props, err := parseCommaList(args[i])
			if err != nil {
				return nil, err
			}
			params["props"] = props
		default:
			return nil, fmt.Errorf("unknown flag %q", args[i])
		}
	}
	return params, nil
}

func parseGameNodeSetArgs(args []string) (map[string]any, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("usage: game node set <path> --prop <name> --value <value>")
	}
	params := map[string]any{"action": "set", "path": normalizeGameNodePath(args[0])}
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
	params := map[string]any{"action": "call", "path": normalizeGameNodePath(args[0]), "method": args[1]}
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

func parseGameScreenshotArgs(args []string) (map[string]any, error) {
	params := map[string]any{"action": "screenshot"}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--path":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--path requires a value")
			}
			i++
			params["path"] = args[i]
		case "--analyze":
			params["analyze"] = true
		default:
			return nil, fmt.Errorf("unknown flag %q", args[i])
		}
	}
	return params, nil
}

func parseGameAssertArgs(args []string) (map[string]any, error) {
	if len(args) < 3 {
		return nil, fmt.Errorf("usage: game assert <path> <prop> <eq|ne|contains|gt|lt|exists> [value]")
	}
	op := args[2]
	if !validGameAssertOp(op) {
		return nil, fmt.Errorf("unknown assert op %q (want eq|ne|contains|gt|lt|exists)", op)
	}
	if op == "exists" {
		if len(args) != 3 {
			return nil, fmt.Errorf("game assert exists does not accept a value")
		}
		return map[string]any{
			"action": "assert",
			"path":   normalizeGameNodePath(args[0]),
			"prop":   args[1],
			"op":     op,
		}, nil
	}
	if len(args) != 4 {
		return nil, fmt.Errorf("game assert %s requires a value", op)
	}
	return map[string]any{
		"action": "assert",
		"path":   normalizeGameNodePath(args[0]),
		"prop":   args[1],
		"op":     op,
		"value":  args[3],
	}, nil
}

func validGameAssertOp(op string) bool {
	switch op {
	case "eq", "ne", "contains", "gt", "lt", "exists":
		return true
	default:
		return false
	}
}

func normalizeGameNodePath(path string) string {
	cleaned := strings.ReplaceAll(path, "\\", "/")
	marker := "/Git/root/"
	_, suffix, found := strings.Cut(cleaned, marker)
	if !found {
		return path
	}
	return "/root/" + suffix
}

func parseCommaList(raw string) ([]string, error) {
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value == "" {
			return nil, fmt.Errorf("empty value in comma list %q", raw)
		}
		values = append(values, value)
	}
	return values, nil
}

func cloneJSONMap(values map[string]any) map[string]any {
	cloned := make(map[string]any, len(values))
	for key, value := range values {
		switch typed := value.(type) {
		case json.Number:
			cloned[key] = typed.String()
		default:
			cloned[key] = value
		}
	}
	return cloned
}
