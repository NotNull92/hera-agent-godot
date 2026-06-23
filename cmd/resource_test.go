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
		{name: "get no path", args: []string{"get"}, wantErr: true},
		{name: "get extra", args: []string{"get", "a", "b"}, wantErr: true},
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
		})
	}
}

func TestResourceIsReadOnly(t *testing.T) {
	// resource has no mutating actions, so it must not be classified as a
	// mutation by any shared guard. There is no resourceActionMutates; runResource
	// always uses the read path. This test documents that intent.
	p, err := parseResourceArgs([]string{"get", "res://x.tres"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p["action"] != "get" {
		t.Fatalf("action = %v, want get", p["action"])
	}
}
