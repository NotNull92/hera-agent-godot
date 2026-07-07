package cmd

import (
	"fmt"
	"strconv"
	"strings"
)

func parseGameInputArgs(args []string) (map[string]any, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("usage: game input <mouse|key|action|text> ...")
	}
	switch args[0] {
	case "mouse":
		return parseGameInputMouseArgs(args[1:])
	case "key":
		return parseGameInputKeyArgs(args[1:])
	case "action":
		return parseGameInputActionArgs(args[1:])
	case "text":
		return parseGameInputTextArgs(args[1:])
	default:
		return nil, fmt.Errorf("unknown game input kind %q (want mouse|key|action|text)", args[0])
	}
}

func parseGameInputMouseArgs(args []string) (map[string]any, error) {
	params := map[string]any{"action": "input", "kind": "mouse", "button": "left"}
	hasX := false
	hasY := false
	mode := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--x":
			value, err := parseRequiredScreenCoordinate(args, &i, "--x")
			if err != nil {
				return nil, err
			}
			params["x"] = value
			hasX = true
		case "--y":
			value, err := parseRequiredScreenCoordinate(args, &i, "--y")
			if err != nil {
				return nil, err
			}
			params["y"] = value
			hasY = true
		case "--dx":
			value, err := parseRequiredInt(args, &i, "--dx")
			if err != nil {
				return nil, err
			}
			params["dx"] = value
		case "--dy":
			value, err := parseRequiredInt(args, &i, "--dy")
			if err != nil {
				return nil, err
			}
			params["dy"] = value
		case "--button":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--button requires a value")
			}
			i++
			button := strings.ToLower(args[i])
			if !validMouseButton(button) {
				return nil, fmt.Errorf("unknown mouse button %q (want left|right|middle|wheel_up|wheel_down)", args[i])
			}
			params["button"] = button
		case "--press", "--release", "--click", "--move":
			nextMode := strings.TrimPrefix(args[i], "--")
			if mode != "" {
				return nil, fmt.Errorf("game input mouse accepts exactly one of --press, --release, --click, or --move")
			}
			mode = nextMode
			params["mode"] = nextMode
		case "--double":
			params["double"] = true
		case "--modifiers":
			modifiers, err := parseRequiredModifiers(args, &i)
			if err != nil {
				return nil, err
			}
			params["modifiers"] = modifiers
		default:
			return nil, fmt.Errorf("unknown flag %q", args[i])
		}
	}
	if !hasX || !hasY {
		return nil, fmt.Errorf("game input mouse requires --x and --y")
	}
	if mode == "" {
		return nil, fmt.Errorf("game input mouse requires one of --press, --release, --click, or --move")
	}
	return params, nil
}

func parseGameInputKeyArgs(args []string) (map[string]any, error) {
	params := map[string]any{"action": "input", "kind": "key"}
	mode := ""
	hasKey := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--key":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--key requires a value")
			}
			i++
			if args[i] == "" {
				return nil, fmt.Errorf("--key requires a non-empty value")
			}
			params["key"] = args[i]
			hasKey = true
		case "--keycode":
			value, err := parseRequiredInt(args, &i, "--keycode")
			if err != nil {
				return nil, err
			}
			if value <= 0 {
				return nil, fmt.Errorf("--keycode must be positive")
			}
			params["keycode"] = value
			hasKey = true
		case "--unicode":
			value, err := parseRequiredInt(args, &i, "--unicode")
			if err != nil {
				return nil, err
			}
			if value < 0 {
				return nil, fmt.Errorf("--unicode must be non-negative")
			}
			params["unicode"] = value
		case "--press", "--release":
			nextMode := strings.TrimPrefix(args[i], "--")
			if mode != "" {
				return nil, fmt.Errorf("game input key accepts exactly one of --press or --release")
			}
			mode = nextMode
			params["mode"] = nextMode
		case "--physical":
			params["physical"] = true
		case "--route":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--route requires a value")
			}
			i++
			if args[i] != "input" && args[i] != "viewport" {
				return nil, fmt.Errorf("unknown key route %q (want input|viewport)", args[i])
			}
			params["route"] = args[i]
		case "--modifiers":
			modifiers, err := parseRequiredModifiers(args, &i)
			if err != nil {
				return nil, err
			}
			params["modifiers"] = modifiers
		default:
			return nil, fmt.Errorf("unknown flag %q", args[i])
		}
	}
	if !hasKey {
		return nil, fmt.Errorf("game input key requires --key or --keycode")
	}
	if mode == "" {
		return nil, fmt.Errorf("game input key requires --press or --release")
	}
	return params, nil
}

func parseGameInputActionArgs(args []string) (map[string]any, error) {
	if len(args) == 0 || strings.HasPrefix(args[0], "--") {
		return nil, fmt.Errorf("usage: game input action <name> --press|--release [--strength <0..1>]")
	}
	params := map[string]any{"action": "input", "kind": "action", "name": args[0]}
	mode := ""
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--press", "--release":
			nextMode := strings.TrimPrefix(args[i], "--")
			if mode != "" {
				return nil, fmt.Errorf("game input action accepts exactly one of --press or --release")
			}
			mode = nextMode
			params["mode"] = nextMode
		case "--strength":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--strength requires a value")
			}
			i++
			strength, err := strconv.ParseFloat(args[i], 64)
			if err != nil || strength < 0.0 || strength > 1.0 {
				return nil, fmt.Errorf("invalid --strength %q (want 0..1)", args[i])
			}
			params["strength"] = strength
		default:
			return nil, fmt.Errorf("unknown flag %q", args[i])
		}
	}
	if mode == "" {
		return nil, fmt.Errorf("game input action requires --press or --release")
	}
	return params, nil
}

func parseGameInputTextArgs(args []string) (map[string]any, error) {
	if len(args) != 1 || args[0] == "" {
		return nil, fmt.Errorf("usage: game input text <text>")
	}
	return map[string]any{"action": "input", "kind": "text", "text": args[0]}, nil
}

func parseGameInputLogArgs(args []string) (map[string]any, error) {
	params := map[string]any{"action": "input_log"}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--limit":
			value, err := parseRequiredInt(args, &i, "--limit")
			if err != nil {
				return nil, err
			}
			if value < 0 {
				return nil, fmt.Errorf("--limit must be non-negative")
			}
			params["limit"] = value
		case "--clear":
			params["clear"] = true
		default:
			return nil, fmt.Errorf("unknown flag %q", args[i])
		}
	}
	return params, nil
}

func parseRequiredScreenCoordinate(args []string, index *int, flag string) (int, error) {
	if *index+1 >= len(args) {
		return 0, fmt.Errorf("%s requires a value", flag)
	}
	*index = *index + 1
	return parseScreenCoordinate(args[*index], flag)
}

func parseRequiredInt(args []string, index *int, flag string) (int, error) {
	if *index+1 >= len(args) {
		return 0, fmt.Errorf("%s requires a value", flag)
	}
	*index = *index + 1
	value, err := strconv.Atoi(args[*index])
	if err != nil {
		return 0, fmt.Errorf("invalid %s %q (want an integer)", flag, args[*index])
	}
	return value, nil
}

func parseRequiredModifiers(args []string, index *int) ([]string, error) {
	if *index+1 >= len(args) {
		return nil, fmt.Errorf("--modifiers requires a value")
	}
	*index = *index + 1
	modifiers, err := parseCommaList(args[*index])
	if err != nil {
		return nil, err
	}
	for _, modifier := range modifiers {
		if !validInputModifier(modifier) {
			return nil, fmt.Errorf("unknown modifier %q (want shift|ctrl|alt|meta)", modifier)
		}
	}
	return modifiers, nil
}

func validMouseButton(button string) bool {
	switch button {
	case "left", "right", "middle", "wheel_up", "wheel_down":
		return true
	default:
		return false
	}
}

func validInputModifier(modifier string) bool {
	switch strings.ToLower(modifier) {
	case "shift", "ctrl", "alt", "meta":
		return true
	default:
		return false
	}
}
