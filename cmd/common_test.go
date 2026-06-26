package cmd

import (
	"strings"
	"testing"

	"github.com/NotNull92/hera-agent-godot/internal/discovery"
)

func TestSelectEditor_reportsActionableMessage_whenNoInstances(t *testing.T) {
	// Given
	instances := []discovery.Instance{}

	// When
	_, err := selectEditor(instances, false, 0)

	// Then
	if err == nil {
		t.Fatal("expected an error for an empty live-editor list")
	}
	message := err.Error()
	for _, want := range []string{"no live Godot editor found", "Hera Agent plugin", "hera instances"} {
		if !strings.Contains(message, want) {
			t.Fatalf("error %q does not include %q", message, want)
		}
	}
}
