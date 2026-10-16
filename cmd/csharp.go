package cmd

import (
	"context"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

var dotnetVersionPattern = regexp.MustCompile(`^\d+\.\d+\.\d+(?:[-+][A-Za-z0-9.-]+)?$`)
var statusSDKProbe = dotnetSDK

func dotnetSDK(ctx context.Context) (string, string, string) {
	path, err := exec.LookPath("dotnet")
	if err != nil {
		return "", "", "unsupported"
	}
	run := runEngineBounded(ctx, path, []string{"--version"}, "", nil, 4096)
	version := strings.TrimSpace(run.Output)
	if runFailed(run) || !dotnetVersionPattern.MatchString(version) {
		return path, "", "unverified"
	}
	return path, version, "supported"
}

func statusWithSDK(value any) any {
	data, ok := value.(map[string]any)
	if !ok {
		return value
	}
	caps, ok := data["capabilities"].(map[string]any)
	if !ok {
		return value
	}
	timeout := 2 * time.Second
	if requestTimeout > 0 && requestTimeout < timeout {
		timeout = requestTimeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	path, version, state := statusSDKProbe(ctx)
	caps["dotnet_sdk"] = state
	data["dotnet"] = map[string]any{"path": path, "sdk_version": version, "source": "cli_process"}
	return data
}
