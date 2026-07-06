package cmd

import (
	"fmt"
	"testing"
)

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
		{name: "ui tree", args: []string{"ui", "tree"}, wantAction: "ui_tree"},
		{name: "node get", args: []string{"node", "get", "/root/Main"}, wantAction: "get", wantPath: "/root/Main"},
		{name: "node get prop", args: []string{"node", "get", "/root/Main/Label", "--prop", "text"}, wantAction: "get", wantPath: "/root/Main/Label", wantProp: "text"},
		{name: "node get props", args: []string{"node", "get", "/root/Main", "--props", "visible,position"}, wantAction: "get", wantPath: "/root/Main"},
		{name: "node get normalizes Git Bash root path", args: []string{"node", "get", "C:/Program Files/Git/root/Main"}, wantAction: "get", wantPath: "/root/Main"},
		{name: "node set", args: []string{"node", "set", "/root/Main", "--prop", "visible", "--value", "false"}, wantAction: "set", wantPath: "/root/Main", wantProp: "visible", wantValue: "false"},
		{name: "assert eq", args: []string{"assert", "/root/Main/Label", "text", "eq", "Ready"}, wantAction: "assert", wantPath: "/root/Main/Label", wantProp: "text", wantValue: "Ready"},
		{name: "assert exists", args: []string{"assert", "/root/Main/Label", "text", "exists"}, wantAction: "assert", wantPath: "/root/Main/Label", wantProp: "text"},
		{name: "instances", args: []string{"instances"}, wantAction: "instances"},
		{name: "screenshot", args: []string{"screenshot", "--path", "tmp/game.png"}, wantAction: "screenshot", wantPath: "tmp/game.png"},
		{name: "screenshot analyze", args: []string{"screenshot", "--analyze"}, wantAction: "screenshot"},
		{name: "node call", args: []string{"node", "call", "/root/Main", "get_class"}, wantAction: "call", wantPath: "/root/Main", wantMethod: "get_class"},
		{name: "empty", wantErr: true},
		{name: "node missing get", args: []string{"node"}, wantErr: true},
		{name: "ui missing tree", args: []string{"ui"}, wantErr: true},
		{name: "ui unknown", args: []string{"ui", "nope"}, wantErr: true},
		{name: "node get missing path", args: []string{"node", "get"}, wantErr: true},
		{name: "node get extra", args: []string{"node", "get", "a", "b"}, wantErr: true},
		{name: "node get prop and props conflict", args: []string{"node", "get", "a", "--prop", "text", "--props", "visible"}, wantErr: true},
		{name: "node set missing prop", args: []string{"node", "set", "/root/Main", "--value", "1"}, wantErr: true},
		{name: "node set missing value", args: []string{"node", "set", "/root/Main", "--prop", "visible"}, wantErr: true},
		{name: "node call missing method", args: []string{"node", "call", "/root/Main"}, wantErr: true},
		{name: "assert missing value", args: []string{"assert", "/root/Main", "text", "eq"}, wantErr: true},
		{name: "assert unknown op", args: []string{"assert", "/root/Main", "text", "matches", "Ready"}, wantErr: true},
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
			if tt.name == "node get props" {
				props, ok := p["props"].([]string)
				if !ok {
					t.Fatalf("props = %T, want []string", p["props"])
				}
				if fmt.Sprint(props) != "[visible position]" {
					t.Fatalf("props = %v, want [visible position]", props)
				}
			}
			if tt.name == "screenshot analyze" && p["analyze"] != true {
				t.Fatalf("analyze = %v, want true", p["analyze"])
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
		{action: "ui_tree"},
		{action: "instances"},
		{action: "screenshot"},
		{action: "assert"},
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

func TestParseGameQADiscoverArgs_returnsRuntimeQAHelperDiscoveryParams(t *testing.T) {
	// Given
	args := []string{"/root/Main"}

	// When
	got, err := parseGameQADiscoverArgs(args)

	// Then
	if err != nil {
		t.Fatalf("parseGameQADiscoverArgs error: %v", err)
	}
	if got["action"] != "qa_discover" {
		t.Fatalf("action = %v, want qa_discover", got["action"])
	}
	if got["path"] != "/root/Main" {
		t.Fatalf("path = %v, want /root/Main", got["path"])
	}
}

func TestParseGameQADiscoverArgs_rejectsExtraArguments(t *testing.T) {
	// Given
	args := []string{"/root/Main", "extra"}

	// When
	_, err := parseGameQADiscoverArgs(args)

	// Then
	if err == nil {
		t.Fatal("expected qa discover parse error")
	}
}

func TestParseGameArgs_returnsScopedUITreeParams(t *testing.T) {
	// Given
	args := []string{
		"ui", "tree",
		"--path", "/root/Main/HUD",
		"--depth", "2",
		"--fields", "name,path,text,rect,disabled",
		"--type", "Button",
		"--text", "Restart",
	}

	// When
	got, err := parseGameArgs(args)

	// Then
	if err != nil {
		t.Fatalf("parseGameArgs error: %v", err)
	}
	if got["action"] != "ui_tree" {
		t.Fatalf("action = %v, want ui_tree", got["action"])
	}
	if got["path"] != "/root/Main/HUD" {
		t.Fatalf("path = %v, want /root/Main/HUD", got["path"])
	}
	if got["depth"] != 2 {
		t.Fatalf("depth = %v, want 2", got["depth"])
	}
	fields, ok := got["fields"].([]string)
	if !ok {
		t.Fatalf("fields = %T, want []string", got["fields"])
	}
	if fmt.Sprint(fields) != "[name path text rect disabled]" {
		t.Fatalf("fields = %v, want [name path text rect disabled]", fields)
	}
	if got["type"] != "Button" {
		t.Fatalf("type = %v, want Button", got["type"])
	}
	if got["text"] != "Restart" {
		t.Fatalf("text = %v, want Restart", got["text"])
	}
}

func TestParseGameArgs_rejectsScopedUITreeInvalidFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "dangling path", args: []string{"ui", "tree", "--path"}},
		{name: "dangling depth", args: []string{"ui", "tree", "--depth"}},
		{name: "negative depth", args: []string{"ui", "tree", "--depth", "-1"}},
		{name: "non integer depth", args: []string{"ui", "tree", "--depth", "deep"}},
		{name: "empty field", args: []string{"ui", "tree", "--fields", "name,,text"}},
		{name: "unknown field", args: []string{"ui", "tree", "--fields", "name,color"}},
		{name: "dangling type", args: []string{"ui", "tree", "--type"}},
		{name: "dangling text", args: []string{"ui", "tree", "--text"}},
		{name: "unknown flag", args: []string{"ui", "tree", "--bad"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			_, err := parseGameArgs(tt.args)

			// Then
			if err == nil {
				t.Fatal("expected scoped ui tree parse error")
			}
		})
	}
}
