package cmd

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/NotNull92/hera-agent-godot/internal/protocol"
)

func runScriptValidate(params map[string]any) int {
	c, err := dialMutationEditor()
	if err != nil {
		fmt.Fprintf(os.Stderr, "script validate: %v\n", err)
		return 1
	}
	request := map[string]any{"action": "validate-context", "path": params["path"]}
	if params["evidence"] == true {
		request["evidence"] = true
	}
	resp, err := c.Post("script", request)
	if err != nil {
		fmt.Fprintf(os.Stderr, "script validate: %v\n", err)
		return 1
	}
	if !resp.OK {
		fmt.Fprintf(os.Stderr, "script validate: %s\n", resp.Error)
		return 1
	}
	data, ok := resp.Data.(map[string]any)
	if !ok {
		fmt.Fprintln(os.Stderr, "script validate: invalid validation context")
		return 1
	}
	executable, _ := data["executable"].(string)
	project, _ := data["project_path"].(string)
	path, _ := data["path"].(string)
	if executable == "" || project == "" || path == "" || path != params["path"] {
		fmt.Fprintln(os.Stderr, "script validate: incomplete validation context")
		return 1
	}
	timeout := requestTimeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	var before, after string
	var beforeErr, afterErr error
	diskPath := filepath.Join(project, filepath.FromSlash(strings.TrimPrefix(path, "res://")))
	if params["evidence"] == true {
		before, beforeErr = scriptDiskHash(diskPath)
	}
	started := time.Now().UTC().Format(time.RFC3339Nano)
	command := exec.CommandContext(ctx, executable, "--headless", "--path", project, "--check-only", "--script", path)
	command.WaitDelay = time.Second
	output := &scriptValidationOutput{}
	command.Stdout = output
	command.Stderr = output
	err = command.Run()
	if params["evidence"] == true {
		after, afterErr = scriptDiskHash(diskPath)
	}
	exitCode := -1
	if command.ProcessState != nil {
		exitCode = command.ProcessState.ExitCode()
	}
	result := map[string]any{
		"path": path, "valid": err == nil, "exit_code": exitCode,
		"output": strings.TrimSpace(string(output.text)), "output_truncated": output.truncated,
		"timed_out": ctx.Err() != nil,
	}
	if err != nil && exitCode == -1 {
		result["error"] = err.Error()
	}
	if params["evidence"] == true {
		version, engineAvailable := data["engine_version"].(map[string]any)
		hashAvailable := beforeErr == nil && afterErr == nil
		available := hashAvailable && engineAvailable && version["string"] != nil
		changed := hashAvailable && before != after
		result["source"] = "disk"
		result["engine"] = map[string]any{"path": executable, "version": data["engine_version"], "version_source": "editor_context"}
		result["sha256_before"], result["sha256_after"] = before, after
		result["changed_during_validation"] = changed
		result["started_at"], result["finished_at"] = started, time.Now().UTC().Format(time.RFC3339Nano)
		result["evidence"] = map[string]any{"available": available, "atomic_snapshot": false}
		for _, source := range []string{"editor_buffer", "loaded_script", "running_code"} {
			result[source] = map[string]any{"available": false, "reason": "not observed by disk validation"}
		}
		if !available || changed {
			result["valid"] = false
			result["error"] = "source_changed: disk file changed during validation"
			if !hashAvailable {
				result["error"] = fmt.Sprintf("evidence_unavailable: disk hash before=%v after=%v", beforeErr, afterErr)
			} else if !available {
				result["error"] = "evidence_unavailable: selected engine version was not reported"
			}
		}
	}
	if code := printData(&protocol.Response{OK: true, Data: result}); code != 0 {
		return code
	}
	if result["valid"] != true {
		return 1
	}
	return 0
}

func scriptDiskHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	const limit = 64 * 1024 * 1024
	n, err := io.Copy(hash, io.LimitReader(file, limit+1))
	if err != nil {
		return "", err
	}
	if n > limit {
		return "", fmt.Errorf("script exceeds 64 MiB hash limit")
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

type scriptValidationOutput struct {
	text      []byte
	truncated bool
}

func (o *scriptValidationOutput) Write(p []byte) (int, error) {
	const limit = 64 * 1024
	remaining := limit - len(o.text)
	if len(p) > remaining {
		o.text = append(o.text, p[:remaining]...)
		o.truncated = true
	} else {
		o.text = append(o.text, p...)
	}
	return len(p), nil
}
