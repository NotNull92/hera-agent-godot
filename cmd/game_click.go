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
		switch {
		case isFlag(args[i], "--x"):
			raw, next, err := takeFlagValue(args, i, "--x")
			if err != nil {
				return nil, err
			}
			i = next - 1
			x, err := parseScreenCoordinate(raw, "--x")
			if err != nil {
				return nil, err
			}
			params["x"] = x
			hasX = true
		case isFlag(args[i], "--y"):
			raw, next, err := takeFlagValue(args, i, "--y")
			if err != nil {
				return nil, err
			}
			i = next - 1
			y, err := parseScreenCoordinate(raw, "--y")
			if err != nil {
				return nil, err
			}
			params["y"] = y
			hasY = true
		case isFlag(args[i], "--node"):
			raw, next, err := takeFlagValue(args, i, "--node")
			if err != nil {
				return nil, err
			}
			i = next - 1
			if raw == "" {
				return nil, fmt.Errorf("--node requires a non-empty value")
			}
			params["path"] = normalizeGameNodePath(raw)
			hasNode = true
		case isFlag(args[i], "--text"):
			raw, next, err := takeFlagValue(args, i, "--text")
			if err != nil {
				return nil, err
			}
			i = next - 1
			if raw == "" {
				return nil, fmt.Errorf("--text requires a non-empty value")
			}
			params["text"] = raw
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
