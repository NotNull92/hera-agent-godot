package cmd

import (
	"fmt"
	"strconv"
)

const (
	defaultGameUIAuditLimit = 50
	maxGameUIAuditLimit     = 500
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

var allowedGameUIAuditSeverities = map[string]bool{
	"all":     true,
	"error":   true,
	"warning": true,
}

var allowedGameUIAuditRules = map[string]bool{
	"empty_interactive_rect":           true,
	"interactive_outside_viewport":     true,
	"fully_clipped_interactive":        true,
	"fullscreen_mouse_blocker":         true,
	"minimum_size_exceeds_parent":      true,
	"overlapping_interactive_siblings": true,
}

func parseGameUIArgs(args []string) (map[string]any, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("usage: game ui <tree|audit> ...")
	}
	switch args[0] {
	case "tree":
		return parseGameUITreeArgs(args[1:])
	case "audit":
		return parseGameUIAuditArgs(args[1:])
	default:
		return nil, fmt.Errorf("unknown game ui subcommand %q (want tree|audit)", args[0])
	}
}

func parseGameUITreeArgs(args []string) (map[string]any, error) {
	params := map[string]any{"action": "ui_tree"}
	for i := 0; i < len(args); i++ {
		switch {
		case isFlag(args[i], "--path"):
			raw, next, err := takeFlagValue(args, i, "--path")
			if err != nil {
				return nil, err
			}
			i = next - 1
			if raw == "" {
				return nil, fmt.Errorf("--path requires a non-empty value")
			}
			params["path"] = normalizeGameNodePath(raw)
		case isFlag(args[i], "--depth"):
			raw, next, err := takeFlagValue(args, i, "--depth")
			if err != nil {
				return nil, err
			}
			i = next - 1
			depth, err := parseNonNegativeInt(raw, "--depth")
			if err != nil {
				return nil, err
			}
			params["depth"] = depth
		case isFlag(args[i], "--fields"):
			raw, next, err := takeFlagValue(args, i, "--fields")
			if err != nil {
				return nil, err
			}
			i = next - 1
			fields, err := parseGameUIFields(raw)
			if err != nil {
				return nil, err
			}
			params["fields"] = fields
		case isFlag(args[i], "--type"):
			raw, next, err := takeFlagValue(args, i, "--type")
			if err != nil {
				return nil, err
			}
			i = next - 1
			if raw == "" {
				return nil, fmt.Errorf("--type requires a non-empty value")
			}
			params["type"] = raw
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
		default:
			return nil, fmt.Errorf("unknown flag %q", args[i])
		}
	}
	return params, nil
}

func parseGameUIAuditArgs(args []string) (map[string]any, error) {
	params := map[string]any{
		"action":   "ui_audit",
		"severity": "all",
		"strict":   false,
		"limit":    defaultGameUIAuditLimit,
	}
	for i := 0; i < len(args); i++ {
		switch {
		case isFlag(args[i], "--path"):
			value, err := requiredGameUIAuditValue(args, &i, "--path")
			if err != nil {
				return nil, err
			}
			params["path"] = normalizeGameNodePath(value)
		case isFlag(args[i], "--severity"):
			value, err := requiredGameUIAuditValue(args, &i, "--severity")
			if err != nil {
				return nil, err
			}
			if !allowedGameUIAuditSeverities[value] {
				return nil, fmt.Errorf("unknown UI audit severity %q (want all|error|warning)", value)
			}
			params["severity"] = value
		case isFlag(args[i], "--rule"):
			value, err := requiredGameUIAuditValue(args, &i, "--rule")
			if err != nil {
				return nil, err
			}
			if !allowedGameUIAuditRules[value] {
				return nil, fmt.Errorf("unknown UI audit rule %q", value)
			}
			params["rule"] = value
		case args[i] == "--strict":
			params["strict"] = true
		case isFlag(args[i], "--limit"):
			value, err := requiredGameUIAuditValue(args, &i, "--limit")
			if err != nil {
				return nil, err
			}
			limit, parseErr := strconv.Atoi(value)
			if parseErr != nil || limit < 1 || limit > maxGameUIAuditLimit {
				return nil, fmt.Errorf("--limit must be between 1 and %d", maxGameUIAuditLimit)
			}
			params["limit"] = limit
		default:
			return nil, fmt.Errorf("unknown flag %q", args[i])
		}
	}
	return params, nil
}

func requiredGameUIAuditValue(args []string, index *int, flag string) (string, error) {
	raw, next, err := takeFlagValue(args, *index, flag)
	if err != nil {
		return "", err
	}
	*index = next - 1
	value := raw
	if value == "" {
		return "", fmt.Errorf("%s requires a non-empty value", flag)
	}
	return value, nil
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
