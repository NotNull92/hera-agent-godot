package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/NotNull92/hera-agent-godot/internal/protocol"
)

// project export kinds map to the engine's --export-* flags.
var projectExportKinds = []string{"pack", "release", "debug", "patch"}

// parseProjectImportArgs parses: project import [--project DIR] [--engine PATH]
func parseProjectImportArgs(args []string) (map[string]any, error) {
	params := map[string]any{"action": "import"}
	for i := 0; i < len(args); i++ {
		switch {
		case isFlag(args[i], "--project"):
			value, next, err := takeFlagValue(args, i, "--project")
			if err != nil {
				return nil, err
			}
			params["project"] = value
			i = next - 1
		case isFlag(args[i], "--engine"):
			value, next, err := takeFlagValue(args, i, "--engine")
			if err != nil {
				return nil, err
			}
			engineFlagPath = value
			i = next - 1
		default:
			return nil, fmt.Errorf("unknown flag %q", args[i])
		}
	}
	return params, nil
}

// parseProjectExportArgs parses:
//
//	project export <preset> <output> [--kind pack|release|debug|patch] [--patches a,b]
//	                              [--verbose] [--project DIR] [--engine PATH]
//
// --verbose includes the raw engine output, which is omitted by default because
// signing tools and export plugins can log credential-bearing arguments.
func parseProjectExportArgs(args []string) (map[string]any, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("usage: project export <preset> <output> [--kind pack|release|debug|patch] [--patches a,b] [--verbose]")
	}
	params := map[string]any{"action": "export", "preset": args[0], "output": args[1], "kind": "pack", "verbose": false}
	for i := 2; i < len(args); i++ {
		switch {
		case args[i] == "--verbose":
			params["verbose"] = true
		case isFlag(args[i], "--kind"):
			value, next, err := takeFlagValue(args, i, "--kind")
			if err != nil {
				return nil, err
			}
			if !validProjectExportKind(value) {
				return nil, fmt.Errorf("--kind must be one of %s", strings.Join(projectExportKinds, "|"))
			}
			params["kind"] = value
			i = next - 1
		case isFlag(args[i], "--patches"):
			value, next, err := takeFlagValue(args, i, "--patches")
			if err != nil {
				return nil, err
			}
			params["patches"] = value
			i = next - 1
		case isFlag(args[i], "--project"):
			value, next, err := takeFlagValue(args, i, "--project")
			if err != nil {
				return nil, err
			}
			params["project"] = value
			i = next - 1
		case isFlag(args[i], "--engine"):
			value, next, err := takeFlagValue(args, i, "--engine")
			if err != nil {
				return nil, err
			}
			engineFlagPath = value
			i = next - 1
		default:
			return nil, fmt.Errorf("unknown flag %q", args[i])
		}
	}
	return params, nil
}

func validProjectExportKind(value string) bool {
	for _, kind := range projectExportKinds {
		if kind == value {
			return true
		}
	}
	return false
}

// runProjectImport runs the engine's headless import pass. Success is the exit
// code plus observed artifacts: engine progress text is localized and is never
// parsed.
func runProjectImport(params map[string]any) int {
	project, err := resolveProjectDir(stringParam(params, "project"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "project import: %v\n", err)
		return 1
	}
	ref, ctx, cancel, err := runEngineCommand(engineProjectRunTimeout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "project import: %v\n", err)
		return 1
	}
	defer cancel()

	// Version identity is collected before the workload consumes the deadline.
	version, _ := readEngineVersion(ctx, ref.Path)
	run := runEngineBounded(ctx, ref.Path, []string{"--headless", "--path", project, "--import"}, project, nil, engineOutputLimit)

	result := mergeResult(map[string]any{
		"project_path":     filepath.ToSlash(project),
		"cache":            projectCacheState(project),
		"script_uid_files": countScriptUIDFiles(project),
		"engine":           engineIdentity(ref, version),
		"notes":            []string{"engine progress text is localized; read exit_code and artifacts, not stdout text"},
		"source":           "engine_process",
	}, run)
	return finishEngineResult(result, run)
}

// runProjectExport exports with the engine after a fail-fast preflight. The
// preflight reads export_presets.cfg only to list preset names; the file can
// hold credentials and is never printed. Template lookup is advisory because a
// preset can point at custom templates.
func runProjectExport(params map[string]any) int {
	preset := fmt.Sprint(params["preset"])
	output := fmt.Sprint(params["output"])
	kind := fmt.Sprint(params["kind"])
	project, err := resolveProjectDir(stringParam(params, "project"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "project export: %v\n", err)
		return 1
	}
	outputAbs, err := filepath.Abs(output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "project export: %v\n", err)
		return 2
	}
	base := map[string]any{"project_path": filepath.ToSlash(project), "kind": kind, "preset": preset,
		"output_path": filepath.ToSlash(outputAbs)}
	if code, detail := exportDirectoryProblem(filepath.Dir(outputAbs)); code != "" {
		base["error"], base["detail"] = code, detail
		return failStructured(base)
	}
	presets, presetErr := readExportPresetNames(filepath.Join(project, "export_presets.cfg"))
	if presetErr != nil {
		code, detail := "export_presets_unreadable", "cannot read export_presets.cfg: "+presetErr.Error()
		if errors.Is(presetErr, fs.ErrNotExist) {
			code, detail = "export_presets_missing", "create an export preset in Godot (Project > Export) first"
		}
		base["error"], base["detail"] = code, detail
		return failStructured(base)
	}
	if !containsString(presets, preset) {
		base["error"], base["available_presets"] = "preset_not_found", presets
		return failStructured(base)
	}
	if kind == "pack" {
		switch strings.ToLower(filepath.Ext(outputAbs)) {
		case ".pck", ".zip":
		default:
			base["error"] = "pack_extension"
			base["detail"] = "the pack output extension selects the format: .pck or .zip"
			return failStructured(base)
		}
	}

	ref, ctx, cancel, err := runEngineCommand(engineProjectRunTimeout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "project export: %v\n", err)
		return 1
	}
	defer cancel()
	version, _ := readEngineVersion(ctx, ref.Path)
	help := readEngineHelp(ctx, ref.Path)
	identity := engineIdentity(ref, version)
	warnings := []string{}
	// Only a completed help probe can prove the build kind.
	if help.Available && !help.Editor {
		base["engine"] = identity
		base["error"] = "editor_build_required"
		base["detail"] = "exporting requires an editor build of the engine"
		return failStructured(base)
	}
	if !help.Available {
		warnings = append(warnings, "editor build kind unknown ("+help.ProbeError+"); continuing and letting the engine decide")
	}
	if kind == "release" || kind == "debug" {
		versionData := map[string]any{"string": version.String, "mono": version.Mono}
		if _, dir, files, templateErr := engineTemplateDir(versionData, ref.Path); templateErr == nil && dir != "" {
			identity["export_templates_dir"], identity["export_template_files"] = dir, files
		} else if templateErr == nil {
			warnings = append(warnings, "export templates were not found in the usual locations; a preset can still use a custom template, so the engine validates it")
		}
	}

	engineArgs := []string{"--headless", "--path", project, "--export-" + kind, preset, outputAbs}
	if patches := stringParam(params, "patches"); patches != "" {
		engineArgs = append(engineArgs, "--patches", patches)
	}
	run := runEngineBounded(ctx, ref.Path, engineArgs, project, nil, engineOutputLimit)
	stat, statErr := os.Stat(outputAbs)
	result := mergeResult(base, run)
	result["available_presets"] = presets
	result["engine"] = identity
	result["notes"] = []string{"export_presets.cfg and .godot/export_credentials.cfg can contain secrets and are never printed"}
	result["source"] = "engine_process"
	result["output_exists"] = statErr == nil && !stat.IsDir()
	if statErr == nil && !stat.IsDir() {
		result["output_bytes"] = stat.Size()
	}
	if verbose, _ := params["verbose"].(bool); !verbose {
		// Signing tools and export plugins can log credential-bearing arguments.
		delete(result, "output")
		delete(result, "output_truncated")
		result["output_omitted"] = true
		result["output_omitted_reason"] = "export logs can contain credential-bearing signing-tool output; pass --verbose to include them"
	}
	if len(warnings) > 0 {
		result["warnings"] = warnings
	}
	if run.ExitCode == 0 {
		// A clean exit that produced no artifact is not a successful export: pack
		// writes a file, release/debug write into the target directory.
		expectsDirectory := kind == "release" || kind == "debug"
		artifactPresent := statErr == nil && stat.IsDir() == expectsDirectory
		if !artifactPresent {
			result["error"] = "no_artifact_produced"
			result["detail"] = "the engine exited without writing the requested export artifact: " + filepath.ToSlash(outputAbs)
		}
	}
	return finishEngineResult(result, run)
}

// exportDirectoryProblem reports why the engine could not write into dir,
// separating a missing directory from an inaccessible one or a wrong type.
func exportDirectoryProblem(dir string) (code string, detail string) {
	info, err := os.Stat(dir)
	switch {
	case err == nil && !info.IsDir():
		return "output_directory_missing", "the target path is not a directory: " + filepath.ToSlash(dir)
	case err != nil && errors.Is(err, fs.ErrNotExist):
		return "output_directory_missing", "the engine requires the target directory to exist: " + filepath.ToSlash(dir)
	case err != nil:
		return "output_directory_unreadable", "cannot read the target directory: " + err.Error()
	}
	return "", ""
}

// failStructured prints the structured failure on stdout, a one-line diagnostic
// on stderr, and returns 1.
func failStructured(data map[string]any) int {
	message := fmt.Sprint(data["error"])
	if detail, ok := data["detail"].(string); ok && detail != "" {
		message += ": " + detail
	}
	data["ok"] = false
	data["message"] = message
	fmt.Fprintf(os.Stderr, "error: %s\n", message)
	if code := printData(&protocol.Response{OK: true, Data: data}); code != 0 {
		return code
	}
	return 1
}

var exportPresetSection = regexp.MustCompile(`^\[preset\.(\d+)\]\s*$`)
var exportPresetName = regexp.MustCompile(`^name\s*=\s*"(.*)"\s*$`)

// readExportPresetNames returns the preset names in section order. Only names
// are read; option values may contain credentials.
func readExportPresetNames(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	type preset struct {
		index int
		name  string
	}
	found := []preset{}
	current := -1
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if match := exportPresetSection.FindStringSubmatch(line); match != nil {
			index, convErr := strconv.Atoi(match[1])
			if convErr != nil {
				current = -1
				continue
			}
			current = index
			found = append(found, preset{index: index})
			continue
		}
		if strings.HasPrefix(line, "[") {
			// Any other section ends the preset being read, so a later
			// options block cannot fill in a missing name.
			current = -1
			continue
		}
		if current < 0 {
			continue
		}
		if match := exportPresetName.FindStringSubmatch(line); match != nil {
			for i := range found {
				if found[i].index == current && found[i].name == "" {
					found[i].name = unescapeGodotString(match[1])
				}
			}
		}
	}
	names := []string{}
	for _, item := range found {
		if item.name != "" {
			names = append(names, item.name)
		}
	}
	return names, nil
}

// unescapeGodotString decodes the escapes Godot writes inside quoted strings, so
// a preset stored as Windows \"QA\" can be selected by its real name.
func unescapeGodotString(value string) string {
	if !strings.Contains(value, "\\") {
		return value
	}
	var builder strings.Builder
	for i := 0; i < len(value); i++ {
		if value[i] != '\\' || i+1 >= len(value) {
			builder.WriteByte(value[i])
			continue
		}
		i++
		switch value[i] {
		case 'n':
			builder.WriteByte('\n')
		case 't':
			builder.WriteByte('\t')
		case 'r':
			builder.WriteByte('\r')
		default:
			builder.WriteByte(value[i])
		}
	}
	return builder.String()
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
