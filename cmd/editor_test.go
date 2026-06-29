package cmd

import "testing"

func TestParseEditorArgs(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantAction string
		wantPath   any
		wantAdd    bool
		wantErr    bool
	}{
		{name: "state", args: []string{"state"}, wantAction: "state"},
		{name: "selected", args: []string{"selected"}, wantAction: "selected"},
		{name: "select", args: []string{"select", "Player"}, wantAction: "select", wantPath: "Player"},
		{name: "select add", args: []string{"select", "Player", "--add"}, wantAction: "select", wantPath: "Player", wantAdd: true},
		{name: "clear selection", args: []string{"clear-selection"}, wantAction: "clear_selection"},
		{name: "missing", wantErr: true},
		{name: "extra", args: []string{"state", "extra"}, wantErr: true},
		{name: "select missing path", args: []string{"select"}, wantErr: true},
		{name: "select unknown flag", args: []string{"select", "Player", "--bad"}, wantErr: true},
		{name: "unknown", args: []string{"focus"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := parseEditorArgs(tt.args)
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
				t.Fatalf("action = %v, want %v", p["action"], tt.wantAction)
			}
			if tt.wantPath != nil && p["path"] != tt.wantPath {
				t.Fatalf("path = %v, want %v", p["path"], tt.wantPath)
			}
			if p["add"] != tt.wantAdd && tt.wantAdd {
				t.Fatalf("add = %v, want %v", p["add"], tt.wantAdd)
			}
		})
	}
}

func TestEditorActionMutates(t *testing.T) {
	tests := []struct {
		action any
		want   bool
	}{
		{action: "state"},
		{action: "selected"},
		{action: "select", want: true},
		{action: "clear_selection", want: true},
		{action: nil},
	}
	for _, tt := range tests {
		if got := editorActionMutates(tt.action); got != tt.want {
			t.Fatalf("editorActionMutates(%v) = %v, want %v", tt.action, got, tt.want)
		}
	}
}
