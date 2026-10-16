package cmd

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDebugCSharpRefusedBeforeProjectOrConnection(t *testing.T) {
	for _, path := range []string{"res://Missing.cs:12", "res://Missing.CS:1"} {
		stdout, stderr, code := captureContractRun([]string{"debug", "attach", "--project", "does-not-exist", "--dap-port", "1", "--break", path})
		var result map[string]any
		if err := json.Unmarshal([]byte(stdout), &result); err != nil {
			t.Fatal(err, stdout)
		}
		if code != 2 || result["error"] != "csharp_debug_unsupported" || !strings.Contains(stderr, "Rider") || result["docs_url"] == nil || result["guidance"] == nil {
			t.Fatalf("%s: %d %s %s", path, code, stdout, stderr)
		}
	}
}
