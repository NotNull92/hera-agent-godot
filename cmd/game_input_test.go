package cmd

import "testing"

func TestParseGameArgs_returnsMouseInputParams(t *testing.T) {
	got, err := parseGameArgs([]string{"input", "mouse", "--x", "120", "--y", "240", "--button", "right", "--press", "--modifiers", "shift,ctrl"})
	if err != nil {
		t.Fatalf("parseGameArgs error: %v", err)
	}
	if got["action"] != "input" || got["kind"] != "mouse" {
		t.Fatalf("input action/kind = %v/%v, want input/mouse", got["action"], got["kind"])
	}
	if got["x"] != 120 || got["y"] != 240 || got["button"] != "right" || got["mode"] != "press" {
		t.Fatalf("mouse input params = %v", got)
	}
	modifiers, ok := got["modifiers"].([]string)
	if !ok || len(modifiers) != 2 || modifiers[0] != "shift" || modifiers[1] != "ctrl" {
		t.Fatalf("modifiers = %v, want [shift ctrl]", got["modifiers"])
	}
}

func TestParseGameArgs_returnsKeyInputParams(t *testing.T) {
	got, err := parseGameArgs([]string{"input", "key", "--key", "KEY_W", "--press", "--physical", "--route", "input"})
	if err != nil {
		t.Fatalf("parseGameArgs error: %v", err)
	}
	if got["action"] != "input" || got["kind"] != "key" {
		t.Fatalf("input action/kind = %v/%v, want input/key", got["action"], got["kind"])
	}
	if got["key"] != "KEY_W" || got["mode"] != "press" || got["physical"] != true || got["route"] != "input" {
		t.Fatalf("key input params = %v", got)
	}
}

func TestParseGameArgs_returnsActionInputParams(t *testing.T) {
	got, err := parseGameArgs([]string{"input", "action", "jump", "--press", "--strength", "0.75"})
	if err != nil {
		t.Fatalf("parseGameArgs error: %v", err)
	}
	if got["action"] != "input" || got["kind"] != "action" || got["name"] != "jump" || got["mode"] != "press" {
		t.Fatalf("action input params = %v", got)
	}
	if got["strength"] != 0.75 {
		t.Fatalf("strength = %v, want 0.75", got["strength"])
	}
}

func TestParseGameArgs_returnsTextInputParams(t *testing.T) {
	got, err := parseGameArgs([]string{"input", "text", "hello"})
	if err != nil {
		t.Fatalf("parseGameArgs error: %v", err)
	}
	if got["action"] != "input" || got["kind"] != "text" || got["text"] != "hello" {
		t.Fatalf("text input params = %v", got)
	}
}

func TestParseGameArgs_returnsInputLogParams(t *testing.T) {
	got, err := parseGameArgs([]string{"input-log", "--limit", "3", "--clear"})
	if err != nil {
		t.Fatalf("parseGameArgs error: %v", err)
	}
	if got["action"] != "input_log" || got["limit"] != 3 || got["clear"] != true {
		t.Fatalf("input-log params = %v", got)
	}
}

func TestParseGameArgs_rejectsInvalidInputArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "missing kind", args: []string{"input"}},
		{name: "mouse missing mode", args: []string{"input", "mouse", "--x", "1", "--y", "2"}},
		{name: "mouse missing y", args: []string{"input", "mouse", "--x", "1", "--click"}},
		{name: "mouse bad button", args: []string{"input", "mouse", "--x", "1", "--y", "2", "--button", "bad", "--click"}},
		{name: "key missing key", args: []string{"input", "key", "--press"}},
		{name: "key missing mode", args: []string{"input", "key", "--key", "KEY_W"}},
		{name: "action missing mode", args: []string{"input", "action", "jump"}},
		{name: "action bad strength", args: []string{"input", "action", "jump", "--press", "--strength", "2"}},
		{name: "bad modifier", args: []string{"input", "mouse", "--x", "1", "--y", "2", "--click", "--modifiers", "super"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseGameArgs(tt.args)
			if err == nil {
				t.Fatal("expected parse error")
			}
		})
	}
}

func TestGameActionMutates_returnsTrue_forInput(t *testing.T) {
	for _, action := range []string{"input", "input_log"} {
		if !gameActionMutates(action) {
			t.Fatalf("gameActionMutates(%s) = false, want true", action)
		}
	}
}
