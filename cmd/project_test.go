package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestParseScriptArgs(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantAction string
		wantPath   any
		wantErr    bool
	}{
		{name: "current", args: []string{"current"}, wantAction: "current"},
		{name: "inspect", args: []string{"inspect", "res://scripts/player.gd"}, wantAction: "inspect", wantPath: "res://scripts/player.gd"},
		{name: "create", args: []string{"create", "res://scripts/player.gd"}, wantAction: "create", wantPath: "res://scripts/player.gd"},
		{name: "create options", args: []string{"create", "res://scripts/player.gd", "--extends", "Node2D", "--class-name", "Player", "--force"}, wantAction: "create", wantPath: "res://scripts/player.gd"},
		{name: "create template flags", args: []string{"create", "res://scripts/player.gd", "--tool", "--ready", "--process", "--physics-process", "--input", "--unhandled-input", "--signal", "health_changed", "--signal", "died", "--export", "speed:float=240.0", "--export", "target:NodePath"}, wantAction: "create", wantPath: "res://scripts/player.gd"},
		{name: "missing subcommand", wantErr: true},
		{name: "current extra", args: []string{"current", "extra"}, wantErr: true},
		{name: "inspect missing path", args: []string{"inspect"}, wantErr: true},
		{name: "inspect extra", args: []string{"inspect", "res://a.gd", "extra"}, wantErr: true},
		{name: "missing path", args: []string{"create"}, wantErr: true},
		{name: "dangling extends", args: []string{"create", "res://a.gd", "--extends"}, wantErr: true},
		{name: "dangling class", args: []string{"create", "res://a.gd", "--class-name"}, wantErr: true},
		{name: "dangling signal", args: []string{"create", "res://a.gd", "--signal"}, wantErr: true},
		{name: "dangling export", args: []string{"create", "res://a.gd", "--export"}, wantErr: true},
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
			if tt.name == "create template flags" {
				signals, ok := p["signals"].([]string)
				if !ok || len(signals) != 2 || signals[0] != "health_changed" || signals[1] != "died" {
					t.Fatalf("signals = %#v, want health_changed/died", p["signals"])
				}
				exports, ok := p["exports"].([]string)
				if !ok || len(exports) != 2 || exports[0] != "speed:float=240.0" || exports[1] != "target:NodePath" {
					t.Fatalf("exports = %#v, want speed/target", p["exports"])
				}
				for _, key := range []string{"tool", "ready", "process", "physics_process", "input", "unhandled_input"} {
					if p[key] != true {
						t.Fatalf("%s = %v, want true in %v", key, p[key], p)
					}
				}
			}
		})
	}
}

func TestScriptActionMutates(t *testing.T) {
	tests := []struct {
		action any
		want   bool
	}{
		{action: "current"},
		{action: "inspect"},
		{action: "create", want: true},
		{action: nil},
	}
	for _, tt := range tests {
		if got := scriptActionMutates(tt.action); got != tt.want {
			t.Fatalf("scriptActionMutates(%v) = %v, want %v", tt.action, got, tt.want)
		}
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
		{name: "set main scene", args: []string{"set-main-scene", "res://main.tscn"}, wantAction: "set_main_scene", wantPath: "res://main.tscn"},
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
		{name: "set main scene missing path", args: []string{"set-main-scene"}, wantErr: true},
		{name: "set main scene extra", args: []string{"set-main-scene", "res://main.tscn", "extra"}, wantErr: true},
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

func TestProjectActionMutates(t *testing.T) {
	tests := []struct {
		action any
		want   bool
	}{
		{action: "info"},
		{action: "list_files"},
		{action: "mkdir", want: true},
		{action: "set_main_scene", want: true},
		{action: nil},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprint(tt.action), func(t *testing.T) {
			got := projectActionMutates(tt.action)
			if got != tt.want {
				t.Fatalf("projectActionMutates(%v) = %v, want %v", tt.action, got, tt.want)
			}
		})
	}
}

func TestSetMainSceneInProjectFile(t *testing.T) {
	dir := t.TempDir()
	projectPath := filepath.Join(dir, "project.godot")
	scenePath := filepath.Join(dir, "main.tscn")
	if err := os.WriteFile(projectPath, []byte("[application]\nconfig/name=\"Demo\"\nrun/main_scene=\"res://old.tscn\"\n"), 0o600); err != nil {
		t.Fatalf("write project: %v", err)
	}
	if err := os.WriteFile(scenePath, []byte("[gd_scene format=3]\n"), 0o600); err != nil {
		t.Fatalf("write scene: %v", err)
	}

	if err := setMainSceneInProjectFile(dir, "res://main.tscn"); err != nil {
		t.Fatalf("setMainSceneInProjectFile: %v", err)
	}

	got, err := os.ReadFile(projectPath)
	if err != nil {
		t.Fatalf("read project: %v", err)
	}
	want := "[application]\nconfig/name=\"Demo\"\nrun/main_scene=\"res://main.tscn\"\n"
	if string(got) != want {
		t.Fatalf("project.godot = %q, want %q", string(got), want)
	}
}

func TestReadMainSceneFromProjectFile(t *testing.T) {
	dir := t.TempDir()
	projectPath := filepath.Join(dir, "project.godot")
	if err := os.WriteFile(projectPath, []byte("[application]\nconfig/name=\"Demo\"\nrun/main_scene=\"res://main.tscn\"\n"), 0o600); err != nil {
		t.Fatalf("write project: %v", err)
	}

	got, err := readMainSceneFromProjectFile(dir)
	if err != nil {
		t.Fatalf("readMainSceneFromProjectFile: %v", err)
	}
	if got != "res://main.tscn" {
		t.Fatalf("main scene = %q, want res://main.tscn", got)
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
