package cmd

import (
	"strings"
	"testing"

	"github.com/NotNull92/hera-agent-godot/internal/discovery"
)

func TestParseNodeArgs_Write(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantAction string
		want       map[string]any
		wantErr    bool
	}{
		{name: "add minimal", args: []string{"add", "Node2D"}, wantAction: "add", want: map[string]any{"type": "Node2D"}},
		{name: "add with parent and name", args: []string{"add", "Sprite2D", "--parent", "Root", "--name", "Hero"},
			wantAction: "add", want: map[string]any{"type": "Sprite2D", "parent": "Root", "name": "Hero"}},
		{name: "add no type", args: []string{"add"}, wantErr: true},
		{name: "add dangling parent", args: []string{"add", "Node2D", "--parent"}, wantErr: true},
		{name: "add unknown flag", args: []string{"add", "Node2D", "--bad"}, wantErr: true},

		{name: "set", args: []string{"set", "Hero", "--prop", "position", "--value", "Vector2(1, 2)"},
			wantAction: "set", want: map[string]any{"path": "Hero", "prop": "position", "value": "Vector2(1, 2)"}},
		{name: "set missing prop", args: []string{"set", "Hero", "--value", "1"}, wantErr: true},
		{name: "set missing value", args: []string{"set", "Hero", "--prop", "position"}, wantErr: true},
		{name: "set no path", args: []string{"set"}, wantErr: true},

		{name: "remove", args: []string{"remove", "Hero"}, wantAction: "remove", want: map[string]any{"path": "Hero"}},
		{name: "remove no path", args: []string{"remove"}, wantErr: true},
		{name: "remove extra", args: []string{"remove", "a", "b"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := parseNodeArgs(tt.args)
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
			for k, v := range tt.want {
				if p[k] != v {
					t.Errorf("%s = %v, want %v", k, p[k], v)
				}
			}
		})
	}
}

func TestParseSceneArgs_Write(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantAction string
		wantPath   any
		wantErr    bool
	}{
		{name: "open", args: []string{"open", "res://Main.tscn"}, wantAction: "open", wantPath: "res://Main.tscn"},
		{name: "open no path", args: []string{"open"}, wantErr: true},
		{name: "open extra", args: []string{"open", "a", "b"}, wantErr: true},
		{name: "save", args: []string{"save"}, wantAction: "save"},
		{name: "save extra", args: []string{"save", "x"}, wantErr: true},
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
		})
	}
}

func TestSelectEditor_rejectsMultipleInstancesForMutation(t *testing.T) {
	instances := []discovery.Instance{
		{PID: 1, Port: 8770},
		{PID: 2, Port: 8771},
	}

	_, err := selectEditor(instances, true)

	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "multiple live Godot editors") {
		t.Fatalf("error = %q, want multiple-editor guard", err)
	}
}

func TestSelectEditor_allowsMultipleInstancesForReadOnly(t *testing.T) {
	instances := []discovery.Instance{
		{PID: 1, Port: 8770},
		{PID: 2, Port: 8771},
	}

	got, err := selectEditor(instances, false)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.PID != 1 {
		t.Fatalf("pid = %d, want 1", got.PID)
	}
}
