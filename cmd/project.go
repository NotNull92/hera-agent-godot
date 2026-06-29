package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/NotNull92/hera-agent-godot/internal/discovery"
)

func runProject(args []string) int {
	params, err := parseProjectArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "project: %v\n", err)
		return 2
	}
	if params["action"] == "set_main_scene" {
		return runProjectSetMainScene(params)
	}
	if projectActionMutates(params["action"]) {
		return dialMutationPostPrint("project", params, "project")
	}
	return dialPostPrint("project", params, "project")
}

func projectActionMutates(action any) bool {
	switch action {
	case "mkdir", "set_main_scene", "scan", "reimport":
		return true
	default:
		return false
	}
}

func parseProjectArgs(args []string) (map[string]any, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("usage: project <info|list-files|scan|reimport|mkdir|set-main-scene> ...")
	}
	switch args[0] {
	case "info":
		if len(args) != 1 {
			return nil, fmt.Errorf("usage: project info")
		}
		return map[string]any{"action": "info"}, nil

	case "list-files":
		return parseProjectListFilesArgs(args[1:])

	case "scan":
		if len(args) != 1 {
			return nil, fmt.Errorf("usage: project scan")
		}
		return map[string]any{"action": "scan"}, nil
	case "reimport":
		return parseProjectReimportArgs(args[1:])
	case "mkdir":
		if len(args) != 2 {
			return nil, fmt.Errorf("usage: project mkdir <res://dir>")
		}
		return map[string]any{"action": "mkdir", "path": args[1]}, nil
	case "set-main-scene":
		if len(args) != 2 {
			return nil, fmt.Errorf("usage: project set-main-scene <res://scene.tscn>")
		}
		return map[string]any{"action": "set_main_scene", "path": args[1]}, nil
	default:
		return nil, fmt.Errorf("unknown project subcommand %q (want info|list-files|scan|reimport|mkdir|set-main-scene)", args[0])
	}
}

func parseProjectReimportArgs(args []string) (map[string]any, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("usage: project reimport <res://file> ...")
	}
	paths := make([]string, 0, len(args))
	paths = append(paths, args...)
	return map[string]any{"action": "reimport", "paths": paths}, nil
}

func parseProjectListFilesArgs(args []string) (map[string]any, error) {
	params := map[string]any{"action": "list_files"}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--type":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--type requires a value")
			}
			i++
			if !validProjectFileType(args[i]) {
				return nil, fmt.Errorf("--type must be one of all|scene|script|resource|asset|shader")
			}
			params["type"] = args[i]
		case "--pattern":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--pattern requires a value")
			}
			i++
			params["pattern"] = args[i]
		case "--limit":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--limit requires a value")
			}
			i++
			limit, err := strconv.Atoi(args[i])
			if err != nil || limit <= 0 {
				return nil, fmt.Errorf("--limit must be a positive integer")
			}
			params["limit"] = limit
		default:
			return nil, fmt.Errorf("unknown flag %q", args[i])
		}
	}
	return params, nil
}

func validProjectFileType(value string) bool {
	switch value {
	case "all", "scene", "script", "resource", "asset", "shader":
		return true
	default:
		return false
	}
}

func runProjectSetMainScene(params map[string]any) int {
	scenePath, ok := params["path"].(string)
	if !ok {
		fmt.Fprintln(os.Stderr, "project: scene path is required")
		return 2
	}
	instances, err := discovery.Discover()
	if err != nil {
		fmt.Fprintf(os.Stderr, "project: %v\n", err)
		return 1
	}
	inst, err := selectEditor(instances, true, targetPID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "project: %v\n", err)
		return 1
	}
	if err := setMainSceneInProjectFile(inst.ProjectPath, scenePath); err != nil {
		fmt.Fprintf(os.Stderr, "project: %v\n", err)
		return 1
	}
	data := map[string]any{"main_scene": scenePath, "project_path": filepath.Clean(inst.ProjectPath)}
	out, err := json.Marshal(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "project: %v\n", err)
		return 1
	}
	fmt.Println(string(out))
	return 0
}

func setMainSceneInProjectFile(projectPath string, scenePath string) error {
	if !validMainScenePath(scenePath) {
		return fmt.Errorf("scene path must be a safe res:// .tscn path")
	}
	projectFile := filepath.Join(projectPath, "project.godot")
	sceneFile := filepath.Join(projectPath, filepath.FromSlash(strings.TrimPrefix(scenePath, "res://")))
	if _, err := os.Stat(sceneFile); err != nil {
		return fmt.Errorf("scene not found: %s", scenePath)
	}
	raw, err := os.ReadFile(projectFile)
	if err != nil {
		return fmt.Errorf("read project.godot: %w", err)
	}
	updated := updateMainSceneSetting(string(raw), scenePath)
	if err := os.WriteFile(projectFile, []byte(updated), 0o600); err != nil {
		return fmt.Errorf("write project.godot: %w", err)
	}
	return nil
}

func readMainSceneFromProjectFile(projectPath string) (string, error) {
	raw, err := os.ReadFile(filepath.Join(projectPath, "project.godot"))
	if err != nil {
		return "", fmt.Errorf("read project.godot: %w", err)
	}
	for _, line := range strings.Split(string(raw), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "run/main_scene=") {
			return strings.Trim(strings.TrimPrefix(trimmed, "run/main_scene="), "\""), nil
		}
	}
	return "", nil
}

func validMainScenePath(scenePath string) bool {
	if !strings.HasPrefix(scenePath, "res://") || filepath.Ext(scenePath) != ".tscn" || strings.Contains(scenePath, "\\") {
		return false
	}
	rel := strings.TrimPrefix(scenePath, "res://")
	if rel == "" || strings.HasPrefix(rel, "/") {
		return false
	}
	for _, part := range strings.Split(rel, "/") {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	return true
}

func updateMainSceneSetting(contents string, scenePath string) string {
	lines := strings.SplitAfter(contents, "\n")
	inApplication := false
	insertAt := len(lines)
	for idx, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			if inApplication {
				insertAt = idx
				break
			}
			inApplication = trimmed == "[application]"
			continue
		}
		if inApplication && strings.HasPrefix(trimmed, "run/main_scene=") {
			lines[idx] = fmt.Sprintf("run/main_scene=\"%s\"\n", scenePath)
			return strings.Join(lines, "")
		}
	}
	if !inApplication && insertAt == len(lines) {
		if contents != "" && !strings.HasSuffix(contents, "\n") {
			contents += "\n"
		}
		return contents + "\n[application]\n" + fmt.Sprintf("run/main_scene=\"%s\"\n", scenePath)
	}
	newLine := fmt.Sprintf("run/main_scene=\"%s\"\n", scenePath)
	lines = append(lines[:insertAt], append([]string{newLine}, lines[insertAt:]...)...)
	return strings.Join(lines, "")
}
