package cmd

import (
	"fmt"
	"testing"
)

func TestParseSceneArgsDetailed(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantAction string
		wantPath   any
		wantErr    bool
	}{
		{name: "default tree", wantAction: "tree"},
		{name: "tree", args: []string{"tree"}, wantAction: "tree"},
		{name: "list", args: []string{"list"}, wantAction: "open_scenes"},
		{name: "open", args: []string{"open", "res://scenes/Main.tscn"}, wantAction: "open", wantPath: "res://scenes/Main.tscn"},
		{name: "reload current", args: []string{"reload"}, wantAction: "reload"},
		{name: "reload path", args: []string{"reload", "res://scenes/Main.tscn"}, wantAction: "reload", wantPath: "res://scenes/Main.tscn"},
		{name: "save", args: []string{"save"}, wantAction: "save"},
		{name: "create", args: []string{"create", "res://scenes/New.tscn", "--root", "Node2D", "--force", "--open"}, wantAction: "create", wantPath: "res://scenes/New.tscn"},
		{name: "save as", args: []string{"save-as", "res://scenes/Copy.tscn", "--force"}, wantAction: "save_as", wantPath: "res://scenes/Copy.tscn"},
		{name: "tree extra", args: []string{"tree", "extra"}, wantErr: true},
		{name: "list extra", args: []string{"list", "extra"}, wantErr: true},
		{name: "open missing", args: []string{"open"}, wantErr: true},
		{name: "reload extra", args: []string{"reload", "res://a.tscn", "extra"}, wantErr: true},
		{name: "save extra", args: []string{"save", "extra"}, wantErr: true},
		{name: "unknown", args: []string{"nope"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := parseSceneArgs(tt.args)
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
			if tt.name == "create" {
				if p["root"] != "Node2D" || p["force"] != true || p["open"] != true {
					t.Fatalf("params = %v, want root/force/open", p)
				}
			}
			if tt.name == "save as" && p["force"] != true {
				t.Fatalf("force = %v, want true", p["force"])
			}
		})
	}
}

func TestSceneActionMutates(t *testing.T) {
	tests := []struct {
		action any
		want   bool
	}{
		{action: "tree"},
		{action: "open_scenes"},
		{action: "open", want: true},
		{action: "reload", want: true},
		{action: "save", want: true},
		{action: "create", want: true},
		{action: "save_as", want: true},
		{action: nil},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprint(tt.action), func(t *testing.T) {
			got := sceneActionMutates(tt.action)
			if got != tt.want {
				t.Fatalf("sceneActionMutates(%v) = %v, want %v", tt.action, got, tt.want)
			}
		})
	}
}
