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

func TestExecute_instanceFlag(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantCode int
		wantPID  int
	}{
		{name: "space form", args: []string{"--instance", "123", "help"}, wantCode: 0, wantPID: 123},
		{name: "equals form", args: []string{"--instance=456", "help"}, wantCode: 0, wantPID: 456},
		{name: "with output mode", args: []string{"--json", "--instance", "7", "help"}, wantCode: 0, wantPID: 7},
		{name: "before output mode", args: []string{"--instance", "8", "--ids", "help"}, wantCode: 0, wantPID: 8},
		{name: "missing value", args: []string{"--instance"}, wantCode: 2, wantPID: 0},
		{name: "missing equals value", args: []string{"--instance="}, wantCode: 2, wantPID: 0},
		{name: "non-numeric", args: []string{"--instance", "abc"}, wantCode: 2, wantPID: 0},
		{name: "zero pid", args: []string{"--instance", "0"}, wantCode: 2, wantPID: 0},
		{name: "negative pid", args: []string{"--instance", "-3"}, wantCode: 2, wantPID: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := Execute(tt.args)
			if code != tt.wantCode {
				t.Fatalf("exit code = %d, want %d", code, tt.wantCode)
			}
			if targetPID != tt.wantPID {
				t.Fatalf("targetPID = %d, want %d", targetPID, tt.wantPID)
			}
		})
	}
}

func TestExecute_instanceFlagAfterCommandIsNotGlobal(t *testing.T) {
	code := Execute([]string{"node", "find", "--instance", "123"})

	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if targetPID != 0 {
		t.Fatalf("targetPID = %d, want 0", targetPID)
	}
}

func TestExecute_resetsTargetPID_whenNoInstanceFlag(t *testing.T) {
	targetPID = 42

	code := Execute([]string{"help"})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if targetPID != 0 {
		t.Fatalf("targetPID = %d, want 0", targetPID)
	}
}
