package cmd

import "testing"

func TestVersion_returnsDevByDefault(t *testing.T) {
	if got := Version(); got != "dev" {
		t.Fatalf("Version() = %q, want dev", got)
	}
}

func TestExecute_versionCommand(t *testing.T) {
	code := Execute([]string{"version"})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
}

func TestExecute_versionFlag(t *testing.T) {
	code := Execute([]string{"--version"})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
}
