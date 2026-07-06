package cmd

import (
	"fmt"
	"strconv"
)

var allowedGameUIFields = map[string]bool{
	"path":     true,
	"type":     true,
	"name":     true,
	"visible":  true,
	"rect":     true,
	"text":     true,
	"disabled": true,
	"pressed":  true,
}

func parseGameUIArgs(args []string) (map[string]any, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("usage: game ui tree [--path <node>] [--depth N] [--fields a,b] [--type Class] [--text text]")
	}
	switch args[0] {
	case "tree":
		return parseGameUITreeArgs(args[1:])
	default:
		return nil, fmt.Errorf("unknown game ui subcommand %q (want tree)", args[0])
	}
}

func parseGameUITreeArgs(args []string) (map[string]any, error) {
	params := map[string]any{"action": "ui_tree"}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--path":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--path requires a value")
			}
			i++
			if args[i] == "" {
				return nil, fmt.Errorf("--path requires a non-empty value")
			}
			params["path"] = normalizeGameNodePath(args[i])
		case "--depth":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--depth requires a value")
			}
			i++
			depth, err := parseNonNegativeInt(args[i], "--depth")
			if err != nil {
				return nil, err
			}
			params["depth"] = depth
		case "--fields":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--fields requires a value")
			}
			i++
			fields, err := parseGameUIFields(args[i])
			if err != nil {
				return nil, err
			}
			params["fields"] = fields
		case "--type":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--type requires a value")
			}
			i++
			if args[i] == "" {
				return nil, fmt.Errorf("--type requires a non-empty value")
			}
			params["type"] = args[i]
		case "--text":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--text requires a value")
			}
			i++
			if args[i] == "" {
				return nil, fmt.Errorf("--text requires a non-empty value")
			}
			params["text"] = args[i]
		default:
			return nil, fmt.Errorf("unknown flag %q", args[i])
		}
	}
	return params, nil
}

func parseNonNegativeInt(raw string, flag string) (int, error) {
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return 0, fmt.Errorf("invalid %s %q (want a non-negative integer)", flag, raw)
	}
	return value, nil
}

func parseGameUIFields(raw string) ([]string, error) {
	fields, err := parseCommaList(raw)
	if err != nil {
		return nil, err
	}
	for _, field := range fields {
		if !allowedGameUIFields[field] {
			return nil, fmt.Errorf("unknown ui field %q", field)
		}
	}
	return fields, nil
}
