package cmd

import (
	"fmt"
	"strconv"
	"strings"
)

func parseGameInputArgs(args []string) (map[string]any, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("usage: game input <mouse|key|action|text|joypad|axis|sequence> ...")
	}
	switch args[0] {
	case "sequence":
		return parseGameInputSequenceArgs(args[1:])
	case "mouse":
		return parseGameInputMouseArgs(args[1:])
	case "key":
		return parseGameInputKeyArgs(args[1:])
	case "action":
		return parseGameInputActionArgs(args[1:])
	case "text":
		return parseGameInputTextArgs(args[1:])
	case "joypad":
		return parseGameInputJoypadArgs(args[1:])
	case "axis":
		return parseGameInputAxisArgs(args[1:])
	default:
		return nil, fmt.Errorf("unknown game input kind %q (want mouse|key|action|text|joypad|axis|sequence)", args[0])
	}
}

func parseGameInputMouseArgs(args []string) (map[string]any, error) {
	params := map[string]any{"action": "input", "kind": "mouse", "button": "left"}
	hasX := false
	hasY := false
	mode := ""
	for i := 0; i < len(args); i++ {
		switch {
		case isFlag(args[i], "--x"):
			value, err := parseRequiredScreenCoordinate(args, &i, "--x")
			if err != nil {
				return nil, err
			}
			params["x"] = value
			hasX = true
		case isFlag(args[i], "--y"):
			value, err := parseRequiredScreenCoordinate(args, &i, "--y")
			if err != nil {
				return nil, err
			}
			params["y"] = value
			hasY = true
		case isFlag(args[i], "--dx"):
			value, err := parseRequiredInt(args, &i, "--dx")
			if err != nil {
				return nil, err
			}
			params["dx"] = value
		case isFlag(args[i], "--dy"):
			value, err := parseRequiredInt(args, &i, "--dy")
			if err != nil {
				return nil, err
			}
			params["dy"] = value
		case isFlag(args[i], "--button"):
			raw, next, err := takeFlagValue(args, i, "--button")
			if err != nil {
				return nil, err
			}
			i = next - 1
			button := strings.ToLower(raw)
			if !validMouseButton(button) {
				return nil, fmt.Errorf("unknown mouse button %q (want left|right|middle|wheel_up|wheel_down)", raw)
			}
			params["button"] = button
		case args[i] == "--press", args[i] == "--release", args[i] == "--click", args[i] == "--move":
			nextMode := strings.TrimPrefix(args[i], "--")
			if mode != "" {
				return nil, fmt.Errorf("game input mouse accepts exactly one of --press, --release, --click, or --move")
			}
			mode = nextMode
			params["mode"] = nextMode
		case args[i] == "--double":
			params["double"] = true
		case isFlag(args[i], "--modifiers"):
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
		switch {
		case isFlag(args[i], "--key"):
			raw, next, err := takeFlagValue(args, i, "--key")
			if err != nil {
				return nil, err
			}
			i = next - 1
			if raw == "" {
				return nil, fmt.Errorf("--key requires a non-empty value")
			}
			params["key"] = raw
			hasKey = true
		case isFlag(args[i], "--keycode"):
			value, err := parseRequiredInt(args, &i, "--keycode")
			if err != nil {
				return nil, err
			}
			if value <= 0 {
				return nil, fmt.Errorf("--keycode must be positive")
			}
			params["keycode"] = value
			hasKey = true
		case isFlag(args[i], "--unicode"):
			value, err := parseRequiredInt(args, &i, "--unicode")
			if err != nil {
				return nil, err
			}
			if value < 0 {
				return nil, fmt.Errorf("--unicode must be non-negative")
			}
			params["unicode"] = value
		case args[i] == "--press", args[i] == "--release":
			nextMode := strings.TrimPrefix(args[i], "--")
			if mode != "" {
				return nil, fmt.Errorf("game input key accepts exactly one of --press or --release")
			}
			mode = nextMode
			params["mode"] = nextMode
		case args[i] == "--physical":
			params["physical"] = true
		case isFlag(args[i], "--route"):
			raw, next, err := takeFlagValue(args, i, "--route")
			if err != nil {
				return nil, err
			}
			i = next - 1
			if raw != "input" && raw != "viewport" {
				return nil, fmt.Errorf("unknown key route %q (want input|viewport)", raw)
			}
			params["route"] = raw
		case isFlag(args[i], "--modifiers"):
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
		switch {
		case args[i] == "--press", args[i] == "--release":
			nextMode := strings.TrimPrefix(args[i], "--")
			if mode != "" {
				return nil, fmt.Errorf("game input action accepts exactly one of --press or --release")
			}
			mode = nextMode
			params["mode"] = nextMode
		case isFlag(args[i], "--strength"):
			raw, next, err := takeFlagValue(args, i, "--strength")
			if err != nil {
				return nil, err
			}
			i = next - 1
			strength, err := strconv.ParseFloat(raw, 64)
			if err != nil || strength < 0.0 || strength > 1.0 {
				return nil, fmt.Errorf("invalid --strength %q (want 0..1)", raw)
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

func parseGameInputJoypadArgs(args []string) (map[string]any, error) {
	params := map[string]any{"action": "input", "kind": "joypad"}
	mode := ""
	hasButton := false
	for i := 0; i < len(args); i++ {
		switch {
		case isFlag(args[i], "--button"):
			raw, next, err := takeFlagValue(args, i, "--button")
			if err != nil || raw == "" {
				return nil, fmt.Errorf("--button requires a value")
			}
			i = next - 1
			params["button"] = raw
			hasButton = true
		case args[i] == "--press", args[i] == "--release":
			nextMode := strings.TrimPrefix(args[i], "--")
			if mode != "" {
				return nil, fmt.Errorf("game input joypad accepts exactly one of --press or --release")
			}
			mode = nextMode
			params["mode"] = nextMode
		case isFlag(args[i], "--device"):
			value, err := parseRequiredInt(args, &i, "--device")
			if err != nil {
				return nil, err
			}
			if value < 0 {
				return nil, fmt.Errorf("--device must be non-negative")
			}
			params["device"] = value
		default:
			return nil, fmt.Errorf("unknown flag %q", args[i])
		}
	}
	if !hasButton {
		return nil, fmt.Errorf("game input joypad requires --button")
	}
	if mode == "" {
		return nil, fmt.Errorf("game input joypad requires --press or --release")
	}
	return params, nil
}

func parseGameInputAxisArgs(args []string) (map[string]any, error) {
	params := map[string]any{"action": "input", "kind": "axis"}
	hasAxis := false
	hasValue := false
	for i := 0; i < len(args); i++ {
		switch {
		case isFlag(args[i], "--axis"):
			raw, next, err := takeFlagValue(args, i, "--axis")
			if err != nil || raw == "" {
				return nil, fmt.Errorf("--axis requires a value")
			}
			i = next - 1
			params["axis"] = raw
			hasAxis = true
		case isFlag(args[i], "--value"):
			raw, next, err := takeFlagValue(args, i, "--value")
			if err != nil {
				return nil, err
			}
			i = next - 1
			value, err := strconv.ParseFloat(raw, 64)
			if err != nil || value < -1.0 || value > 1.0 || value != value {
				return nil, fmt.Errorf("invalid --value %q (want -1..1)", raw)
			}
			params["value"] = value
			hasValue = true
		case isFlag(args[i], "--device"):
			dev, err := parseRequiredInt(args, &i, "--device")
			if err != nil {
				return nil, err
			}
			if dev < 0 {
				return nil, fmt.Errorf("--device must be non-negative")
			}
			params["device"] = dev
		default:
			return nil, fmt.Errorf("unknown flag %q", args[i])
		}
	}
	if !hasAxis {
		return nil, fmt.Errorf("game input axis requires --axis")
	}
	if !hasValue {
		return nil, fmt.Errorf("game input axis requires --value")
	}
	return params, nil
}

func parseGameInputLogArgs(args []string) (map[string]any, error) {
	params := map[string]any{"action": "input_log"}
	for i := 0; i < len(args); i++ {
		switch {
		case isFlag(args[i], "--limit"):
			value, err := parseRequiredInt(args, &i, "--limit")
			if err != nil {
				return nil, err
			}
			if value < 0 {
				return nil, fmt.Errorf("--limit must be non-negative")
			}
			params["limit"] = value
		case args[i] == "--clear":
			params["clear"] = true
		default:
			return nil, fmt.Errorf("unknown flag %q", args[i])
		}
	}
	return params, nil
}

func parseRequiredScreenCoordinate(args []string, index *int, flag string) (int, error) {
	raw, next, err := takeFlagValue(args, *index, flag)
	if err != nil {
		return 0, err
	}
	*index = next - 1
	return parseScreenCoordinate(raw, flag)
}

func parseRequiredInt(args []string, index *int, flag string) (int, error) {
	raw, next, err := takeFlagValue(args, *index, flag)
	if err != nil {
		return 0, err
	}
	*index = next - 1
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid %s %q (want an integer)", flag, raw)
	}
	return value, nil
}

func parseRequiredModifiers(args []string, index *int) ([]string, error) {
	raw, next, err := takeFlagValue(args, *index, "--modifiers")
	if err != nil {
		return nil, err
	}
	*index = next - 1
	modifiers, err := parseCommaList(raw)
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
