package cmd

import (
	"fmt"
	"strconv"
)

func parseGameClickArgs(args []string) (map[string]any, error) {
	params := map[string]any{"action": "click"}
	hasX := false
	hasY := false
	hasNode := false
	hasText := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--x":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--x requires a value")
			}
			i++
			x, err := parseScreenCoordinate(args[i], "--x")
			if err != nil {
				return nil, err
			}
			params["x"] = x
			hasX = true
		case "--y":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--y requires a value")
			}
			i++
			y, err := parseScreenCoordinate(args[i], "--y")
			if err != nil {
				return nil, err
			}
			params["y"] = y
			hasY = true
		case "--node":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--node requires a value")
			}
			i++
			if args[i] == "" {
				return nil, fmt.Errorf("--node requires a non-empty value")
			}
			params["path"] = normalizeGameNodePath(args[i])
			hasNode = true
		case "--text":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--text requires a value")
			}
			i++
			if args[i] == "" {
				return nil, fmt.Errorf("--text requires a non-empty value")
			}
			params["text"] = args[i]
			hasText = true
		default:
			return nil, fmt.Errorf("unknown flag %q", args[i])
		}
	}
	targetModes := 0
	if hasX || hasY {
		targetModes++
	}
	if hasNode {
		targetModes++
	}
	if hasText {
		targetModes++
	}
	if targetModes != 1 {
		return nil, fmt.Errorf("game click requires exactly one target: --x/--y, --node, or --text")
	}
	if hasX != hasY {
		return nil, fmt.Errorf("game click coordinates require both --x and --y")
	}
	return params, nil
}

func parseScreenCoordinate(raw string, flag string) (int, error) {
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return 0, fmt.Errorf("invalid %s %q (want a non-negative integer)", flag, raw)
	}
	return value, nil
}
