package cmd

import "testing"

func TestParseResourceArgs(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantAction string
		wantPath   any
		wantErr    bool
	}{
		{name: "get", args: []string{"get", "res://x.tres"}, wantAction: "get", wantPath: "res://x.tres"},
		{name: "uid", args: []string{"uid", "res://x.tres"}, wantAction: "uid", wantPath: "res://x.tres"},
		{name: "list default", args: []string{"list"}, wantAction: "list", wantPath: "res://"},
		{name: "list path", args: []string{"list", "res://ui"}, wantAction: "list", wantPath: "res://ui"},
		{name: "list with filters", args: []string{"list", "res://ui", "--type", "StyleBoxFlat", "--pattern", "panel", "--limit", "25"}, wantAction: "list", wantPath: "res://ui"},
		{name: "set", args: []string{"set", "res://ui/panel.tres", "--prop", "bg_color=Color(1, 0, 0, 1)"}, wantAction: "set", wantPath: "res://ui/panel.tres"},
		{name: "create", args: []string{"create", "StyleBoxFlat", "res://ui/panel.tres"}, wantAction: "create", wantPath: "res://ui/panel.tres"},
		{name: "create with options", args: []string{"create", "StyleBoxFlat", "res://ui/panel.tres", "--force", "--prop", "bg_color=Color(1, 0, 0, 1)", "--prop", "corner_radius_top_left=4"}, wantAction: "create", wantPath: "res://ui/panel.tres"},
		{name: "resave", args: []string{"resave", "res://x.tres"}, wantAction: "resave", wantPath: "res://x.tres"},
		{name: "update uids", args: []string{"update-uids"}, wantAction: "update_uids"},
		{name: "export mesh library", args: []string{"export-mesh-library", "res://tiles.tscn", "res://tiles.meshlib"}, wantAction: "export_mesh_library", wantPath: "res://tiles.tscn"},
		{name: "export mesh library with items", args: []string{"export-mesh-library", "res://tiles.tscn", "res://tiles.meshlib", "--item", "Wall", "--item", "Floor"}, wantAction: "export_mesh_library", wantPath: "res://tiles.tscn"},
		{name: "get no path", args: []string{"get"}, wantErr: true},
		{name: "get extra", args: []string{"get", "a", "b"}, wantErr: true},
		{name: "uid no path", args: []string{"uid"}, wantErr: true},
		{name: "list bad limit", args: []string{"list", "--limit", "bad"}, wantErr: true},
		{name: "list unknown flag", args: []string{"list", "--bad"}, wantErr: true},
		{name: "set no prop", args: []string{"set", "res://x.tres"}, wantErr: true},
		{name: "set bad prop", args: []string{"set", "res://x.tres", "--prop", "bad"}, wantErr: true},
		{name: "set unknown flag", args: []string{"set", "res://x.tres", "--bad"}, wantErr: true},
		{name: "create missing path", args: []string{"create", "StyleBoxFlat"}, wantErr: true},
		{name: "create dangling prop", args: []string{"create", "StyleBoxFlat", "res://x.tres", "--prop"}, wantErr: true},
		{name: "create bad prop", args: []string{"create", "StyleBoxFlat", "res://x.tres", "--prop", "bad"}, wantErr: true},
		{name: "create unknown flag", args: []string{"create", "StyleBoxFlat", "res://x.tres", "--bad"}, wantErr: true},
		{name: "resave no path", args: []string{"resave"}, wantErr: true},
		{name: "update uids extra", args: []string{"update-uids", "extra"}, wantErr: true},
		{name: "export mesh library missing output", args: []string{"export-mesh-library", "res://tiles.tscn"}, wantErr: true},
		{name: "export mesh library dangling item", args: []string{"export-mesh-library", "res://tiles.tscn", "res://tiles.meshlib", "--item"}, wantErr: true},
		{name: "export mesh library unknown flag", args: []string{"export-mesh-library", "res://tiles.tscn", "res://tiles.meshlib", "--bad"}, wantErr: true},
		{name: "no subcommand", args: nil, wantErr: true},
		{name: "unknown subcommand", args: []string{"bad", "res://x.tres"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := parseResourceArgs(tt.args)
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
			if tt.name == "export mesh library" && p["output"] != "res://tiles.meshlib" {
				t.Fatalf("output = %v, want res://tiles.meshlib", p["output"])
			}
			if tt.name == "export mesh library with items" {
				items, ok := p["items"].([]string)
				if !ok || len(items) != 2 || items[0] != "Wall" || items[1] != "Floor" {
					t.Fatalf("items = %#v, want Wall/Floor", p["items"])
				}
			}
			if tt.name == "create with options" {
				props, ok := p["props"].(map[string]string)
				if !ok || props["bg_color"] != "Color(1, 0, 0, 1)" || props["corner_radius_top_left"] != "4" || p["force"] != true {
					t.Fatalf("create params = %#v, want force and props", p)
				}
			}
			if tt.name == "set" {
				props, ok := p["props"].(map[string]string)
				if !ok || props["bg_color"] != "Color(1, 0, 0, 1)" {
					t.Fatalf("set params = %#v, want props", p)
				}
			}
			if tt.name == "list with filters" {
				if p["type"] != "StyleBoxFlat" || p["pattern"] != "panel" || p["limit"] != 25 {
					t.Fatalf("list params = %#v, want filters", p)
				}
			}
		})
	}
}

func TestResourceActionMutates(t *testing.T) {
	tests := []struct {
		action any
		want   bool
	}{
		{action: "get", want: false},
		{action: "uid", want: false},
		{action: "list", want: false},
		{action: "set", want: true},
		{action: "create", want: true},
		{action: "resave", want: true},
		{action: "update_uids", want: true},
		{action: "export_mesh_library", want: true},
	}
	for _, tt := range tests {
		if got := resourceActionMutates(tt.action); got != tt.want {
			t.Fatalf("resourceActionMutates(%v) = %v, want %v", tt.action, got, tt.want)
		}
	}
}
