package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/NotNull92/hera-agent-godot/internal/client"
	"github.com/NotNull92/hera-agent-godot/internal/protocol"
)

func TestParseGameQADiagnoseArgs_acceptsGenericThresholds(t *testing.T) {
	// Given
	args := []string{"--lines", "80", "--max-errors", "1", "--max-warnings", "2", "--path", "user://qa.png"}

	// When
	options, err := parseGameQADiagnoseArgs(args)

	// Then
	if err != nil {
		t.Fatalf("parseGameQADiagnoseArgs error: %v", err)
	}
	if options.lines != 80 || options.maxErrors != 1 || options.maxWarnings != 2 || options.screenshotPath != "user://qa.png" {
		t.Fatalf("options = %+v", options)
	}
}

func TestExecuteGameQADiagnosis_passesWhenGenericRuntimeSignalsAreHealthy(t *testing.T) {
	// Given
	srv := newGameQADiagnoseServer(t, map[string]protocol.Response{
		"diagnostics":     {OK: true, Data: map[string]any{"error_count": 0, "warning_count": 0}},
		"game:instances":  {OK: true, Data: map[string]any{"instances": []any{map[string]any{"pid": 42}}}},
		"game:tree":       {OK: true, Data: map[string]any{"count": 3, "scene": "res://Main.tscn", "truncated": false}},
		"game:ui_tree":    {OK: true, Data: map[string]any{"count": 2, "truncated": false}},
		"game:screenshot": {OK: true, Data: map[string]any{"analysis": map[string]any{"nonblank": true, "low_detail": false, "possible_clipping": false}}},
	})
	t.Cleanup(srv.Close)

	// When
	data, ok := executeGameQADiagnosis(client.New(srv.URL), gameQADiagnoseOptions{lines: 40})

	// Then
	if !ok {
		t.Fatalf("diagnosis = %v, want success", data)
	}
	if data["ok"] != true || data["issues"] == nil {
		t.Fatalf("diagnosis data = %v", data)
	}
	checks, ok := data["checks"].([]map[string]any)
	if !ok || len(checks) != 5 {
		t.Fatalf("checks = %T %v, want five compact checks", data["checks"], data["checks"])
	}
}

func TestExecuteGameQADiagnosis_reportsClippingWithoutProjectSpecificRules(t *testing.T) {
	// Given
	srv := newGameQADiagnoseServer(t, map[string]protocol.Response{
		"diagnostics":     {OK: true, Data: map[string]any{"error_count": 0, "warning_count": 0}},
		"game:instances":  {OK: true, Data: map[string]any{"instances": []any{map[string]any{"pid": 42}}}},
		"game:tree":       {OK: true, Data: map[string]any{"count": 1, "scene": "res://Main.tscn", "truncated": false}},
		"game:ui_tree":    {OK: true, Data: map[string]any{"count": 0, "truncated": false}},
		"game:screenshot": {OK: true, Data: map[string]any{"analysis": map[string]any{"nonblank": true, "low_detail": false, "possible_clipping": true}}},
	})
	t.Cleanup(srv.Close)

	// When
	data, ok := executeGameQADiagnosis(client.New(srv.URL), gameQADiagnoseOptions{lines: 40})

	// Then
	if ok || data["ok"] != false {
		t.Fatalf("diagnosis = %v, want failure", data)
	}
	issues, ok := data["issues"].([]string)
	if !ok {
		t.Fatalf("issues = %T, want []string", data["issues"])
	}
	if !containsGameQAIssue(issues, "runtime screenshot may be clipped") {
		t.Fatalf("issues = %v, want clipping finding", issues)
	}
}

func TestExecuteGameQADiagnosis_reportsAllUnavailableRuntimeSignals(t *testing.T) {
	// Given
	srv := newGameQADiagnoseServer(t, map[string]protocol.Response{
		"diagnostics":     {OK: true, Data: map[string]any{"error_count": 0, "warning_count": 0}},
		"game:instances":  {OK: true, Data: map[string]any{"instances": []any{}}},
		"game:tree":       {OK: false, Error: "no game is running"},
		"game:ui_tree":    {OK: false, Error: "no game is running"},
		"game:screenshot": {OK: false, Error: "no game is running"},
	})
	t.Cleanup(srv.Close)

	// When
	data, ok := executeGameQADiagnosis(client.New(srv.URL), gameQADiagnoseOptions{lines: 40})

	// Then
	if ok || data["ok"] != false {
		t.Fatalf("diagnosis = %v, want failure", data)
	}
	issues, ok := data["issues"].([]string)
	if !ok {
		t.Fatalf("issues = %T, want []string", data["issues"])
	}
	if !containsGameQAIssue(issues, "expected exactly one live game process") || !containsGameQAIssue(issues, "runtime tree unavailable") {
		t.Fatalf("issues = %v, want process and runtime findings", issues)
	}
}

func newGameQADiagnoseServer(t *testing.T, responses map[string]protocol.Response) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request protocol.Request
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		key := request.Tool
		if request.Tool == "game" {
			key += ":" + request.Params["action"].(string)
		}
		response, ok := responses[key]
		if !ok {
			t.Fatalf("unexpected request %q", key)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
}

func containsGameQAIssue(issues []string, want string) bool {
	for _, issue := range issues {
		if strings.Contains(issue, want) {
			return true
		}
	}
	return false
}
