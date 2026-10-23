package cmd

import (
	"context"
	"errors"
	"fmt"
	"github.com/NotNull92/hera-agent-godot/internal/protocol"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

// engineFlagPath is set from the leading global flag --engine <path> and may
// also be given after an engine-backed command. No-editor commands never
// require a live editor; the engine comes from this flag, HERA_GODOT_ENGINE or
// GODOT_BIN.
var engineFlagPath string

const (
	engineOutputLimit       = 64 * 1024
	engineVersionTimeout    = 10 * time.Second
	engineProjectRunTimeout = 300 * time.Second
)

func runEngine(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: engine <info|doctor> [--engine <path>] [--project <dir>]")
		return 2
	}
	switch args[0] {
	case "info":
		return runEngineInfo(args[1:])
	case "doctor":
		return runEngineDoctor(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown engine subcommand %q (want info|doctor)\n", args[0])
		return 2
	}
}

func parseEngineArgs(args []string) (project string, err error) {
	for i := 0; i < len(args); i++ {
		switch {
		case isFlag(args[i], "--engine"):
			value, next, err := takeFlagValue(args, i, "--engine")
			if err != nil {
				return "", err
			}
			engineFlagPath = value
			i = next - 1
		case isFlag(args[i], "--project"):
			value, next, err := takeFlagValue(args, i, "--project")
			if err != nil {
				return "", err
			}
			project = value
			i = next - 1
		default:
			return "", fmt.Errorf("unknown flag %q", args[i])
		}
	}
	return project, nil
}

func runEngineInfo(args []string) int {
	project, err := parseEngineArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "engine info: %v\n", err)
		return 2
	}
	if project != "" {
		project, err = resolveProjectDir(project)
		if err != nil {
			fmt.Fprintf(os.Stderr, "engine info: %v\n", err)
			return 1
		}
	}
	ref, ctx, cancel, err := runEngineCommand(engineVersionTimeout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "engine info: %v\n", err)
		return 1
	}
	defer cancel()
	info, err := collectEngineInfo(ctx, ref, true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "engine info: %v\n", err)
		return 1
	}
	if project != "" {
		info["project_path"] = filepath.ToSlash(project)
	}
	if code := printData(&protocol.Response{OK: true, Data: info}); code != 0 {
		return code
	}
	return 0
}

// collectEngineInfo gathers engine identity. withHelp=false skips the --help run
// when only the version string is needed.
func collectEngineInfo(ctx context.Context, ref engineRef, withHelp bool) (map[string]any, error) {
	stat, statErr := os.Stat(ref.Path)
	info := map[string]any{
		"engine": map[string]any{"path": ref.Path, "source": ref.Source},
	}
	if statErr == nil {
		info["engine"].(map[string]any)["bytes"] = stat.Size()
	}
	version, err := readEngineVersion(ctx, ref.Path)
	if err != nil {
		return nil, err
	}
	versionData := map[string]any{"string": version.String, "mono": version.Mono}
	if version.Status != "" {
		versionData["status"] = version.Status
	}
	if version.Commit != "" {
		versionData["commit"] = version.Commit
	}
	info["version"] = versionData
	if !withHelp {
		return info, nil
	}
	help := readEngineHelp(ctx, ref.Path)
	info["help_available"] = help.Available
	if help.Available {
		info["editor_build"] = help.Editor
		present := map[string]any{}
		for _, flag := range helpFlags {
			present[flag] = help.Present[flag]
		}
		info["help_flags"] = present
	} else if help.ProbeError != "" {
		// Absence of evidence is not proof of a template build, and no flag
		// inventory is claimed for a probe that did not complete.
		info["help_probe_error"] = help.ProbeError
	}
	return info, nil
}

// editorBuildState reports whether the build kind is known and whether it is an
// editor build. An unavailable help probe leaves it unknown.
func editorBuildState(info map[string]any) (known bool, editor bool) {
	value, ok := info["editor_build"].(bool)
	if !ok {
		return false, false
	}
	return true, value
}

func runEngineDoctor(args []string) int {
	project, err := parseEngineArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "engine doctor: %v\n", err)
		return 2
	}
	ref, ctx, cancel, err := runEngineCommand(engineVersionTimeout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "engine doctor: %v\n", err)
		return 1
	}
	defer cancel()
	result := map[string]any{}
	checks := []map[string]any{}
	warnings := []string{}
	info, err := collectEngineInfo(ctx, ref, true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "engine doctor: %v\n", err)
		return 1
	}
	for key, value := range info {
		result[key] = value
	}
	version, versionOK := info["version"].(map[string]any)
	checks = append(checks, engineCheck("engine_resolved", true,
		fmt.Sprintf("%s (%s)", ref.Path, ref.Source)))
	checks = append(checks, engineCheck("engine_version", versionOK && version["string"] != "", fmt.Sprint(version["string"])))
	editorKnown, editorBuild := editorBuildState(info)
	switch {
	case !editorKnown:
		detail := "help probe unavailable: the build kind could not be observed"
		if probe, ok := info["help_probe_error"].(string); ok && probe != "" {
			detail += " (" + probe + ")"
		}
		checks = append(checks, engineCheck("editor_build", false, detail))
		warnings = append(warnings, "editor build kind unknown: --help did not complete, so editor-only flags cannot be confirmed")
	case editorBuild:
		checks = append(checks, engineCheck("editor_build", true, editorBuildDetail(true)))
	default:
		checks = append(checks, engineCheck("editor_build", false, editorBuildDetail(false)))
		warnings = append(warnings, "not an editor build: --import, --export-*, --doctool, --dap-port and --lsp-port are unavailable")
	}

	templateRoot, templateDir, templateFiles, templateErr := engineTemplateDir(version, ref.Path)
	template := map[string]any{}
	switch {
	case templateErr != nil:
		template["version_dir"] = ""
		warnings = append(warnings, "export templates not checked: "+templateErr.Error())
	case templateDir != "":
		template["root"], template["version_dir"], template["files"] = templateRoot, templateDir, templateFiles
		template["found"] = true
	default:
		template["root"], template["version_dir"], template["found"] = templateRoot, "", false
	}
	result["export_templates"] = template
	checks = append(checks, engineCheck("export_templates", template["found"] == true,
		fmt.Sprintf("looked for %s under %s; a custom template can still be configured in the preset", template["version_dir"], templateRoot)))

	projectDir, projectErr := resolveProjectDir(project)
	if projectErr != nil {
		result["project"] = map[string]any{"found": false, "reason": projectErr.Error()}
		checks = append(checks, engineCheck("project", false, projectErr.Error()))
	} else {
		projectInfo := map[string]any{"found": true, "path": filepath.ToSlash(projectDir)}
		projectInfo["cache"] = projectCacheState(projectDir)
		projectInfo["hera_addon"] = projectHeraAddonState(projectDir)
		result["project"] = projectInfo
		checks = append(checks, engineCheck("project", true, filepath.ToSlash(projectDir)))
		cache := projectInfo["cache"].(map[string]any)
		checks = append(checks, engineCheck("project_cache", cache["uid_cache"] == true,
			"uid_cache.bin present: import ran at least once"))
		addon := projectInfo["hera_addon"].(map[string]any)
		checks = append(checks, engineCheck("hera_addon", addon["listed_in_project_file"] == true,
			fmt.Sprint(addon["detail"])))
	}

	result["checks"] = checks
	result["warnings"] = warnings
	result["ok"] = true
	if code := printData(&protocol.Response{OK: true, Data: result}); code != 0 {
		return code
	}
	return 0
}

func engineCheck(name string, ok bool, detail string) map[string]any {
	return map[string]any{"name": name, "ok": ok, "detail": detail}
}

func editorBuildDetail(editor bool) string {
	if editor {
		return "editor build: editor-only flags are available"
	}
	return "template build: editor-only flags unavailable"
}

// engineTemplateDir resolves the export-template version directory for a
// version, mirroring Godot's "<major>.<minor>.<patch>.<status>[.mono]" layout
// across the editor data directories Godot can use.
func engineTemplateDir(version map[string]any, enginePath string) (root string, dir string, files int, err error) {
	raw, _ := version["string"].(string)
	parts := strings.Split(raw, ".")
	if len(parts) < 4 {
		return "", "", 0, errors.New("engine version string was not parsed")
	}
	name := strings.Join(parts[:4], ".")
	if version["mono"] == true {
		name += ".mono"
	}
	roots := engineTemplateRoots(enginePath)
	for _, candidate := range roots {
		if candidate == "" {
			continue
		}
		path := filepath.Join(candidate, name)
		info, statErr := os.Stat(path)
		if statErr != nil || !info.IsDir() {
			continue
		}
		entries, readErr := os.ReadDir(path)
		count := 0
		if readErr == nil {
			for _, entry := range entries {
				if !entry.IsDir() {
					count++
				}
			}
		}
		return filepath.ToSlash(candidate), filepath.ToSlash(path), count, nil
	}
	return filepath.ToSlash(firstNonEmpty(roots)), "", 0, nil
}

// engineTemplateRoots lists the directories Godot may resolve export templates
// from: the platform data directory, XDG_DATA_HOME on Unix, and the
// self-contained layout next to the executable.
func engineTemplateRoots(enginePath string) []string {
	roots := []string{}
	if dir, err := os.UserConfigDir(); err == nil {
		roots = append(roots, filepath.Join(dir, "Godot", "export_templates"))
	}
	if xdg := strings.TrimSpace(os.Getenv("XDG_DATA_HOME")); xdg != "" {
		roots = append(roots, filepath.Join(xdg, "godot", "export_templates"))
	}
	if home, err := os.UserHomeDir(); err == nil {
		roots = append(roots, filepath.Join(home, ".local", "share", "godot", "export_templates"))
	}
	// Godot's self-contained mode keeps editor data next to the executable.
	if enginePath != "" {
		exeDir := filepath.Dir(enginePath)
		if fileExists(filepath.Join(exeDir, "_sc_")) || runtime.GOOS == "windows" {
			roots = append(roots, filepath.Join(exeDir, "editor_data", "export_templates"))
		}
	}
	return roots
}

func firstNonEmpty(values []string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

// projectCacheState reports which generated caches exist in a project.
func projectCacheState(projectDir string) map[string]any {
	godot := filepath.Join(projectDir, ".godot")
	return map[string]any{
		"dir":          dirExists(godot),
		"uid_cache":    fileExists(filepath.Join(godot, "uid_cache.bin")),
		"imported":     dirExists(filepath.Join(godot, "imported")),
		"editor_state": dirExists(filepath.Join(godot, "editor")),
	}
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// projectHeraAddonState reports whether the Hera addon is installed and listed
// as an enabled editor plugin. The check reads project.godot only; it says
// nothing about a running editor.
func projectHeraAddonState(projectDir string) map[string]any {
	const addonPath = "addons/hera_agent_godot/plugin.cfg"
	installed := fileExists(filepath.Join(projectDir, filepath.FromSlash(addonPath)))
	state := map[string]any{"installed": installed, "listed_in_project_file": false}
	text, err := os.ReadFile(filepath.Join(projectDir, "project.godot"))
	if err != nil {
		state["detail"] = "project.godot unreadable: " + err.Error()
		return state
	}
	enabled := strings.Contains(string(text), "addons/hera_agent_godot/plugin.cfg")
	state["listed_in_project_file"] = enabled
	switch {
	case !installed:
		state["detail"] = "addon not installed in this project"
	case enabled:
		state["detail"] = "addon installed and enabled in project.godot"
	default:
		state["detail"] = "addon installed but not enabled in project.godot"
	}
	return state
}

// sortedBenchmarkEntries returns benchmark phases ordered by descending cost.
func sortedBenchmarkEntries(values map[string]float64) []map[string]any {
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		if values[names[i]] == values[names[j]] {
			return names[i] < names[j]
		}
		return values[names[i]] > values[names[j]]
	})
	entries := make([]map[string]any, 0, len(names))
	for _, name := range names {
		entries = append(entries, map[string]any{"name": name, "seconds": values[name]})
	}
	return entries
}
