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

var csharpHostPath string

func resolveCSharpHost() (string, error) {
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
			return "", fmt.Errorf("csharp_host_unavailable: host file not found")
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
	return "", fmt.Errorf("csharp_host_unavailable: install the compiler helper or set --csharp-host / HERA_GODOT_CSHARP_HOST")
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
		return fmt.Errorf("invalid_language: eval lang must be a string")
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
		return fmt.Errorf("parse_error: eval requires an expression")
	}
	resp, err := c.Post("eval", map[string]any{"lang": "csharp", "action": "csharp-context"})
	if err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("%s", resp.Error)
	}
	data, ok := resp.Data.(map[string]any)
	if !ok {
		return fmt.Errorf("csharp_build_required: invalid loader context")
	}
	project, _ := data["project_reference"].(string)
	mvid, _ := data["project_mvid"].(string)
	refs, ok := data["references"].([]any)
	if !ok || len(refs) == 0 || project == "" || mvid == "" {
		return fmt.Errorf("csharp_build_required: incomplete loader context")
	}
	if !fileExists(project) {
		return fmt.Errorf("csharp_build_required: project assembly is missing; build and reload the project")
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
			return fmt.Errorf("dotnet_unavailable: %w", err)
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
		return fmt.Errorf("csharp_compile_incomplete: worker unfinished; retained staging at %s", stage)
	}
	if run.TimedOut {
		return fmt.Errorf("csharp_compile_timeout: compiler exceeded deadline")
	}
	var compiled struct {
		OK          bool     `json:"ok"`
		Assembly    string   `json:"assembly"`
		Error       string   `json:"error"`
		Diagnostics []string `json:"diagnostics"`
	}
	if run.Truncated || json.Unmarshal([]byte(run.Output), &compiled) != nil {
		return fmt.Errorf("csharp_compile_failed: compiler did not return a complete response")
	}
	if !compiled.OK {
		return fmt.Errorf("%s: %s", compiled.Error, strings.Join(compiled.Diagnostics, "; "))
	}
	if runFailed(run) || compiled.Assembly == "" {
		return fmt.Errorf("csharp_compile_failed: compiler failed")
	}
	params["assembly"], params["project_mvid"] = compiled.Assembly, mvid
	return nil
}
