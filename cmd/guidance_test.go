package cmd

import (
	"strings"
	"testing"
)

func TestParseGuidanceArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		action  string
		wantErr bool
	}{
		{name: "ui", args: []string{"ui"}, action: "ui"},
		{name: "game-feel", args: []string{"game-feel"}, action: "game_feel"},
		{name: "game_feel", args: []string{"game_feel"}, action: "game_feel"},
		{name: "missing", wantErr: true},
		{name: "unknown", args: []string{"unknown"}, wantErr: true},
		{name: "extra", args: []string{"ui", "--json"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params, err := parseGuidanceArgs(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got params %v", params)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if params["action"] != tt.action {
				t.Fatalf("action = %v, want %s", params["action"], tt.action)
			}
		})
	}
}

func TestGuidanceDataFromStatus_whenGameFeelEnabled(t *testing.T) {
	data := guidanceDataFromStatus(map[string]any{"game_feel_ui_mode": true})

	if data["mode"] != "game_feel" {
		t.Fatalf("mode = %v, want game_feel", data["mode"])
	}
	if data["game_feel_ui_mode"] != true {
		t.Fatalf("game_feel_ui_mode = %v, want true", data["game_feel_ui_mode"])
	}
	checklist, ok := data["checklist"].([]string)
	if !ok {
		t.Fatalf("checklist type = %T, want []string", data["checklist"])
	}
	if len(checklist) == 0 {
		t.Fatal("checklist is empty")
	}
}

func TestGuidanceDataFromStatus_whenGameFeelEnabledIncludesLayoutQA(t *testing.T) {
	// Given
	data := guidanceDataFromStatus(map[string]any{"game_feel_ui_mode": true})
	checklist, ok := data["checklist"].([]string)
	if !ok {
		t.Fatalf("checklist type = %T, want []string", data["checklist"])
	}

	// Then
	for _, want := range []string{
		"padding",
		"contrast",
		"sibling panel",
		"viewport",
	} {
		if !containsSubstring(checklist, want) {
			t.Fatalf("checklist = %v, want substring %q", checklist, want)
		}
	}
}

func TestGuidanceDataFromStatus_whenGameFeelDisabled(t *testing.T) {
	data := guidanceDataFromStatus(map[string]any{"game_feel_ui_mode": false})

	if data["mode"] != "standard" {
		t.Fatalf("mode = %v, want standard", data["mode"])
	}
	if data["game_feel_ui_mode"] != false {
		t.Fatalf("game_feel_ui_mode = %v, want false", data["game_feel_ui_mode"])
	}
}

func TestGameFeelGuidanceDataFromStatus_whenGameFeelEnabled(t *testing.T) {
	data := gameFeelGuidanceDataFromStatus(map[string]any{"game_feel_mode": true})

	if data["mode"] != "game_feel" {
		t.Fatalf("mode = %v, want game_feel", data["mode"])
	}
	if data["game_feel_mode"] != true {
		t.Fatalf("game_feel_mode = %v, want true", data["game_feel_mode"])
	}
	topics, ok := data["topics"].([]string)
	if !ok {
		t.Fatalf("topics type = %T, want []string", data["topics"])
	}
	if len(topics) == 0 {
		t.Fatal("topics is empty")
	}
}

func TestGameFeelGuidanceDataFromStatus_whenEnabledIncludesTowerDefensePatterns(t *testing.T) {
	data := gameFeelGuidanceDataFromStatus(map[string]any{"game_feel_mode": true})

	patterns, ok := data["game_qa_patterns"].([]string)
	if !ok {
		t.Fatalf("game_qa_patterns type = %T, want []string", data["game_qa_patterns"])
	}
	if !containsSubstring(patterns, "guidance game-feel") {
		t.Fatalf("game_qa_patterns = %v, want guidance separation pattern", patterns)
	}
	if !containsSubstring(patterns, "MOUSE_FILTER_IGNORE") {
		t.Fatalf("game_qa_patterns = %v, want HUD mouse filter pattern", patterns)
	}
}

func TestGameFeelGuidanceDataFromStatus_whenEnabledIncludesPromptGamePatterns(t *testing.T) {
	data := gameFeelGuidanceDataFromStatus(map[string]any{"game_feel_mode": true})
	patterns, ok := data["game_qa_patterns"].([]string)
	if !ok {
		t.Fatalf("game_qa_patterns type = %T, want []string", data["game_qa_patterns"])
	}

	tests := []struct {
		name string
		want string
	}{
		{name: "deterministic events", want: "targeted event helpers"},
		{name: "delayed states", want: "disabled control counts"},
		{name: "hidden preconditions", want: "avoid preconditions created by earlier QA steps"},
		{name: "automated turns", want: "undo boundary"},
		{name: "autonomous loops", want: "restart-paused helper"},
		{name: "ai priority", want: "priority setup helpers"},
		{name: "collision feedback", want: "forced-overlap"},
		{name: "wave economy", want: "economy checks"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !containsSubstring(patterns, tt.want) {
				t.Fatalf("game_qa_patterns = %v, want substring %q", patterns, tt.want)
			}
		})
	}
}

func TestGameFeelGuidanceDataFromStatus_whenEnabledIncludesV07GenericRules(t *testing.T) {
	// Given
	data := gameFeelGuidanceDataFromStatus(map[string]any{"game_feel_mode": true})
	patterns, ok := data["game_qa_patterns"].([]string)
	if !ok {
		t.Fatalf("game_qa_patterns type = %T, want []string", data["game_qa_patterns"])
	}

	// Then
	for _, want := range []string{
		"state-changing runtime QA",
		"primary input scheme",
		"stateful controls",
		"terminal-state instruction",
		"live viewport",
	} {
		if !containsSubstring(patterns, want) {
			t.Fatalf("game_qa_patterns = %v, want substring %q", patterns, want)
		}
	}
}

func TestGameFeelGuidanceDataFromStatus_whenGameFeelDisabled(t *testing.T) {
	data := gameFeelGuidanceDataFromStatus(map[string]any{"game_feel_mode": false})

	if data["mode"] != "standard" {
		t.Fatalf("mode = %v, want standard", data["mode"])
	}
	if data["game_feel_mode"] != false {
		t.Fatalf("game_feel_mode = %v, want false", data["game_feel_mode"])
	}
}

func containsSubstring(values []string, needle string) bool {
	lowerNeedle := strings.ToLower(needle)
	for _, value := range values {
		if strings.Contains(strings.ToLower(value), lowerNeedle) {
			return true
		}
	}
	return false
}
