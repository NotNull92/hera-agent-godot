package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScriptValidationOutputBounded(t *testing.T) {
	output := &scriptValidationOutput{}
	text := bytes.Repeat([]byte("x"), 64*1024+1)
	if n, err := output.Write(text); err != nil || n != len(text) {
		t.Fatalf("write = %d, %v", n, err)
	}
	if _, err := output.Write(text); err != nil {
		t.Fatal(err)
	}
	if len(output.text) != 64*1024 || !output.truncated {
		t.Fatalf("length=%d truncated=%v", len(output.text), output.truncated)
	}
}

func TestScriptValidateEvidenceFlag(t *testing.T) {
	p, err := parseScriptArgs([]string{"validate", "res://probe.gd", "--evidence"})
	if err != nil || p["evidence"] != true {
		t.Fatalf("evidence flag: %v, %v", p, err)
	}
}

func TestScriptValidateNativeGodot(t *testing.T) {
	executable := os.Getenv("HERA_TEST_GODOT")
	if executable == "" {
		t.Skip("set HERA_TEST_GODOT to the direct Godot executable")
	}
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "project.godot"), []byte("config_version=5\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "base.gd"), []byte("extends Node\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	context, err := json.Marshal(map[string]any{"ok": true, "data": map[string]any{
		"executable": executable, "project_path": project, "path": "res://probe.gd",
	}})
	if err != nil {
		t.Fatal(err)
	}
	server := startContractEditor(t, map[string]string{"script:validate-context": string(context)})
	defer server.Close()
	installContractHome(t, contractServerPort(t, server), true)
	for _, tc := range []struct {
		name, source string
		valid        bool
	}{
		{"valid", "extends Node\n", true},
		{"changed on disk", "extends Node\nfunc broken(\n", false},
		{"type error", "extends Node\nvar value: int = Vector2.ZERO\n", false},
		{"fixed on disk", "extends Node\nvar value: int = 2\n", true},
		{"relative dependency", "extends \"base.gd\"\n", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := os.WriteFile(filepath.Join(project, "probe.gd"), []byte(tc.source), 0o600); err != nil {
				t.Fatal(err)
			}
			stdout, stderr, code := captureContractRun([]string{"script", "validate", "res://probe.gd"})
			var result struct {
				Valid    bool   `json:"valid"`
				Output   string `json:"output"`
				ExitCode int    `json:"exit_code"`
			}
			if err := json.Unmarshal([]byte(stdout), &result); err != nil {
				t.Fatalf("decode: %v; stdout=%q stderr=%q", err, stdout, stderr)
			}
			if result.Valid != tc.valid || (code == 0) != tc.valid || (result.ExitCode == 0) != tc.valid {
				t.Fatalf("code=%d result=%+v stderr=%q", code, result, stderr)
			}
			if !tc.valid && !strings.Contains(result.Output, "Parse Error") {
				t.Fatalf("missing native error detail: %q", result.Output)
			}
		})
	}
	t.Run("changed during validation", func(t *testing.T) {
		source := "extends Node\nstatic func _static_init() -> void:\n\tvar file := FileAccess.open(\"res://probe.gd\", FileAccess.WRITE)\n\tfile.store_string(\"extends Node\\n# changed\\n\")\n\tfile.close()\n"
		if err := os.WriteFile(filepath.Join(project, "probe.gd"), []byte(source), 0o600); err != nil {
			t.Fatal(err)
		}
		stdout, stderr, code := captureContractRun([]string{"script", "validate", "res://probe.gd", "--evidence"})
		var result map[string]any
		if err := json.Unmarshal([]byte(stdout), &result); err != nil {
			t.Fatalf("decode %v: %s %s", err, stdout, stderr)
		}
		if code != 1 || result["valid"] != false || result["changed_during_validation"] != true || result["source"] != "disk" {
			t.Fatalf("code=%d result=%v stderr=%s", code, result, stderr)
		}
	})
	t.Run("bounded static initializer", func(t *testing.T) {
		source := "extends Node\nstatic func _static_init() -> void:\n\twhile true:\n\t\tpass\n"
		if err := os.WriteFile(filepath.Join(project, "probe.gd"), []byte(source), 0o600); err != nil {
			t.Fatal(err)
		}
		stdout, stderr, code := captureContractRun([]string{"--timeout", "500", "script", "validate", "res://probe.gd"})
		var result struct {
			Valid    bool `json:"valid"`
			TimedOut bool `json:"timed_out"`
		}
		if err := json.Unmarshal([]byte(stdout), &result); err != nil || code != 1 || result.Valid || !result.TimedOut {
			t.Fatalf("code=%d result=%+v decode=%v stderr=%q stdout=%q", code, result, err, stderr, stdout)
		}
	})
}
