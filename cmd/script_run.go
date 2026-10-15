package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const scriptRunTimeout = 60 * time.Second

// parseScriptRunArgs parses:
//
//	script run <res://tool.gd> [--quit-after N] [--project DIR] [--engine PATH] [-- args...]
//
// The script runs on the engine's headless SceneTree harness; arguments after
// -- reach the script through OS.get_cmdline_user_args().
func parseScriptRunArgs(args []string) (map[string]any, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("usage: script run <res://tool.gd> [--quit-after N] [-- args...]")
	}
	path := args[0]
	if !strings.HasPrefix(path, "res://") {
		return nil, fmt.Errorf("script run path must start with res://")
	}
	params := map[string]any{"action": "run", "path": path}
	arguments := []string{}
	for i := 1; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			arguments = append(arguments, args[i+1:]...)
			break
		}
		switch {
		case isFlag(arg, "--quit-after"):
			value, next, err := takeFlagValue(args, i, "--quit-after")
			if err != nil {
				return nil, err
			}
			frames, err := strconv.Atoi(value)
			if err != nil || frames < 0 {
				return nil, fmt.Errorf("--quit-after must be a non-negative integer")
			}
			params["quit_after"] = frames
			i = next - 1
		case isFlag(arg, "--project"):
			value, next, err := takeFlagValue(args, i, "--project")
			if err != nil {
				return nil, err
			}
			params["project"] = value
			i = next - 1
		case isFlag(arg, "--engine"):
			value, next, err := takeFlagValue(args, i, "--engine")
			if err != nil {
				return nil, err
			}
			engineFlagPath = value
			i = next - 1
		default:
			return nil, fmt.Errorf("unknown flag %q (script arguments go after --)", arg)
		}
	}
	if len(arguments) > 0 {
		params["arguments"] = arguments
	}
	return params, nil
}

// runScriptRun executes a project script with the engine's headless harness.
// The engine exit code, not stdout text, decides success.
func runScriptRun(params map[string]any) int {
	path := fmt.Sprint(params["path"])
	project, err := resolveProjectDir(stringParam(params, "project"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "script run: %v\n", err)
		return 1
	}
	if harnessErr := engineHarnessError(project, path); harnessErr != nil {
		return failStructured(map[string]any{"path": path, "project_path": filepath.ToSlash(project),
			"error": "not_a_harness", "detail": harnessErr.Error()})
	}
	ref, ctx, cancel, err := runEngineCommand(scriptRunTimeout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "script run: %v\n", err)
		return 1
	}
	defer cancel()

	arguments := []string{}
	if value, ok := params["arguments"].([]string); ok {
		arguments = value
	}
	engineArgs := []string{"--headless", "--path", project, "--script", path}
	if frames, ok := params["quit_after"].(int); ok {
		engineArgs = append(engineArgs, "--quit-after", strconv.Itoa(frames))
	}
	if len(arguments) > 0 {
		engineArgs = append(engineArgs, "--")
		engineArgs = append(engineArgs, arguments...)
	}
	// Version identity is collected before the workload consumes the deadline.
	version, _ := readEngineVersion(ctx, ref.Path)
	run := runEngineBounded(ctx, ref.Path, engineArgs, project, nil, engineOutputLimit)

	result := mergeResult(map[string]any{
		"path": path, "project_path": filepath.ToSlash(project), "arguments": arguments,
		"engine": engineIdentity(ref, version),
		"source": "engine_process",
	}, run)
	if failure := runFailureCode(run); failure != "" {
		result["error"] = failure
	}
	return finishEngineResult(result, run)
}

func stringParam(params map[string]any, key string) string {
	value, _ := params[key].(string)
	return value
}

// engineHarnessError rejects scripts that cannot be a headless harness. Godot
// 4.7.2 hangs instead of erroring when --script points at a non-MainLoop
// script, so this pre-flight replaces a silent timeout with a clear failure.
// Only bases known to be incompatible are rejected: an unresolved class name
// (for example `extends MyHarness`, where MyHarness extends SceneTree) is left
// to the engine.
//
// shortcut: the list is a heuristic set of common engine classes; a rare
// incompatible base outside it still ends in the engine timeout.
func engineHarnessError(projectDir, path string) error {
	if !strings.HasSuffix(path, ".gd") {
		return nil
	}
	diskPath := filepath.Join(projectDir, filepath.FromSlash(strings.TrimPrefix(path, "res://")))
	data, err := os.ReadFile(diskPath)
	if err != nil {
		return nil
	}
	base := scriptExtendsBase(string(data))
	if base == "" || base == "SceneTree" || base == "MainLoop" || !incompatibleHarnessBases[base] {
		return nil
	}
	return fmt.Errorf("%s extends %s; the harness script must extend SceneTree or MainLoop", path, base)
}

// incompatibleHarnessBases are engine classes that cannot run as a MainLoop.
var incompatibleHarnessBases = map[string]bool{
	"Object": true, "RefCounted": true, "Resource": true,
	"Node": true, "Node2D": true, "Node3D": true, "CanvasItem": true, "CanvasLayer": true,
	"Control": true, "Window": true, "Viewport": true, "SubViewport": true, "SubViewportContainer": true,
	"Sprite2D": true, "Sprite3D": true, "AnimatedSprite2D": true, "AnimatedSprite3D": true,
	"CharacterBody2D": true, "CharacterBody3D": true, "RigidBody2D": true, "RigidBody3D": true,
	"StaticBody2D": true, "StaticBody3D": true, "Area2D": true, "Area3D": true,
	"CollisionObject2D": true, "CollisionObject3D": true, "CollisionShape2D": true, "CollisionShape3D": true,
	"Camera2D": true, "Camera3D": true, "MeshInstance3D": true, "MultiMeshInstance3D": true,
	"AnimationPlayer": true, "AudioStreamPlayer": true, "AudioStreamPlayer2D": true, "AudioStreamPlayer3D": true,
	"Timer": true, "Label": true, "RichTextLabel": true, "Button": true, "TextureRect": true, "ColorRect": true,
	"Panel": true, "PanelContainer": true, "MarginContainer": true, "VBoxContainer": true, "HBoxContainer": true,
	"GridContainer": true, "CenterContainer": true, "ScrollContainer": true, "TabContainer": true,
	"LineEdit": true, "TextEdit": true, "CodeEdit": true, "ItemList": true, "Tree": true, "OptionButton": true,
	"CheckBox": true, "CheckButton": true, "ProgressBar": true, "Slider": true, "HSlider": true, "VSlider": true,
	"SpinBox": true, "AcceptDialog": true, "ConfirmationDialog": true, "FileDialog": true,
	"TileMap": true, "TileMapLayer": true, "Marker2D": true, "Marker3D": true, "Path2D": true, "Path3D": true,
	"RayCast2D": true, "RayCast3D": true, "ShapeCast2D": true, "ShapeCast3D": true,
	"NavigationAgent2D": true, "NavigationAgent3D": true, "NavigationRegion2D": true, "NavigationRegion3D": true,
	"Parallax2D": true, "ParallaxLayer": true, "ParallaxBackground": true, "GPUParticles2D": true, "GPUParticles3D": true,
}

// scriptExtendsBase returns the base class named by the first meaningful
// `extends` line, or "" when the file has none.
func scriptExtendsBase(source string) string {
	for _, line := range strings.Split(source, "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "\ufeff")
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "@") ||
			strings.HasPrefix(line, "class_name") {
			continue
		}
		rest, ok := strings.CutPrefix(line, "extends ")
		if !ok {
			return ""
		}
		target := strings.TrimSpace(rest)
		if cut := strings.IndexAny(target, " \t#"); cut >= 0 {
			target = target[:cut]
		}
		target = strings.Trim(target, "\"'")
		if strings.Contains(target, "/") || strings.Contains(target, ".") {
			return ""
		}
		return target
	}
	return ""
}

// countScriptUIDFiles counts the .uid sidecars a project import generates.
func countScriptUIDFiles(projectDir string) int {
	count := 0
	_ = filepath.WalkDir(projectDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() {
			if entry.Name() == ".godot" || entry.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(entry.Name(), ".uid") {
			count++
		}
		return nil
	})
	return count
}
