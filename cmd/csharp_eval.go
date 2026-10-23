package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/NotNull92/hera-agent-godot/internal/client"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type codedError struct {
	Code string
	Err  error
}

func (e *codedError) Error() string { return e.Code + ": " + e.Err.Error() }
func (e *codedError) Unwrap() error { return e.Err }

var csharpHostPath string

func resolveCSharpHost() (string, error) {
	// Build csharp-host/ separately; source checkouts do not imply an installed helper.
	path := csharpHostPath
	if path == "" {
		path = os.Getenv("HERA_GODOT_CSHARP_HOST")
	}
	if path != "" {
		abs, err := filepath.Abs(path)
		if err != nil {
			return "", err
		}
		if !fileExists(abs) {
			return "", &codedError{Code: "csharp_host_unavailable", Err: fmt.Errorf("host file not found; run hera-godot csharp-host install")}
		}
		return abs, nil
	}
	exe, _ := os.Executable()
	home, _ := os.UserHomeDir()
	for _, path := range []string{filepath.Join(filepath.Dir(exe), "csharp", "Hera.CSharpHost.dll"), filepath.Join(home, ".hera-agent-godot", "csharp", "Hera.CSharpHost.dll")} {
		if fileExists(path) {
			return path, nil
		}
	}
	return "", &codedError{Code: "csharp_host_unavailable", Err: fmt.Errorf("run hera-godot csharp-host install or set --csharp-host / HERA_GODOT_CSHARP_HOST")}
}

func prepareEvalRequests(c *client.Client, tool string, params map[string]any) error {
	if tool == "batch" {
		var first error
		forEachBatchChild(params, func(child string, p map[string]any) {
			if first == nil {
				first = prepareEvalRequests(c, child, p)
			}
		})
		return first
	}
	if tool != "eval" || params == nil {
		return nil
	}
	explicit, ok := params["lang"].(string)
	if _, exists := params["lang"]; exists && !ok {
		return &codedError{Code: "invalid_language", Err: fmt.Errorf("eval lang must be a string")}
	}
	lang, err := evalLanguage(explicit)
	if err != nil {
		return err
	}
	params["lang"] = lang
	if lang == "gdscript" {
		return nil
	}
	expression, ok := params["expr"].(string)
	if !ok || strings.TrimSpace(expression) == "" {
		return &codedError{Code: "parse_error", Err: fmt.Errorf("eval requires an expression")}
	}
	resp, err := c.Post("eval", map[string]any{"lang": "csharp", "action": "csharp-context"})
	if err != nil {
		return &codedError{Code: "csharp_compile_failed", Err: err}
	}
	if !resp.OK {
		return fmt.Errorf("%s", resp.Error)
	}
	data, ok := resp.Data.(map[string]any)
	if !ok {
		return &codedError{Code: "csharp_build_required", Err: fmt.Errorf("invalid loader context")}
	}
	project, _ := data["project_reference"].(string)
	mvid, _ := data["project_mvid"].(string)
	refs, ok := data["references"].([]any)
	if !ok || len(refs) == 0 || project == "" || mvid == "" {
		return &codedError{Code: "csharp_build_required", Err: fmt.Errorf("incomplete loader context")}
	}
	if !fileExists(project) {
		return &codedError{Code: "csharp_build_required", Err: fmt.Errorf("project assembly is missing; build and reload the project")}
	}
	references := []string{}
	for _, ref := range refs {
		value, ok := ref.(string)
		if !ok || value == "" {
			return fmt.Errorf("invalid compiler reference")
		}
		references = append(references, value)
	}
	siblings, _ := filepath.Glob(filepath.Join(filepath.Dir(project), "*.dll"))
	references = append(references, siblings...)
	host, err := resolveCSharpHost()
	if err != nil {
		return err
	}
	stage, err := os.MkdirTemp("", "hera-csharp-eval-")
	if err != nil {
		return err
	}
	keep := false
	defer func() {
		if !keep {
			_ = os.RemoveAll(stage)
		}
	}()
	request, _ := json.Marshal(map[string]any{"expression": expression, "references": references, "project_reference": project, "project_mvid": mvid})
	requestPath := filepath.Join(stage, "request.json")
	if err = os.WriteFile(requestPath, request, 0600); err != nil {
		return err
	}
	executable := host
	args := []string{"--request", requestPath}
	if strings.EqualFold(filepath.Ext(host), ".dll") {
		executable, err = exec.LookPath("dotnet")
		if err != nil {
			return &codedError{Code: "dotnet_unavailable", Err: fmt.Errorf("%w", err)}
		}
		args = append([]string{host}, args...)
	}
	timeout := requestTimeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	run := runEngineBounded(ctx, executable, args, stage, nil, 768*1024)
	if run.Abandoned || run.CleanupPending {
		keep = true
		return &codedError{Code: "csharp_compile_incomplete", Err: fmt.Errorf("worker unfinished; retained staging at %s", stage)}
	}
	if run.TimedOut {
		return &codedError{Code: "csharp_compile_timeout", Err: fmt.Errorf("compiler exceeded deadline")}
	}
	var compiled struct {
		OK          bool     `json:"ok"`
		Assembly    string   `json:"assembly"`
		Error       string   `json:"error"`
		Diagnostics []string `json:"diagnostics"`
	}
	if run.Truncated || json.Unmarshal([]byte(run.Output), &compiled) != nil {
		return &codedError{Code: "csharp_compile_failed", Err: fmt.Errorf("compiler did not return a complete response")}
	}
	if !compiled.OK {
		return &codedError{Code: compiled.Error, Err: fmt.Errorf("%s", strings.Join(compiled.Diagnostics, "; "))}
	}
	if runFailed(run) || compiled.Assembly == "" {
		return &codedError{Code: "csharp_compile_failed", Err: fmt.Errorf("compiler failed")}
	}
	params["assembly"], params["project_mvid"] = compiled.Assembly, mvid
	return nil
}
