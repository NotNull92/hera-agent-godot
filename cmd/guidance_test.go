package cmd

import "testing"

func TestParseGuidanceArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{name: "ui", args: []string{"ui"}},
		{name: "missing", wantErr: true},
		{name: "unknown", args: []string{"game-feel"}, wantErr: true},
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
			if params["action"] != "ui" {
				t.Fatalf("action = %v, want ui", params["action"])
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

func TestGuidanceDataFromStatus_whenGameFeelDisabled(t *testing.T) {
	data := guidanceDataFromStatus(map[string]any{"game_feel_ui_mode": false})

	if data["mode"] != "standard" {
		t.Fatalf("mode = %v, want standard", data["mode"])
	}
	if data["game_feel_ui_mode"] != false {
		t.Fatalf("game_feel_ui_mode = %v, want false", data["game_feel_ui_mode"])
	}
}
