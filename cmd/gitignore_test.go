package cmd

import (
	"os"
	"strings"
	"testing"
)

func TestGitignore_ignoresLocalHeraBuildArtifacts(t *testing.T) {
	// Given
	raw, err := os.ReadFile("../.gitignore")
	if err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}
	content := string(raw)

	// Then
	for _, want := range []string{"/hera", "/hera.exe"} {
		if !strings.Contains(content, want) {
			t.Fatalf(".gitignore does not include %q", want)
		}
	}
}
