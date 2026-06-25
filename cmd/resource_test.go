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
		{name: "resave", args: []string{"resave", "res://x.tres"}, wantAction: "resave", wantPath: "res://x.tres"},
		{name: "update uids", args: []string{"update-uids"}, wantAction: "update_uids"},
		{name: "export mesh library", args: []string{"export-mesh-library", "res://tiles.tscn", "res://tiles.meshlib"}, wantAction: "export_mesh_library", wantPath: "res://tiles.tscn"},
		{name: "export mesh library with items", args: []string{"export-mesh-library", "res://tiles.tscn", "res://tiles.meshlib", "--item", "Wall", "--item", "Floor"}, wantAction: "export_mesh_library", wantPath: "res://tiles.tscn"},
		{name: "get no path", args: []string{"get"}, wantErr: true},
		{name: "get extra", args: []string{"get", "a", "b"}, wantErr: true},
		{name: "uid no path", args: []string{"uid"}, wantErr: true},
		{name: "resave no path", args: []string{"resave"}, wantErr: true},
		{name: "update uids extra", args: []string{"update-uids", "extra"}, wantErr: true},
		{name: "export mesh library missing output", args: []string{"export-mesh-library", "res://tiles.tscn"}, wantErr: true},
		{name: "export mesh library dangling item", args: []string{"export-mesh-library", "res://tiles.tscn", "res://tiles.meshlib", "--item"}, wantErr: true},
		{name: "export mesh library unknown flag", args: []string{"export-mesh-library", "res://tiles.tscn", "res://tiles.meshlib", "--bad"}, wantErr: true},
		{name: "no subcommand", args: nil, wantErr: true},
		{name: "unknown subcommand", args: []string{"set", "res://x.tres"}, wantErr: true},
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
