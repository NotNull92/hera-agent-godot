package cmd

import (
	"context"
	"testing"
)

func TestDotnetSDKAbsent(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	path, version, state := dotnetSDK(context.Background())
	if path != "" || version != "" || state != "unsupported" {
		t.Fatalf("missing SDK: %q %q %q", path, version, state)
	}
}

func TestStatusSDKObservation(t *testing.T) {
	previous := statusSDKProbe
	t.Cleanup(func() { statusSDKProbe = previous })
	for _, state := range []string{"supported", "unsupported", "unverified"} {
		statusSDKProbe = func(context.Context) (string, string, string) { return "dotnet", "8.0.100", state }
		data := statusWithSDK(map[string]any{"capabilities": map[string]any{"csharp": "unsupported"}}).(map[string]any)
		caps := data["capabilities"].(map[string]any)
		if caps["dotnet_sdk"] != state || caps["csharp"] != "unsupported" {
			t.Fatalf("SDK readiness must not imply editor support: %#v", data)
		}
	}
}
