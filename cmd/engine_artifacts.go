package cmd

import (
	"fmt"
	"github.com/NotNull92/hera-agent-godot/internal/engineproc"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// stageArtifacts keeps generation separate from existing destination files.
func stageArtifacts(destination string) (string, error) {
	if err := os.MkdirAll(destination, 0o755); err != nil {
		return "", err
	}
	return os.MkdirTemp(destination, ".hera-run-")
}

// publishArtifacts never follows output links outside the destination. Publication
// is per-file, not a transaction; on error the caller retains remaining artifacts.
func publishArtifacts(stage, destination string, replace bool) (int, error) {
	files := []string{}
	err := filepath.WalkDir(stage, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("non-regular generated artifact: %s", path)
		}
		rel, err := filepath.Rel(stage, path)
		if err != nil {
			return err
		}
		files = append(files, rel)
		return nil
	})
	if err != nil {
		return 0, err
	}
	root, err := os.OpenRoot(destination)
	if err != nil {
		return 0, err
	}
	defer root.Close()
	stageName, err := filepath.Rel(destination, stage)
	if err != nil || filepath.Dir(stageName) != "." {
		return 0, fmt.Errorf("staging directory must be a child of destination")
	}
	for _, rel := range files {
		if err := root.MkdirAll(filepath.Dir(rel), 0o755); err != nil {
			return 0, err
		}
		info, err := root.Lstat(rel)
		if err != nil && !os.IsNotExist(err) {
			return 0, err
		}
		if err == nil && (!replace || !info.Mode().IsRegular()) {
			return 0, fmt.Errorf("destination already exists or is not a regular file: %s", rel)
		}
	}
	published := 0
	for _, rel := range files {
		source := filepath.Join(stageName, rel)
		if replace {
			err = root.Rename(source, rel)
		} else {
			// Exclusive creation also works on filesystems without hard links.
			err = copyArtifactExclusive(root, source, rel)
		}
		if err != nil {
			return published, err
		}
		published++
	}
	return published, root.RemoveAll(stageName)
}

func finishArtifactPublication(result map[string]any, run engineproc.Result, stage, destination string, replace bool) int {
	result["artifact_source"] = "isolated_run_directory"
	result["published"] = false
	result["staging_dir"] = filepath.ToSlash(stage)
	if failure := runFailureCode(run); failure != "" {
		result["error"] = failure
	}
	if resultError(result) == "" {
		count, err := publishArtifacts(stage, destination, replace)
		result["files_published"] = count
		if err != nil {
			result["error"], result["detail"] = "artifact_publish_failed", err.Error()
		} else {
			result["published"] = true
			delete(result, "staging_dir")
		}
	}
	return finishEngineResult(result, run)
}

func copyArtifactExclusive(root *os.Root, source, destination string) error {
	input, err := root.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	info, err := input.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("generated artifact is not regular: %s", source)
	}
	output, err := root.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, info.Mode().Perm())
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(output, input)
	closeErr := output.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}
