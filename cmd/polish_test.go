package cmd

import "testing"

func TestParseScreenshotArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantView any
		wantPath any
		wantErr  bool
	}{
		{name: "empty"},
		{name: "view 2d", args: []string{"--view", "2d"}, wantView: "2d"},
		{name: "view 3d and path", args: []string{"--view", "3d", "--path", "user://x.png"}, wantView: "3d", wantPath: "user://x.png"},
		{name: "bad view", args: []string{"--view", "iso"}, wantErr: true},
		{name: "dangling view", args: []string{"--view"}, wantErr: true},
		{name: "unknown flag", args: []string{"--zoom"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := parseScreenshotArgs(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %v", p)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantView != nil && p["view"] != tt.wantView {
				t.Errorf("view = %v, want %v", p["view"], tt.wantView)
			}
			if tt.wantPath != nil && p["path"] != tt.wantPath {
				t.Errorf("path = %v, want %v", p["path"], tt.wantPath)
			}
		})
	}
}

func TestParseBatchFlags(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		wantFile     string
		wantContinue bool
		wantErr      bool
	}{
		{name: "empty"},
		{name: "file", args: []string{"--file", "cmds.json"}, wantFile: "cmds.json"},
		{name: "continue", args: []string{"--continue"}, wantContinue: true},
		{name: "file and continue", args: []string{"--file", "c.json", "--continue"}, wantFile: "c.json", wantContinue: true},
		{name: "dangling file", args: []string{"--file"}, wantErr: true},
		{name: "unknown flag", args: []string{"--nope"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, keepGoing, err := parseBatchFlags(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if file != tt.wantFile {
				t.Errorf("file = %q, want %q", file, tt.wantFile)
			}
			if keepGoing != tt.wantContinue {
				t.Errorf("continue = %v, want %v", keepGoing, tt.wantContinue)
			}
		})
	}
}

func TestExecute_resetsOutputMode_whenNoGlobalFlag(t *testing.T) {
	outputMode = "ids"

	code := Execute([]string{"help"})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if outputMode != "" {
		t.Fatalf("outputMode = %q, want empty", outputMode)
	}
}
