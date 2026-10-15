package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const docsDumpTimeout = 300 * time.Second

// parseDocsDumpArgs parses:
//
//	docs dump --out DIR [--format xml|json] [--scripts] [--force]
//	          [--project DIR] [--engine PATH]
//
// xml (default) writes the engine class reference, json writes a single
// extension_api.json including documentation, and --scripts writes only the
// project's own GDScript reference. Existing documentation requires --force to
// publish matching files from this run's isolated output directory.
func parseDocsDumpArgs(args []string) (map[string]any, error) {
	params := map[string]any{"format": "xml", "scripts": false, "force": false}
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--force":
			params["force"] = true
		case isFlag(args[i], "--out"):
			value, next, err := takeFlagValue(args, i, "--out")
			if err != nil {
				return nil, err
			}
			params["out"] = value
			i = next - 1
		case isFlag(args[i], "--format"):
			value, next, err := takeFlagValue(args, i, "--format")
			if err != nil {
				return nil, err
			}
			if value != "xml" && value != "json" {
				return nil, fmt.Errorf("--format must be xml or json")
			}
			params["format"] = value
			i = next - 1
		case args[i] == "--scripts":
			params["scripts"] = true
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
	if stringParam(params, "out") == "" {
		return nil, fmt.Errorf("--out <dir> is required: docs dump writes many files and never guesses a destination")
	}
	if params["scripts"] == true && params["format"] == "json" {
		return nil, fmt.Errorf("--scripts writes XML; use --format xml")
	}
	return params, nil
}

// runDocsDump writes engine or project API documentation with the engine's
// doctool in an isolated directory, then publishes verified output. Existing
// destination files are protected unless --force permits replacement.
func runDocsDump(params map[string]any) int {
	out := stringParam(params, "out")
	outAbs, err := filepath.Abs(out)
	if err != nil {
		fmt.Fprintf(os.Stderr, "docs dump: %v\n", err)
		return 2
	}
	force, _ := params["force"].(bool)
	format := fmt.Sprint(params["format"])
	mode := format
	// Resolve the destination first: a linked directory must not hide existing
	// XML from the guard.
	if resolved, resolveErr := resolveDestination(outAbs); resolveErr == nil {
		outAbs = resolved
	} else {
		fmt.Fprintf(os.Stderr, "docs dump: %v\n", resolveErr)
		return 1
	}
	var jsonBefore fileSnapshot
	if format == "json" {
		apiPath := filepath.Join(outAbs, "extension_api.json")
		jsonBefore = snapshotFile(apiPath)
		switch {
		case jsonBefore.IsDir:
			return failStructured(map[string]any{"mode": mode, "output_dir": filepath.ToSlash(outAbs),
				"error":  "output_is_directory",
				"detail": "the engine writes extension_api.json as a file: " + filepath.ToSlash(apiPath)})
		case jsonBefore.Exists && !force:
			return failStructured(map[string]any{"mode": mode, "output_dir": filepath.ToSlash(outAbs),
				"error":  "output_exists",
				"detail": "extension_api.json already exists; pass --force to replace it"})
		}
	} else {
		existing, scanErr := countXMLFilesFollowingLinks(outAbs, maxDocsScanDepth)
		switch {
		case scanErr != nil && !force:
			return failStructured(map[string]any{"mode": mode, "output_dir": filepath.ToSlash(outAbs),
				"error": "output_scan_failed",
				"detail": "the destination could not be scanned for existing XML, so existing documentation cannot be protected: " +
					scanErr.Error()})
		case scanErr == nil && existing > 0 && !force:
			return failStructured(map[string]any{"mode": mode, "output_dir": filepath.ToSlash(outAbs),
				"error": "output_not_empty", "xml_files": existing,
				"detail": "XML documentation already exists; pass --force to replace matching generated files, or choose an empty directory"})
		}
	}
	if err := os.MkdirAll(outAbs, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "docs dump: %v\n", err)
		return 1
	}
	ref, ctx, cancel, err := runEngineCommand(docsDumpTimeout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "docs dump: %v\n", err)
		return 1
	}
	defer cancel()

	scriptProject := ""
	if params["scripts"] == true {
		scriptProject, err = resolveProjectDir(stringParam(params, "project"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "docs dump: %v\n", err)
			return 1
		}
	}
	engineArgs := []string{"--headless"}
	stage, err := stageArtifacts(outAbs)
	if err != nil {
		return failStructured(map[string]any{"error": "artifact_stage_failed", "detail": err.Error()})
	}
	workDir := stage
	switch {
	case format == "json":
		engineArgs = append(engineArgs, "--dump-extension-api-with-docs")
	case params["scripts"] == true:
		mode = "scripts"
		engineArgs = append(engineArgs, "--doctool", stage, "--no-docbase", "--gdscript-docs", "res://")
		workDir = scriptProject
	default:
		engineArgs = append(engineArgs, "--doctool", stage)
	}
	// Version identity is collected before the workload consumes the deadline.
	version, _ := readEngineVersion(ctx, ref.Path)
	run := runEngineBounded(ctx, ref.Path, engineArgs, workDir, nil, engineOutputLimit)

	result := mergeResult(map[string]any{
		"mode": mode, "output_dir": filepath.ToSlash(outAbs),
		"engine": engineIdentity(ref, version),
		"notes": []string{
			"Godot requires the output directory to exist and the GDScript path to be project-relative",
			"generated XML/JSON is written by the engine; Hera reports counts and bytes only",
			"generation uses an isolated empty directory; publication replaces matching files only and retains other destination files",
		},
		"source": "engine_process",
	}, run)
	switch {
	case format == "json":
		apiPath := filepath.Join(stage, "extension_api.json")
		after := snapshotFile(apiPath)
		result["output_file"] = filepath.ToSlash(filepath.Join(outAbs, "extension_api.json"))
		switch {
		case after.IsDir || !after.Exists || after.Bytes == 0:
			result["error"] = "json_artifact_missing"
			result["detail"] = "the engine exited without writing a usable extension_api.json"

		default:
			result["output_bytes"] = after.Bytes
		}
	default:
		xmlFiles, xmlErr := countXMLFilesFollowingLinks(stage, maxDocsScanDepth)
		files, bytes, countErr := dirSize(stage)
		if countErr == nil {
			result["files"], result["bytes"] = files, bytes
		}
		if xmlErr != nil {
			result["xml_scan_error"] = xmlErr.Error()
		}
		if xmlErr == nil {
			result["xml_files"] = xmlFiles
		}
		switch {
		case xmlErr != nil:
			// A destination that cannot be scanned cannot be verified either.
			result["error"] = "docs_scan_failed"
			result["detail"] = "the generated documentation could not be scanned: " + xmlErr.Error()
		case format == "xml" && xmlFiles == 0:
			result["error"] = "docs_output_empty"
			result["detail"] = "the engine produced no XML class reference"
		}

	}
	if failure := runFailureCode(run); failure != "" && resultError(result) == "" {
		result["error"] = failure
	}
	return finishArtifactPublication(result, run, stage, outAbs, force)
}

// maxDocsScanDepth bounds how deep the destination scan follows linked
// directories while looking for existing XML documentation.
const maxDocsScanDepth = 12

// xmlFilesFollowingLinks returns every XML file the destination scan sees. Linked
// directories are followed (visited set plus depth cap), so counting and
// verification observe exactly the same files.
func xmlFilesFollowingLinks(root string, maxDepth int) ([]string, error) {
	if _, err := os.Stat(root); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	files := []string{}
	visited := map[string]bool{}
	var walk func(dir string, depth int) error
	walk = func(dir string, depth int) error {
		if depth > maxDepth {
			// An incomplete scan must never be reported as a clean destination.
			return fmt.Errorf("directory tree exceeds the %d-level scan depth at %s", maxDepth, filepath.ToSlash(dir))
		}
		resolved, err := filepath.EvalSymlinks(dir)
		if err != nil {
			return err
		}
		if visited[resolved] {
			return nil
		}
		visited[resolved] = true
		entries, err := os.ReadDir(dir)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			full := filepath.Join(dir, entry.Name())
			info, infoErr := os.Stat(full)
			if infoErr != nil {
				return infoErr
			}
			if info.IsDir() {
				if err := walk(full, depth+1); err != nil {
					return err
				}
				continue
			}
			if strings.EqualFold(filepath.Ext(entry.Name()), ".xml") {
				files = append(files, full)
			}
		}
		return nil
	}
	if err := walk(root, 0); err != nil {
		return nil, err
	}
	return files, nil
}

// fileSnapshot records the state of one destination path.
type fileSnapshot struct {
	Exists bool
	IsDir  bool
	Bytes  int64
	Mod    time.Time
}

func snapshotFile(path string) fileSnapshot {
	info, err := os.Stat(path)
	if err != nil {
		return fileSnapshot{}
	}
	return fileSnapshot{Exists: true, IsDir: info.IsDir(), Bytes: info.Size(), Mod: info.ModTime()}
}

// resolveDestination follows links on an existing destination so guards apply to
// the directory Godot will actually traverse.
func resolveDestination(path string) (string, error) {
	resolved, err := filepath.EvalSymlinks(path)
	if err == nil {
		return resolved, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return path, nil
	}
	return "", err
}

// countXMLFilesFollowingLinks counts *.xml under root, following directory
// links (symlinks, junctions) so a linked documentation tree cannot hide
// existing XML from the guard. A missing root counts as empty.
func countXMLFilesFollowingLinks(root string, maxDepth int) (int, error) {
	files, err := xmlFilesFollowingLinks(root, maxDepth)
	return len(files), err
}

func dirSize(dir string) (int, int64, error) {
	files := 0
	var bytes int64
	err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			return nil
		}
		files++
		bytes += info.Size()
		return nil
	})
	return files, bytes, err
}

func runDocs(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: docs dump --out <dir> [--format xml|json] [--scripts]")
		return 2
	}
	switch args[0] {
	case "dump":
		params, err := parseDocsDumpArgs(args[1:])
		if err != nil {
			fmt.Fprintf(os.Stderr, "docs dump: %v\n", err)
			return 2
		}
		return runDocsDump(params)
	default:
		fmt.Fprintf(os.Stderr, "unknown docs subcommand %q (want dump)\n", args[0])
		return 2
	}
}

// parseClassDBDumpArgs parses: classdb dump --out DIR [--engine PATH]
func parseClassDBDumpArgs(args []string) (map[string]any, error) {
	params := map[string]any{"action": "dump"}
	for i := 0; i < len(args); i++ {
		switch {
		case isFlag(args[i], "--out"):
			value, next, err := takeFlagValue(args, i, "--out")
			if err != nil {
				return nil, err
			}
			params["out"] = value
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
	if stringParam(params, "out") == "" {
		return nil, fmt.Errorf("--out <dir> is required")
	}
	return params, nil
}

// runClassDBDump uses the hidden --class-db-json flag, which always writes into
// user://, so the engine runs with isolated user directories and Hera publishes
// the result. Generation and publication are reported separately: a copy that
// fails never turns into a success claim.
func runClassDBDump(params map[string]any) int {
	outAbs, err := filepath.Abs(stringParam(params, "out"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "classdb dump: %v\n", err)
		return 2
	}
	if err := os.MkdirAll(outAbs, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "classdb dump: %v\n", err)
		return 1
	}
	ref, ctx, cancel, err := runEngineCommand(engineProjectRunTimeout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "classdb dump: %v\n", err)
		return 1
	}
	defer cancel()

	isolated, err := os.MkdirTemp("", "hera-classdb-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "classdb dump: %v\n", err)
		return 1
	}
	defer os.RemoveAll(isolated)

	env := append(os.Environ(), isolatedUserEnv(isolated)...)
	// Version identity is collected before the workload consumes the deadline.
	version, _ := readEngineVersion(ctx, ref.Path)
	run := runEngineBounded(ctx, ref.Path, []string{"--headless", "--class-db-json", "--quit"}, isolated, env, engineOutputLimit)

	files, publishErr := publishClassDBFiles(isolated, outAbs)
	failed := 0
	for _, file := range files {
		if !file.OK {
			failed++
		}
	}
	result := mergeResult(map[string]any{
		"output_dir": filepath.ToSlash(outAbs), "files": files, "file_count": len(files),
		"failed_copies": failed,
		"engine":        engineIdentity(ref, version),
		"notes": []string{
			"the engine writes ClassDB JSON into user://; Hera ran it with an isolated user directory",
			"class_db_api.json covers runtime classes, class_db_api_editor.json adds editor-only classes",
			"files[] reports generation and publication separately; entry.ok is false when the copy failed",
		},
		"source": "engine_process",
	}, run)
	switch {
	case publishErr != nil:
		result["error"] = "class_db_publish_failed"
		result["detail"] = publishErr.Error()
	case len(files) == 0:
		result["error"] = "class_db_json_missing"
		result["detail"] = "the engine did not write ClassDB JSON into its isolated user directory"
	case failed > 0:
		result["error"] = "class_db_copy_failed"
		result["detail"] = fmt.Sprintf("%d of %d files could not be published", failed, len(files))
	}
	return finishEngineResult(result, run)
}

// classDBFile reports one published ClassDB artifact.
type classDBFile struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	Bytes int    `json:"bytes"`
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

// publishClassDBFiles copies the engine's ClassDB JSON into outDir, replacing a
// destination only after the source was read successfully.
func publishClassDBFiles(sourceDir, outDir string) ([]classDBFile, error) {
	files := []classDBFile{}
	walkErr := filepath.WalkDir(sourceDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "class_db_api") || !strings.HasSuffix(name, ".json") {
			return nil
		}
		record := classDBFile{Name: name, Path: filepath.ToSlash(filepath.Join(outDir, name))}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			record.Error = "read: " + readErr.Error()
			files = append(files, record)
			return nil
		}
		if stat, statErr := os.Stat(filepath.Join(outDir, name)); statErr == nil && stat.IsDir() {
			record.Error = "destination is a directory"
			files = append(files, record)
			return nil
		}
		if writeErr := writeFileAtomic(filepath.Join(outDir, name), data); writeErr != nil {
			record.Error = "write: " + writeErr.Error()
			files = append(files, record)
			return nil
		}
		record.Bytes = len(data)
		record.OK = true
		files = append(files, record)
		return nil
	})
	return files, walkErr
}

// writeFileAtomic writes data through a temporary file in the destination
// directory so an existing file is never truncated before its replacement is
// ready.
func writeFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	temp, err := os.CreateTemp(dir, ".hera-publish-*")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	if _, err := temp.Write(data); err != nil {
		temp.Close()
		os.Remove(tempPath)
		return err
	}
	if err := temp.Close(); err != nil {
		os.Remove(tempPath)
		return err
	}
	if err := os.Rename(tempPath, path); err != nil {
		os.Remove(tempPath)
		return err
	}
	return nil
}

// isolatedUserEnv redirects every user-data root Godot may use, so the engine
// cannot write into the developer's real app data.
func isolatedUserEnv(dir string) []string {
	return []string{
		"HOME=" + dir,
		"USERPROFILE=" + dir,
		"APPDATA=" + dir,
		"XDG_DATA_HOME=" + dir,
		"XDG_CONFIG_HOME=" + dir,
	}
}
