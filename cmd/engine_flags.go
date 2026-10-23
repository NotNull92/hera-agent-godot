package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// engineContextTimeout bounds one engine-backed command. --timeout (global)
// overrides the per-command default.
func engineContextTimeout(fallback time.Duration) (context.Context, context.CancelFunc) {
	timeout := requestTimeout
	if timeout == 0 {
		timeout = fallback
	}
	return context.WithTimeout(context.Background(), timeout)
}

// takeFlagValue reads "--name value" or "--name=value" starting at args[i].
// It returns the value and the index after it.
func takeFlagValue(args []string, i int, name string) (string, int, error) {
	if value, ok := strings.CutPrefix(args[i], name+"="); ok {
		return value, i + 1, nil
	}
	if i+1 >= len(args) {
		return "", i, fmt.Errorf("%s requires a value", name)
	}
	return args[i+1], i + 2, nil
}

func isFlag(arg string, name string) bool {
	return arg == name || strings.HasPrefix(arg, name+"=")
}

// resolveProjectDir returns the project directory for engine-backed commands:
// the --project value when given, else the working directory when it holds a
// project.godot (Godot's own --path semantics).
func resolveProjectDir(explicit string) (string, error) {
	dir := strings.TrimSpace(explicit)
	if dir == "" {
		dir = "."
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(filepath.Join(abs, "project.godot"))
	if err != nil || info.IsDir() {
		if explicit != "" {
			return "", fmt.Errorf("no project.godot in %s", abs)
		}
		return "", fmt.Errorf("no project.godot in %s (pass --project <dir>)", abs)
	}
	return abs, nil
}

// runEngineCommand is the shared entry point for engine-backed commands: it
// resolves the engine and applies the engine timeout.
func runEngineCommand(fallback time.Duration) (engineRef, context.Context, context.CancelFunc, error) {
	ref, err := resolveEngine()
	if err != nil {
		return engineRef{}, nil, nil, err
	}
	ctx, cancel := engineContextTimeout(fallback)
	return ref, ctx, cancel, nil
}
