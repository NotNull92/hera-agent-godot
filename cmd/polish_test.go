package cmd

import "testing"

func TestParseScreenshotArgs(t *testing.T) {
	tests := []struct {
		name            string
		args            []string
		wantPath        any
		wantWidth       any
		wantHeight      any
		wantTransparent bool
		wantErr         bool
	}{
		{name: "empty"},
		{name: "path", args: []string{"--path", "user://x.png"}, wantPath: "user://x.png"},
		{name: "width and height", args: []string{"--width", "640", "--height", "480"}, wantWidth: 640, wantHeight: 480},
		{name: "transparent", args: []string{"--transparent"}, wantTransparent: true},
		{name: "bad width", args: []string{"--width", "x"}, wantErr: true},
		{name: "zero width", args: []string{"--width", "0"}, wantErr: true},
		{name: "too large width", args: []string{"--width", "4097"}, wantErr: true},
		{name: "too large height", args: []string{"--height", "4097"}, wantErr: true},
		{name: "dangling path", args: []string{"--path"}, wantErr: true},
		{name: "removed view flag", args: []string{"--view", "2d"}, wantErr: true},
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
			if tt.wantPath != nil && p["path"] != tt.wantPath {
				t.Errorf("path = %v, want %v", p["path"], tt.wantPath)
			}
			if tt.wantWidth != nil && p["width"] != tt.wantWidth {
				t.Errorf("width = %v, want %v", p["width"], tt.wantWidth)
			}
			if tt.wantHeight != nil && p["height"] != tt.wantHeight {
				t.Errorf("height = %v, want %v", p["height"], tt.wantHeight)
			}
			if tt.wantTransparent && p["transparent"] != true {
				t.Errorf("transparent = %v, want true", p["transparent"])
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
