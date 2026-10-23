package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func argsChanged(params map[string]any, key string) bool {
	value, ok := params[key+"_set"].(bool)
	return ok && value
}

func parseDebugArgs(args []string, allowBreakpoints bool) (map[string]any, error) {
	return parseDebugArgsWithExpression(args, allowBreakpoints, false)
}

// parseDebugArgsWithExpression parses shared debug flags. Only debug evaluate
// accepts the `--` expression terminator.
func parseDebugArgsWithExpression(args []string, allowBreakpoints bool, allowExpression bool) (map[string]any, error) {
	params := map[string]any{"host": "127.0.0.1", "port": defaultDAPPort, "wait_ms": defaultStopWaitMS, "frame": 0}
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--":
			if !allowExpression {
				return nil, fmt.Errorf("-- is only accepted by debug evaluate")
			}
			if _, seen := params["expression"]; seen {
				return nil, fmt.Errorf("only one expression is accepted")
			}
			rest := args[i+1:]
			if len(rest) != 1 {
				return nil, fmt.Errorf("-- must be followed by exactly one expression")
			}
			params["expression"] = rest[0]
			return params, nil
		case args[i] == "--break" || strings.HasPrefix(args[i], "--break="):
			if !allowBreakpoints {
				return nil, fmt.Errorf("--break is only accepted by debug attach")
			}
			value, next, err := takeFlagValue(args, i, "--break")
			if err != nil {
				return nil, err
			}
			path, line, err := parseBreakpoint(value)
			if err != nil {
				return nil, err
			}
			existing, _ := params["breaks"].([]sessionBreakpoint)
			params["breaks"] = append(existing, sessionBreakpoint{Path: path, Line: line})
			i = next - 1
		case isFlag(args[i], "--session"):
			value, next, err := takeFlagValue(args, i, "--session")
			if err != nil {
				return nil, err
			}
			if !validSessionID(value) {
				return nil, fmt.Errorf("--session must be a session id from debug session")
			}
			params["session"] = value
			i = next - 1
		case isFlag(args[i], "--host"):
			value, next, err := takeFlagValue(args, i, "--host")
			if err != nil {
				return nil, err
			}
			if value == "" {
				return nil, fmt.Errorf("--host requires a value")
			}
			params["host"], params["host_set"] = value, true
			i = next - 1
		case isFlag(args[i], "--dap-port"):
			value, next, err := takeFlagValue(args, i, "--dap-port")
			if err != nil {
				return nil, err
			}
			port, convErr := strconv.Atoi(value)
			if convErr != nil || port <= 0 || port > 65535 {
				return nil, fmt.Errorf("--dap-port must be a port number")
			}
			params["port"], params["port_set"] = port, true
			i = next - 1
		case isFlag(args[i], "--wait-ms"):
			value, next, err := takeFlagValue(args, i, "--wait-ms")
			if err != nil {
				return nil, err
			}
			millis, convErr := strconv.Atoi(value)
			if convErr != nil || millis <= 0 || millis > maxWaitMS {
				return nil, fmt.Errorf("--wait-ms must be between 1 and %d", maxWaitMS)
			}
			params["wait_ms"], params["wait_ms_set"] = millis, true
			i = next - 1
		case isFlag(args[i], "--thread"):
			value, next, err := takeFlagValue(args, i, "--thread")
			if err != nil {
				return nil, err
			}
			id, err := strconv.Atoi(value)
			if err != nil || id <= 0 || id > maxDebugThreadID {
				return nil, fmt.Errorf("--thread must be a positive thread id")
			}
			params["thread"], params["thread_set"] = id, true
			i = next - 1
		case isFlag(args[i], "--frame"):
			value, next, err := takeFlagValue(args, i, "--frame")
			if err != nil {
				return nil, err
			}
			frame, convErr := strconv.Atoi(value)
			if convErr != nil || frame < 0 {
				return nil, fmt.Errorf("--frame must be a non-negative integer")
			}
			params["frame"], params["frame_set"] = frame, true
			i = next - 1
		case isFlag(args[i], "--project"):
			value, next, err := takeFlagValue(args, i, "--project")
			if err != nil {
				return nil, err
			}
			params["project"] = value
			i = next - 1
		default:
			return nil, fmt.Errorf("unknown flag %q", args[i])
		}
	}
	return params, nil
}

// parseBreakpoint accepts res://script.gd:LINE and refuses escaping paths.
func parseBreakpoint(value string) (string, int, error) {
	index := strings.LastIndex(value, ":")
	if index <= 0 {
		return "", 0, fmt.Errorf("--break must look like res://script.gd:12")
	}
	path, lineText := value[:index], value[index+1:]
	line, err := strconv.Atoi(lineText)
	if err != nil || line <= 0 {
		return "", 0, fmt.Errorf("--break line must be a positive integer")
	}
	if !strings.HasPrefix(path, "res://") {
		return "", 0, fmt.Errorf("--break path must start with res://")
	}
	trimmed := strings.TrimPrefix(path, "res://")
	if strings.HasPrefix(trimmed, "/") || strings.Contains(trimmed, "..") {
		return "", 0, fmt.Errorf("--break path must stay inside the project")
	}
	return path, line, nil
}

// resolveBreakpointPath maps a res:// path into the project, refusing escapes
// after normalisation and after resolving links, and requires a readable script.
func resolveBreakpointPath(project, scriptPath string) (string, error) {
	if !strings.HasPrefix(scriptPath, "res://") {
		return scriptPath, nil
	}
	if project == "" {
		return "", fmt.Errorf("a res:// breakpoint needs --project so the script can be hashed")
	}
	resolved := filepath.Join(project, filepath.FromSlash(strings.TrimPrefix(scriptPath, "res://")))
	relative, err := filepath.Rel(project, resolved)
	if err != nil || strings.HasPrefix(relative, "..") {
		return "", fmt.Errorf("breakpoint %s escapes the project directory", scriptPath)
	}
	// A link inside the project must not point outside it.
	realProject, projectErr := filepath.EvalSymlinks(project)
	realScript, scriptErr := filepath.EvalSymlinks(resolved)
	if projectErr == nil && scriptErr == nil {
		realRelative, relErr := filepath.Rel(realProject, realScript)
		if relErr != nil || strings.HasPrefix(realRelative, "..") {
			return "", fmt.Errorf("breakpoint %s resolves outside the project directory", scriptPath)
		}
	}
	if _, readErr := os.ReadFile(resolved); readErr != nil {
		return "", fmt.Errorf("breakpoint script %s is not readable: %w", scriptPath, readErr)
	}
	return resolved, nil
}

func sessionWait(params map[string]any) time.Duration {
	if millis, ok := params["wait_ms"].(int); ok && millis > 0 {
		return time.Duration(millis) * time.Millisecond
	}
	return defaultStopWaitMS * time.Millisecond
}
