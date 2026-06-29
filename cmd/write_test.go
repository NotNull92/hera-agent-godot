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
		{name: "instance minimal", args: []string{"instance", "res://scenes/Enemy.tscn"},
			wantAction: "instance", want: map[string]any{"scene": "res://scenes/Enemy.tscn"}},
		{name: "instance with parent and name", args: []string{"instance", "res://scenes/Enemy.tscn", "--parent", "Root", "--name", "Enemy"},
			wantAction: "instance", want: map[string]any{"scene": "res://scenes/Enemy.tscn", "parent": "Root", "name": "Enemy"}},
		{name: "instance no scene", args: []string{"instance"}, wantErr: true},
		{name: "instance dangling parent", args: []string{"instance", "res://Enemy.tscn", "--parent"}, wantErr: true},
		{name: "instance dangling name", args: []string{"instance", "res://Enemy.tscn", "--name"}, wantErr: true},
		{name: "instance unknown flag", args: []string{"instance", "res://Enemy.tscn", "--bad"}, wantErr: true},

		{name: "set", args: []string{"set", "Hero", "--prop", "position", "--value", "Vector2(1, 2)"},
			wantAction: "set", want: map[string]any{"path": "Hero", "prop": "position", "value": "Vector2(1, 2)"}},
		{name: "set missing prop", args: []string{"set", "Hero", "--value", "1"}, wantErr: true},
		{name: "set missing value", args: []string{"set", "Hero", "--prop", "position"}, wantErr: true},
		{name: "set no path", args: []string{"set"}, wantErr: true},
		{name: "set resource", args: []string{"set-resource", "Hero", "--prop", "texture", "--resource", "res://hero.png"},
			wantAction: "set_resource", want: map[string]any{"path": "Hero", "prop": "texture", "resource": "res://hero.png"}},
		{name: "set resource missing prop", args: []string{"set-resource", "Hero", "--resource", "res://hero.png"}, wantErr: true},
		{name: "set resource missing resource", args: []string{"set-resource", "Hero", "--prop", "texture"}, wantErr: true},
		{name: "set resource no path", args: []string{"set-resource"}, wantErr: true},

		{name: "remove", args: []string{"remove", "Hero"}, wantAction: "remove", want: map[string]any{"path": "Hero"}},
		{name: "remove no path", args: []string{"remove"}, wantErr: true},
		{name: "remove extra", args: []string{"remove", "a", "b"}, wantErr: true},

		{name: "attach script", args: []string{"attach-script", "Hero", "res://scripts/hero.gd"},
			wantAction: "attach_script", want: map[string]any{"path": "Hero", "script": "res://scripts/hero.gd"}},
		{name: "attach script no path", args: []string{"attach-script"}, wantErr: true},
		{name: "attach script missing script", args: []string{"attach-script", "Hero"}, wantErr: true},
		{name: "detach script", args: []string{"detach-script", "Hero"}, wantAction: "detach_script", want: map[string]any{"path": "Hero"}},
		{name: "detach script no path", args: []string{"detach-script"}, wantErr: true},
		{name: "detach script extra", args: []string{"detach-script", "Hero", "extra"}, wantErr: true},
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

func TestNodeActionMutates_Instance(t *testing.T) {
	if !nodeActionMutates("instance") {
		t.Fatalf("node instance should require the mutation path")
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
		{name: "create", args: []string{"create", "res://scenes/Enemy.tscn"}, wantAction: "create", wantPath: "res://scenes/Enemy.tscn"},
		{name: "create root force open", args: []string{"create", "res://scenes/Enemy.tscn", "--root", "Node3D", "--force", "--open"}, wantAction: "create", wantPath: "res://scenes/Enemy.tscn"},
		{name: "create no path", args: []string{"create"}, wantErr: true},
		{name: "create dangling root", args: []string{"create", "res://A.tscn", "--root"}, wantErr: true},
		{name: "create unknown flag", args: []string{"create", "res://A.tscn", "--bad"}, wantErr: true},
		{name: "save as", args: []string{"save-as", "res://scenes/Copy.tscn"}, wantAction: "save_as", wantPath: "res://scenes/Copy.tscn"},
		{name: "save as force", args: []string{"save-as", "res://scenes/Copy.tscn", "--force"}, wantAction: "save_as", wantPath: "res://scenes/Copy.tscn"},
		{name: "save as no path", args: []string{"save-as"}, wantErr: true},
		{name: "save as unknown flag", args: []string{"save-as", "res://A.tscn", "--bad"}, wantErr: true},
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
			if tt.name == "create root force open" {
				if p["root"] != "Node3D" || p["force"] != true || p["open"] != true {
					t.Fatalf("params = %v, want root Node3D with force/open", p)
				}
			}
			if tt.name == "save as force" && p["force"] != true {
				t.Fatalf("force = %v, want true", p["force"])
			}
		})
	}
}

func TestSelectEditor_rejectsMultipleInstancesForMutation(t *testing.T) {
	instances := []discovery.Instance{
		{PID: 1, Port: 8770},
		{PID: 2, Port: 8771},
	}

	_, err := selectEditor(instances, true, 0)

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

	got, err := selectEditor(instances, false, 0)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.PID != 1 {
		t.Fatalf("pid = %d, want 1", got.PID)
	}
}

func TestSelectEditor_targetPIDOverridesMutationGuard(t *testing.T) {
	instances := []discovery.Instance{
		{PID: 1, Port: 8770},
		{PID: 2, Port: 8771},
	}

	// --instance picks the second editor even for a mutation (requireSingle).
	got, err := selectEditor(instances, true, 2)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.PID != 2 || got.Port != 8771 {
		t.Fatalf("got pid %d port %d, want pid 2 port 8771", got.PID, got.Port)
	}
}

func TestSelectEditor_targetPIDNotFound(t *testing.T) {
	instances := []discovery.Instance{
		{PID: 1, Port: 8770},
		{PID: 2, Port: 8771},
	}

	_, err := selectEditor(instances, false, 99)

	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "pid 99") {
		t.Fatalf("error = %q, want it to mention the missing pid", err)
	}
}
