package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/NotNull92/hera-agent-godot/internal/protocol"
)

type csharpDiagnostic struct {
	Path     string `json:"path"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	Severity string `json:"severity"`
	Code     string `json:"code"`
	Message  string `json:"message"`
}

var csharpDiagnosticPattern = regexp.MustCompile(`^\s*(.+)\((\d+),(\d+)(?:,\d+,\d+)?\):\s*(error|warning)\s+([A-Za-z]+\d+):\s*(.*?)(?:\s+\[[^\]]+\])?$`)

func csharpDiagnostics(output, project, requested string) ([]csharpDiagnostic, int) {
	items := []csharpDiagnostic{}
	other := 0
	seen := map[string]bool{}
	for _, line := range strings.Split(output, "\n") {
		m := csharpDiagnosticPattern.FindStringSubmatch(strings.TrimSpace(line))
		if m == nil {
			continue
		}
		path := filepath.Clean(m[1])
		if !filepath.IsAbs(path) {
			path = filepath.Join(project, path)
		}
		key := strings.Join(m[1:], "\x00")
		if seen[key] {
			continue
		}
		seen[key] = true
		if !sameDiskFile(path, requested) {
			if m[4] == "error" {
				other++
			}
			continue
		}
		row, _ := strconv.Atoi(m[2])
		column, _ := strconv.Atoi(m[3])
		items = append(items, csharpDiagnostic{filepath.ToSlash(path), row, column, m[4], m[5], m[6]})
	}
	return items, other
}

func sameDiskFile(a, b string) bool {
	ai, ae := os.Stat(a)
	bi, be := os.Stat(b)
	if ae == nil && be == nil {
		return os.SameFile(ai, bi)
	}
	return filepath.Clean(a) == filepath.Clean(b)
}

func csharpProject(project string) (string, error) {
	files, err := filepath.Glob(filepath.Join(project, "*.csproj"))
	if err != nil || len(files) != 1 {
		return "", fmt.Errorf("csproj_not_found: expected exactly one .csproj in project root")
	}
	return files[0], nil
}

func csharpBuildPaths(stage string) []string {
	// Explicit concrete paths are necessary: Godot SDK overrides the base paths.
	return []string{
		"-p:BaseOutputPath=" + filepath.Join(stage, "bin") + string(os.PathSeparator),
		"-p:OutputPath=" + filepath.Join(stage, "bin", "Debug") + string(os.PathSeparator),
		"-p:BaseIntermediateOutputPath=" + filepath.Join(stage, "obj") + string(os.PathSeparator),
		"-p:IntermediateOutputPath=" + filepath.Join(stage, "obj", "Debug") + string(os.PathSeparator),
		"-p:MSBuildProjectExtensionsPath=" + filepath.Join(stage, "obj") + string(os.PathSeparator),
	}
}

func csharpValidateTimeout() time.Duration {
	if requestTimeout != 0 {
		return requestTimeout
	}
	return 60 * time.Second
}

func runCSharpValidate(project, path string, evidence bool) int {
	result := map[string]any{"path": path, "lang": "csharp", "valid": false, "source": "disk", "validation_scope": "project_build"}
	fail := func(code, detail string) int {
		result["error"] = code
		result["detail"] = detail
		return failStructured(result)
	}
	csproj, err := csharpProject(project)
	if err != nil {
		return fail("csproj_not_found", err.Error())
	}
	ctx, cancel := context.WithTimeout(context.Background(), csharpValidateTimeout())
	defer cancel()
	dotnet, version, state := dotnetSDK(ctx)
	if state != "supported" {
		return fail("dotnet_unavailable", "dotnet --version did not establish SDK readiness")
	}
	result["dotnet"] = map[string]any{"path": dotnet, "sdk_version": version}
	stage, err := os.MkdirTemp("", "hera-csharp-validate-")
	if err != nil {
		return fail("artifact_stage_failed", err.Error())
	}
	keep := false
	defer func() {
		if !keep {
			_ = os.RemoveAll(stage)
		}
	}()
	paths := csharpBuildPaths(stage)
	run := func(args []string) engineRunResult {
		r := runEngineBounded(ctx, dotnet, append(args, paths...), project, append(os.Environ(), "DOTNET_CLI_UI_LANGUAGE=en", "DOTNET_NOLOGO=1", "DOTNET_GENERATE_ASPNET_CERTIFICATE=false"), engineOutputLimit)
		if r.Abandoned || r.CleanupPending {
			keep = true
			result["staging_dir"] = filepath.ToSlash(stage)
		}
		return r
	}
	query := run([]string{"msbuild", csproj, "-getItem:Compile,ProjectReference", "-nologo"})
	if runFailed(query) || query.Truncated {
		mergeResult(result, query)
		if query.TimedOut {
			return fail("build_timeout", "compile-input query exceeded deadline")
		}
		return fail("build_context_failed", "could not observe compile inputs")
	}
	var inputs struct {
		Items map[string][]struct{ FullPath string }
	}
	if json.Unmarshal([]byte(query.Output), &inputs) != nil {
		return fail("build_context_failed", "MSBuild did not return compile inputs")
	}
	// Shared global output paths must never collide between project references.
	if len(inputs.Items["ProjectReference"]) != 0 {
		return fail("project_references_unsupported", "isolated validation currently requires a project without ProjectReference items")
	}
	disk := filepath.Join(project, filepath.FromSlash(strings.TrimPrefix(path, "res://")))
	included := false
	for _, item := range inputs.Items["Compile"] {
		if sameDiskFile(item.FullPath, disk) {
			included = true
		}
	}
	if !included {
		return fail("script_not_compiled", "requested file is not a Compile item")
	}
	before, err := scriptDiskHash(disk)
	if err != nil {
		return fail("evidence_unavailable", err.Error())
	}
	result["disk_hash"] = before
	restored := run([]string{"restore", csproj, "--nologo", "--no-dependencies"})
	if runFailed(restored) {
		mergeResult(result, restored)
		code := "restore_failed"
		if restored.TimedOut {
			code = "build_timeout"
		}
		return fail(code, "project restore failed; MSBuild targets and generators may execute code")
	}
	built := run([]string{"build", csproj, "--no-restore", "--nologo", "-v:minimal", "-clp:NoSummary", "-p:BuildProjectReferences=false"})
	mergeResult(result, built)
	items, other := csharpDiagnostics(built.Output, project, disk)
	result["diagnostics"], result["other_files_errors"] = items, other
	after, hashErr := scriptDiskHash(disk)
	result["changed_during_validation"] = before != after || hashErr != nil
	result["valid"] = !runFailed(built) && !built.Truncated && hashErr == nil && before == after
	if evidence {
		result["sha256_before"], result["sha256_after"] = before, after
		result["evidence"] = map[string]any{"available": hashErr == nil, "atomic_snapshot": false}
	}
	if result["valid"] != true {
		code := "build_failed"
		if built.TimedOut {
			code = "build_timeout"
		} else if before != after || hashErr != nil {
			code = "source_changed"
		} else if built.Truncated {
			code = "diagnostics_incomplete"
		}
		return fail(code, "C# project compilation did not verify the requested disk revision")
	}
	return printData(&protocol.Response{OK: true, Data: result})
}
