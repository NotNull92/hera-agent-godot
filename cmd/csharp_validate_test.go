package cmd

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/NotNull92/hera-agent-godot/internal/engineproc"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestCSharpValidateTimeout(t *testing.T) {
	previous := requestTimeout
	t.Cleanup(func() { requestTimeout = previous })
	for _, timeout := range []time.Duration{0, 5 * time.Second, 90 * time.Second} {
		requestTimeout = timeout
		want := timeout
		if want == 0 {
			want = 60 * time.Second
		}
		if got := csharpValidateTimeout(); got != want {
			t.Fatalf("timeout %v: got %v, want %v", timeout, got, want)
		}
	}
}

func TestCSharpDiagnostics(t *testing.T) {
	project := t.TempDir()
	path := filepath.Join(project, "한글 file.cs")
	output := path + "(2,7): error CS1002: ; expected [Project.csproj]\r\n" +
		path + "(4,1): warning CS0168: unused variable [Project.csproj]\r\n" +
		filepath.Join(project, "Other.cs") + "(1,2): error CS0103: missing [Project.csproj]\r\n"
	items, other := csharpDiagnostics(output+output, project, path)
	if len(items) != 2 || other != 1 || items[0].Line != 2 || items[0].Column != 7 || items[0].Code != "CS1002" || items[1].Severity != "warning" {
		t.Fatalf("diagnostics=%+v other=%d", items, other)
	}
}

func TestCSharpBuildIsolationArguments(t *testing.T) {
	stage := t.TempDir()
	args := csharpBuildPaths(stage)
	if len(args) != 5 {
		t.Fatal(args)
	}
	for _, arg := range args {
		if !strings.Contains(arg, stage) {
			t.Fatal(arg)
		}
	}
	if _, err := csharpProject(stage); err == nil {
		t.Fatal("missing project accepted")
	}
}

func TestCSharpValidateLive(t *testing.T) {
	seed := os.Getenv("HERA_TEST_CSHARP_PROJECT")
	if seed == "" {
		t.Skip("set HERA_TEST_CSHARP_PROJECT to an owned disposable spike project")
	}
	project := t.TempDir()
	write := func(name, text string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(project, name), []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	config, err := os.ReadFile(filepath.Join(seed, "NuGet.Config"))
	if err != nil {
		t.Fatal(err)
	}
	write("NuGet.Config", string(config))
	write("Probe.csproj", `<Project Sdk="Godot.NET.Sdk/4.7.2"><PropertyGroup><TargetFramework>net10.0</TargetFramework></PropertyGroup></Project>`)
	write("Probe.cs", `using Godot; public partial class Probe : Node {}`)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	dotnet, _, state := dotnetSDK(ctx)
	if state != "supported" {
		t.Fatal(state)
	}
	baseline := engineproc.Run(ctx, dotnet, []string{"build", "Probe.csproj", "--nologo"}, project, nil, engineOutputLimit)
	if runFailed(baseline) {
		t.Fatal(baseline.Output)
	}
	snapshot := func() map[string]string {
		values := map[string]string{}
		err := filepath.WalkDir(filepath.Join(project, ".godot"), func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				return nil
			}
			b, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			st, err := entry.Info()
			if err != nil {
				return err
			}
			values[path] = fmt.Sprintf("%x:%d", sha256.Sum256(b), st.ModTime().UnixNano())
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		return values
	}
	before := snapshot()
	response, _ := json.Marshal(map[string]any{"ok": true, "data": map[string]any{"executable": "fixture-editor", "project_path": project, "path": "res://Probe.cs"}})
	server := startContractEditor(t, map[string]string{"script:validate-context": string(response)})
	defer server.Close()
	installContractHome(t, contractServerPort(t, server), true)
	for _, tc := range []struct {
		name, probe, other               string
		wantExit, ownErrors, otherErrors int
	}{
		{"clean", `using Godot; public partial class Probe : Node {}`, "", 0, 0, 0},
		{"own error", `using Godot; public partial class Probe : Node { public int X => missing; }`, "", 1, 1, 0},
		{"other error", `using Godot; public partial class Probe : Node {}`, `public class Other { public int X => missing; }`, 1, 0, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			installContractHome(t, contractServerPort(t, server), true)
			write("Probe.cs", tc.probe)
			write("Other.cs", tc.other)
			out, stderr, code := captureContractRun([]string{"--timeout", "60000", "script", "validate", "res://Probe.cs", "--evidence"})
			var data struct {
				Valid       bool
				Diagnostics []csharpDiagnostic
				Other       int `json:"other_files_errors"`
			}
			if err := json.Unmarshal([]byte(out), &data); err != nil {
				t.Fatal(out, stderr, err)
			}
			if code != tc.wantExit || len(data.Diagnostics) != tc.ownErrors || data.Other != tc.otherErrors || data.Valid != (tc.wantExit == 0) {
				t.Fatal(out, stderr, code)
			}
			if !reflect.DeepEqual(before, snapshot()) {
				t.Fatal("editor outputs changed")
			}
		})
	}
}
