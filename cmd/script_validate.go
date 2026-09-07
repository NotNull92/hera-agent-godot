package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
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
	resp, err := c.Post("script", map[string]any{"action": "validate-context", "path": params["path"]})
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
	command := exec.CommandContext(ctx, executable, "--headless", "--path", project, "--check-only", "--script", path)
	command.WaitDelay = time.Second
	output := &scriptValidationOutput{}
	command.Stdout = output
	command.Stderr = output
	err = command.Run()
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
	if code := printData(&protocol.Response{OK: true, Data: result}); code != 0 {
		return code
	}
	if err != nil {
		return 1
	}
	return 0
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
