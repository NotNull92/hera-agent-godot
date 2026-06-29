package cmd

import "testing"

func TestParseProjectFilesystemArgs(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantAction string
		wantPaths  []string
		wantErr    bool
	}{
		{name: "scan", args: []string{"scan"}, wantAction: "scan"},
		{name: "reimport one", args: []string{"reimport", "res://icon.svg"}, wantAction: "reimport", wantPaths: []string{"res://icon.svg"}},
		{name: "reimport many", args: []string{"reimport", "res://a.png", "res://b.wav"}, wantAction: "reimport", wantPaths: []string{"res://a.png", "res://b.wav"}},
		{name: "scan extra", args: []string{"scan", "extra"}, wantErr: true},
		{name: "reimport missing", args: []string{"reimport"}, wantErr: true},
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
				t.Fatalf("action = %v, want %v", p["action"], tt.wantAction)
			}
			if tt.wantPaths != nil {
				paths, ok := p["paths"].([]string)
				if !ok || len(paths) != len(tt.wantPaths) {
					t.Fatalf("paths = %#v, want %#v", p["paths"], tt.wantPaths)
				}
				for i, want := range tt.wantPaths {
					if paths[i] != want {
						t.Fatalf("paths[%d] = %q, want %q", i, paths[i], want)
					}
				}
			}
		})
	}
}

func TestProjectFilesystemActionsMutate(t *testing.T) {
	for _, action := range []string{"scan", "reimport"} {
		if !projectActionMutates(action) {
			t.Fatalf("projectActionMutates(%q) = false, want true", action)
		}
	}
}
