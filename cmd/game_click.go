package cmd

import (
	"fmt"
	"strconv"
)

func parseGameClickArgs(args []string) (map[string]any, error) {
	params := map[string]any{"action": "click"}
	hasX := false
	hasY := false
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
		default:
			return nil, fmt.Errorf("unknown flag %q", args[i])
		}
	}
	if !hasX {
		return nil, fmt.Errorf("game click requires --x")
	}
	if !hasY {
		return nil, fmt.Errorf("game click requires --y")
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
