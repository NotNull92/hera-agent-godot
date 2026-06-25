package cmd

import (
	"fmt"
	"testing"
)

func TestParseOutputArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantType any
		wantLine any
		wantErr  bool
	}{
		{name: "empty"},
		{name: "type error", args: []string{"--type", "error"}, wantType: "error"},
		{name: "lines", args: []string{"--lines", "50"}, wantLine: 50},
		{name: "type and lines", args: []string{"--type", "warning", "--lines", "10"}, wantType: "warning", wantLine: 10},
		{name: "bad type", args: []string{"--type", "nope"}, wantErr: true},
		{name: "zero lines", args: []string{"--lines", "0"}, wantErr: true},
		{name: "non-int lines", args: []string{"--lines", "x"}, wantErr: true},
		{name: "dangling type", args: []string{"--type"}, wantErr: true},
		{name: "unknown flag", args: []string{"--bad"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := parseOutputArgs(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %v", p)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantType != nil && p["type"] != tt.wantType {
				t.Errorf("type = %v, want %v", p["type"], tt.wantType)
			}
			if tt.wantLine != nil && p["lines"] != tt.wantLine {
				t.Errorf("lines = %v, want %v", p["lines"], tt.wantLine)
			}
		})
	}
}

func TestParseDiagnosticsArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantLine any
		wantErr  bool
	}{
		{name: "empty"},
		{name: "lines", args: []string{"--lines", "25"}, wantLine: 25},
		{name: "zero lines", args: []string{"--lines", "0"}, wantErr: true},
		{name: "non-int lines", args: []string{"--lines", "x"}, wantErr: true},
		{name: "dangling lines", args: []string{"--lines"}, wantErr: true},
		{name: "unknown flag", args: []string{"--bad"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := parseDiagnosticsArgs(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %v", p)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantLine != nil && p["lines"] != tt.wantLine {
				t.Errorf("lines = %v, want %v", p["lines"], tt.wantLine)
			}
		})
	}
}

func TestParseGameArgs(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantAction string
		wantPath   any
		wantProp   any
		wantMethod any
		wantValue  any
		wantErr    bool
	}{
		{name: "tree", args: []string{"tree"}, wantAction: "tree"},
		{name: "node get", args: []string{"node", "get", "/root/Main"}, wantAction: "get", wantPath: "/root/Main"},
		{name: "node set", args: []string{"node", "set", "/root/Main", "--prop", "visible", "--value", "false"}, wantAction: "set", wantPath: "/root/Main", wantProp: "visible", wantValue: "false"},
		{name: "node call", args: []string{"node", "call", "/root/Main", "get_class"}, wantAction: "call", wantPath: "/root/Main", wantMethod: "get_class"},
		{name: "empty", wantErr: true},
		{name: "node missing get", args: []string{"node"}, wantErr: true},
		{name: "node get missing path", args: []string{"node", "get"}, wantErr: true},
		{name: "node get extra", args: []string{"node", "get", "a", "b"}, wantErr: true},
		{name: "node set missing prop", args: []string{"node", "set", "/root/Main", "--value", "1"}, wantErr: true},
		{name: "node set missing value", args: []string{"node", "set", "/root/Main", "--prop", "visible"}, wantErr: true},
		{name: "node call missing method", args: []string{"node", "call", "/root/Main"}, wantErr: true},
		{name: "unknown", args: []string{"nope"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := parseGameArgs(tt.args)
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
			if tt.wantProp != nil && p["prop"] != tt.wantProp {
				t.Errorf("prop = %v, want %v", p["prop"], tt.wantProp)
			}
			if tt.wantMethod != nil && p["method"] != tt.wantMethod {
				t.Errorf("method = %v, want %v", p["method"], tt.wantMethod)
			}
			if tt.wantValue != nil && p["value"] != tt.wantValue {
				t.Errorf("value = %v, want %v", p["value"], tt.wantValue)
			}
		})
	}
}

func TestGameActionMutates(t *testing.T) {
	tests := []struct {
		action any
		want   bool
	}{
		{action: "tree"},
		{action: "get"},
		{action: "set", want: true},
		{action: "call", want: true},
		{action: nil},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprint(tt.action), func(t *testing.T) {
			got := gameActionMutates(tt.action)
			if got != tt.want {
				t.Fatalf("gameActionMutates(%v) = %v, want %v", tt.action, got, tt.want)
			}
		})
	}
}

func TestSmokeRequiresMutation(t *testing.T) {
	tests := []struct {
		name string
		opts smokeOptions
		want bool
	}{
		{name: "default"},
		{name: "skip game", opts: smokeOptions{skipGame: true}},
		{name: "run game", opts: smokeOptions{runGame: true}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := smokeRequiresMutation(tt.opts)
			if got != tt.want {
				t.Fatalf("smokeRequiresMutation(%+v) = %v, want %v", tt.opts, got, tt.want)
			}
		})
	}
}

func TestParseSceneArgs(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantAction string
		wantErr    bool
	}{
		{name: "default", wantAction: "tree"},
		{name: "tree", args: []string{"tree"}, wantAction: "tree"},
		{name: "list", args: []string{"list"}, wantAction: "open_scenes"},
		{name: "unknown", args: []string{"nope"}, wantErr: true},
		{name: "tree extra arg", args: []string{"tree", "extra"}, wantErr: true},
		{name: "list unknown flag", args: []string{"list", "--bad"}, wantErr: true},
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
		})
	}
}

func TestParseNodeArgs(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantAction string
		wantQuery  any
		wantType   any
		wantPath   any
		wantErr    bool
	}{
		{name: "find all", args: []string{"find"}, wantAction: "find"},
		{name: "find query", args: []string{"find", "Player"}, wantAction: "find", wantQuery: "Player"},
		{name: "find type", args: []string{"find", "--type", "Camera2D"}, wantAction: "find", wantType: "Camera2D"},
		{name: "find query and type", args: []string{"find", "Cam", "--type", "Camera2D"}, wantAction: "find", wantQuery: "Cam", wantType: "Camera2D"},
		{name: "get path", args: []string{"get", "Player/Sprite"}, wantAction: "get", wantPath: "Player/Sprite"},
		{name: "no subcommand", wantErr: true},
		{name: "get without path", args: []string{"get"}, wantErr: true},
		{name: "unknown subcommand", args: []string{"nope"}, wantErr: true},
		{name: "find two queries", args: []string{"find", "a", "b"}, wantErr: true},
		{name: "find dangling type", args: []string{"find", "--type"}, wantErr: true},
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
			if tt.wantQuery != nil && p["query"] != tt.wantQuery {
				t.Errorf("query = %v, want %v", p["query"], tt.wantQuery)
			}
			if tt.wantType != nil && p["type"] != tt.wantType {
				t.Errorf("type = %v, want %v", p["type"], tt.wantType)
			}
			if tt.wantPath != nil && p["path"] != tt.wantPath {
				t.Errorf("path = %v, want %v", p["path"], tt.wantPath)
			}
		})
	}
}
