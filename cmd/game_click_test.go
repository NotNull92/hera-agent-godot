package cmd

import "testing"

func TestParseGameArgs_returnsClickParams_whenCoordinatesProvided(t *testing.T) {
	// Given
	args := []string{"click", "--x", "120", "--y", "240"}

	// When
	got, err := parseGameArgs(args)

	// Then
	if err != nil {
		t.Fatalf("parseGameArgs error: %v", err)
	}
	if got["action"] != "click" {
		t.Fatalf("action = %v, want click", got["action"])
	}
	if got["x"] != 120 || got["y"] != 240 {
		t.Fatalf("click position = (%v, %v), want (120, 240)", got["x"], got["y"])
	}
}

func TestParseGameArgs_returnsClickParams_whenNodeProvided(t *testing.T) {
	// Given
	args := []string{"click", "--node", "C:/Program Files/Git/root/Main/Restart"}

	// When
	got, err := parseGameArgs(args)

	// Then
	if err != nil {
		t.Fatalf("parseGameArgs error: %v", err)
	}
	if got["action"] != "click" {
		t.Fatalf("action = %v, want click", got["action"])
	}
	if got["path"] != "/root/Main/Restart" {
		t.Fatalf("path = %v, want /root/Main/Restart", got["path"])
	}
}

func TestParseGameArgs_returnsClickParams_whenTextProvided(t *testing.T) {
	// Given
	args := []string{"click", "--text", "Restart"}

	// When
	got, err := parseGameArgs(args)

	// Then
	if err != nil {
		t.Fatalf("parseGameArgs error: %v", err)
	}
	if got["action"] != "click" {
		t.Fatalf("action = %v, want click", got["action"])
	}
	if got["text"] != "Restart" {
		t.Fatalf("text = %v, want Restart", got["text"])
	}
}

func TestParseGameArgs_rejectsClick_whenCoordinatesInvalid(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "missing x", args: []string{"click", "--y", "240"}},
		{name: "missing y", args: []string{"click", "--x", "120"}},
		{name: "non integer", args: []string{"click", "--x", "left", "--y", "240"}},
		{name: "negative", args: []string{"click", "--x", "-1", "--y", "240"}},
		{name: "unknown flag", args: []string{"click", "--x", "120", "--y", "240", "--bad"}},
		{name: "node and x conflict", args: []string{"click", "--node", "/root/Main/Button", "--x", "120", "--y", "240"}},
		{name: "node and text conflict", args: []string{"click", "--node", "/root/Main/Button", "--text", "Restart"}},
		{name: "empty node", args: []string{"click", "--node", ""}},
		{name: "empty text", args: []string{"click", "--text", ""}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When
			_, err := parseGameArgs(tt.args)

			// Then
			if err == nil {
				t.Fatal("expected click parse error")
			}
		})
	}
}

func TestGameActionMutates_returnsTrue_forClick(t *testing.T) {
	// When
	got := gameActionMutates("click")

	// Then
	if !got {
		t.Fatal("gameActionMutates(click) = false, want true")
	}
}
