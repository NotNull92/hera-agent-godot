package cmd

import "testing"

func TestParseScriptArgs(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantAction string
		wantPath   any
		wantErr    bool
	}{
		{name: "create", args: []string{"create", "res://scripts/player.gd"}, wantAction: "create", wantPath: "res://scripts/player.gd"},
		{name: "create options", args: []string{"create", "res://scripts/player.gd", "--extends", "Node2D", "--class-name", "Player", "--force"}, wantAction: "create", wantPath: "res://scripts/player.gd"},
		{name: "missing subcommand", wantErr: true},
		{name: "missing path", args: []string{"create"}, wantErr: true},
		{name: "dangling extends", args: []string{"create", "res://a.gd", "--extends"}, wantErr: true},
		{name: "dangling class", args: []string{"create", "res://a.gd", "--class-name"}, wantErr: true},
		{name: "unknown flag", args: []string{"create", "res://a.gd", "--bad"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := parseScriptArgs(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %v", p)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if p["action"] != tt.wantAction {
				t.Errorf("action = %v, want %v", p["action"], tt.wantAction)
			}
			if tt.wantPath != nil && p["path"] != tt.wantPath {
				t.Errorf("path = %v, want %v", p["path"], tt.wantPath)
			}
			if tt.name == "create options" {
				if p["extends"] != "Node2D" || p["class_name"] != "Player" || p["force"] != true {
					t.Fatalf("params = %v, want extends/class_name/force", p)
				}
			}
		})
	}
}

func TestParseProjectArgs(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantAction string
		wantPath   any
		wantErr    bool
	}{
		{name: "mkdir", args: []string{"mkdir", "res://scripts"}, wantAction: "mkdir", wantPath: "res://scripts"},
		{name: "info", args: []string{"info"}, wantAction: "info"},
		{name: "info extra", args: []string{"info", "extra"}, wantErr: true},
		{name: "list files", args: []string{"list-files"}, wantAction: "list_files"},
		{name: "list files options", args: []string{"list-files", "--type", "scene", "--pattern", "levels", "--limit", "25"}, wantAction: "list_files"},
		{name: "list files dangling type", args: []string{"list-files", "--type"}, wantErr: true},
		{name: "list files bad type", args: []string{"list-files", "--type", "bad"}, wantErr: true},
		{name: "list files dangling pattern", args: []string{"list-files", "--pattern"}, wantErr: true},
		{name: "list files bad limit", args: []string{"list-files", "--limit", "x"}, wantErr: true},
		{name: "list files zero limit", args: []string{"list-files", "--limit", "0"}, wantErr: true},
		{name: "missing subcommand", wantErr: true},
		{name: "mkdir missing path", args: []string{"mkdir"}, wantErr: true},
		{name: "mkdir extra", args: []string{"mkdir", "res://a", "extra"}, wantErr: true},
		{name: "unknown", args: []string{"nope"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := parseProjectArgs(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %v", p)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if p["action"] != tt.wantAction {
				t.Errorf("action = %v, want %v", p["action"], tt.wantAction)
			}
			if tt.wantPath != nil && p["path"] != tt.wantPath {
				t.Errorf("path = %v, want %v", p["path"], tt.wantPath)
			}
			if tt.name == "list files options" {
				if p["type"] != "scene" || p["pattern"] != "levels" || p["limit"] != 25 {
					t.Fatalf("params = %v, want type/pattern/limit", p)
				}
			}
		})
	}
}

func TestParseSmokeArgs(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		wantRunGame  bool
		wantSkipGame bool
		wantErr      bool
	}{
		{name: "default"},
		{name: "run game", args: []string{"--run-game"}, wantRunGame: true},
		{name: "skip game", args: []string{"--skip-game"}, wantSkipGame: true},
		{name: "conflict", args: []string{"--run-game", "--skip-game"}, wantErr: true},
		{name: "unknown", args: []string{"--bad"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, err := parseSmokeArgs(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %v", opts)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if opts.runGame != tt.wantRunGame {
				t.Errorf("runGame = %v, want %v", opts.runGame, tt.wantRunGame)
			}
			if opts.skipGame != tt.wantSkipGame {
				t.Errorf("skipGame = %v, want %v", opts.skipGame, tt.wantSkipGame)
			}
		})
	}
}
