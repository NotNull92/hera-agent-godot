package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/NotNull92/hera-agent-godot/internal/client"
)

func TestEvalLanguageAndArguments(t *testing.T) {
	t.Setenv("HERA_GODOT_EVAL_LANG", "")
	for _, tc := range []struct {
		args       []string
		lang, expr string
		invalid    bool
	}{
		{[]string{"1", "+", "2"}, "csharp", "1 + 2", false},
		{[]string{"--lang=gdscript", "Vector2(1,2)"}, "gdscript", "Vector2(1,2)", false},
		{[]string{"--lang", "csharp", "--", "--counter"}, "csharp", "--counter", false},
		{[]string{"--lang"}, "", "", true},
		{[]string{"--lang", "python", "1"}, "", "", true},
		{nil, "", "", true},
	} {
		got, err := parseEvalArgs(tc.args)
		if (err != nil) != tc.invalid {
			t.Fatalf("%v: %v", tc.args, err)
		}
		if err == nil && (got["lang"] != tc.lang || got["expr"] != tc.expr) {
			t.Fatalf("%v: %v", tc.args, got)
		}
	}
	t.Setenv("HERA_GODOT_EVAL_LANG", "gdscript")
	if got, _ := evalLanguage(""); got != "gdscript" {
		t.Fatal(got)
	}
	if got, _ := evalLanguage("csharp"); got != "csharp" {
		t.Fatal(got)
	}
	t.Setenv("HERA_GODOT_EVAL_LANG", "invalid")
	if _, err := evalLanguage(""); err == nil {
		t.Fatal("accepted invalid environment language")
	}
}

func TestPrepareEvalBatchAndUnavailable(t *testing.T) {
	t.Setenv("HERA_GODOT_EVAL_LANG", "gdscript")
	p := map[string]any{"expr": "1+1"}
	batch := map[string]any{"commands": []any{map[string]any{"tool": "eval", "params": p}}}
	if err := prepareEvalRequests(nil, "batch", batch); err != nil || p["lang"] != "gdscript" {
		t.Fatalf("%v %v", p, err)
	}
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Write([]byte(`{"ok":false,"error":"csharp_unavailable: use --lang gdscript"}`))
	}))
	defer server.Close()
	err := prepareEvalRequests(client.New(server.URL), "eval", map[string]any{"expr": "1+1", "lang": "csharp"})
	if err == nil || !strings.HasPrefix(err.Error(), "csharp_unavailable:") || calls != 1 {
		t.Fatalf("%v, calls=%d", err, calls)
	}
	if err := prepareEvalRequests(nil, "eval", map[string]any{"expr": "1", "lang": 42}); err == nil {
		t.Fatal("accepted non-string language")
	}
}

func TestCSharpHostOverride(t *testing.T) {
	old := csharpHostPath
	t.Cleanup(func() { csharpHostPath = old })
	path := filepath.Join(t.TempDir(), "host.dll")
	if err := os.WriteFile(path, []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	csharpHostPath = ""
	t.Setenv("HERA_GODOT_CSHARP_HOST", path)
	if got, err := resolveCSharpHost(); err != nil || got != path {
		t.Fatalf("%s %v", got, err)
	}
	csharpHostPath = path + ".missing"
	if _, err := resolveCSharpHost(); err == nil {
		t.Fatal("flag did not override environment")
	}
}

func TestCSharpEvalMissingAssembly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.dll")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"ok": true, "data": map[string]any{
			"project_reference": path, "project_mvid": "revision", "references": []string{path},
		}})
	}))
	defer server.Close()
	err := prepareEvalRequests(client.New(server.URL), "eval", map[string]any{"expr": "1", "lang": "csharp"})
	if err == nil || !strings.HasPrefix(err.Error(), "csharp_build_required:") {
		t.Fatal(err)
	}
}

func TestEvalTransportErrorKeepsStructuredCode(t *testing.T) {
	server := startContractEditor(t, map[string]string{"eval:csharp-context": "not JSON"})
	defer server.Close()
	installContractHome(t, contractServerPort(t, server), true)
	stdout, stderr, code := captureContractRun([]string{"eval", "--lang=csharp", "1"})
	var result map[string]any
	if err := json.Unmarshal([]byte(stdout), &result); err != nil || code != 1 || result["error"] != "csharp_compile_failed" {
		t.Fatalf("%d %s %s %v", code, stdout, stderr, err)
	}
	err := prepareEvalRequests(nil, "eval", map[string]any{"lang": 42})
	var coded *codedError
	if !errors.As(fmt.Errorf("wrapped: %w", err), &coded) || coded.Code != "invalid_language" {
		t.Fatalf("lost code: %v", err)
	}
}
