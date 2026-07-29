package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/NotNull92/hera-agent-godot/internal/client"
	"github.com/NotNull92/hera-agent-godot/internal/protocol"
)

func TestParseGameUIAuditArgs_whenAllFlagsAreValid(t *testing.T) {
	// Given
	args := []string{
		"ui", "audit",
		"--path", "/root/Main/HUD",
		"--severity", "warning",
		"--rule", "fullscreen_mouse_blocker",
		"--strict",
		"--limit", "75",
	}

	// When
	got, err := parseGameArgs(args)

	// Then
	if err != nil {
		t.Fatalf("parseGameArgs() error = %v, want nil", err)
	}
	if got["action"] != "ui_audit" {
		t.Fatalf("action = %v, want ui_audit", got["action"])
	}
	if got["path"] != "/root/Main/HUD" {
		t.Fatalf("path = %v, want /root/Main/HUD", got["path"])
	}
	if got["severity"] != "warning" || got["rule"] != "fullscreen_mouse_blocker" {
		t.Fatalf("filters = %v/%v, want warning/fullscreen_mouse_blocker", got["severity"], got["rule"])
	}
	if got["strict"] != true || got["limit"] != 75 {
		t.Fatalf("strict/limit = %v/%v, want true/75", got["strict"], got["limit"])
	}
}

func TestParseGameUIAuditArgs_whenFlagIsInvalid(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "unknown severity", args: []string{"ui", "audit", "--severity", "fatal"}},
		{name: "unknown rule", args: []string{"ui", "audit", "--rule", "crooked_panel"}},
		{name: "zero limit", args: []string{"ui", "audit", "--limit", "0"}},
		{name: "limit over maximum", args: []string{"ui", "audit", "--limit", "501"}},
		{name: "empty path", args: []string{"ui", "audit", "--path", ""}},
		{name: "unknown flag", args: []string{"ui", "audit", "--depth", "2"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			_, err := parseGameArgs(tt.args)

			// Then
			if err == nil {
				t.Fatal("parseGameArgs() accepted invalid UI audit flags")
			}
		})
	}
}

func TestParseGameUIAuditArgs_whenNoFlagsUsesDefaults(t *testing.T) {
	got, err := parseGameArgs([]string{"ui", "audit"})
	if err != nil {
		t.Fatalf("parseGameArgs() error = %v, want nil", err)
	}
	if got["action"] != "ui_audit" || got["severity"] != "all" {
		t.Fatalf("action/severity = %v/%v, want ui_audit/all", got["action"], got["severity"])
	}
	if got["strict"] != false || got["limit"] != defaultGameUIAuditLimit {
		t.Fatalf("strict/limit = %v/%v, want false/%d", got["strict"], got["limit"], defaultGameUIAuditLimit)
	}
}

func TestGameUIAuditPassed_whenStrictModeHasWarning(t *testing.T) {
	// Given
	data := map[string]any{"ok": false, "errors": 0, "warnings": 1, "strict": true}

	// When
	passed, err := gameUIAuditPassed(data)

	// Then
	if err != nil {
		t.Fatalf("gameUIAuditPassed() error = %v, want nil", err)
	}
	if passed {
		t.Fatal("gameUIAuditPassed() = true, want false")
	}
}

func TestGameUIAuditPassed_whenRuntimeBridgeMapsVerdict(t *testing.T) {
	for _, tt := range []struct {
		name string
		data map[string]any
		want bool
	}{
		{name: "clean", data: map[string]any{"ok": true, "errors": 0, "warnings": 0}, want: true},
		{name: "finding", data: map[string]any{"ok": false, "errors": 1, "warnings": 0}, want: false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := gameUIAuditPassed(tt.data)
			if err != nil {
				t.Fatalf("gameUIAuditPassed() error = %v, want nil", err)
			}
			if got != tt.want {
				t.Fatalf("gameUIAuditPassed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGameUIAuditParamsFromQAStep_whenScoped(t *testing.T) {
	// Given
	step := gameQAStep{
		Tool:   "game.ui.audit",
		Path:   "/root/Main/HUD",
		Params: map[string]any{"strict": true, "severity": "warning", "limit": json.Number("12")},
	}

	// When
	got := gameUIAuditParamsFromQAStep(step)

	// Then
	if got["action"] != "ui_audit" || got["path"] != "/root/Main/HUD" {
		t.Fatalf("action/path = %v/%v, want ui_audit//root/Main/HUD", got["action"], got["path"])
	}
	if got["strict"] != true || got["severity"] != "warning" || got["limit"] != "12" {
		t.Fatalf("audit params = %v, want strict warning limit 12", got)
	}
}

func TestExecuteGameQAStep_whenUIAuditFailsPreservesData(t *testing.T) {
	// Given
	auditData := map[string]any{
		"ok":       false,
		"errors":   1,
		"warnings": 2,
		"findings": []any{map[string]any{
			"rule":     "empty_interactive_rect",
			"severity": "error",
			"path":     "/root/Main/Start",
		}},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if err := json.NewEncoder(w).Encode(protocol.Response{OK: true, Data: auditData}); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	defer server.Close()

	// When
	result := executeGameQAStep(client.New(server.URL), 3, gameQAStep{
		Tool:   "game.ui.audit",
		Covers: []string{"ui-layout-clean"},
	})

	// Then
	if result.OK {
		t.Fatal("executeGameQAStep() result.OK = true, want false")
	}
	if result.Data == nil {
		t.Fatal("executeGameQAStep() result.Data = nil, want audit evidence")
	}
	if result.Error != "UI audit failed: 1 error, 2 warnings" {
		t.Fatalf("result.Error = %q", result.Error)
	}
}
