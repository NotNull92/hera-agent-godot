package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
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

func csharpEngineSupported(version engineVersion) bool {
	parts := strings.Split(version.String, ".")
	if len(parts) < 2 || !version.Mono {
		return false
	}
	major, _ := strconv.Atoi(parts[0])
	minor, _ := strconv.Atoi(parts[1])
	return major == 4 && minor >= 7
}

func csharpAssembly(project string) (string, error) {
	data, err := os.ReadFile(filepath.Join(project, "project.godot"))
	if err != nil {
		return "", err
	}
	section := ""
	name := ""
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[") {
			section = line
			continue
		}
		if section == "[dotnet]" && strings.HasPrefix(line, "project/assembly_name=") {
			name, err = strconv.Unquote(strings.TrimSpace(strings.TrimPrefix(line, "project/assembly_name=")))
			if err != nil {
				return "", err
			}
		}
	}
	if name == "" {
		csproj, err := csharpProject(project)
		if err != nil {
			return "", err
		}
		name = strings.TrimSuffix(filepath.Base(csproj), ".csproj")
	}
	if name == "." || name == ".." || strings.ContainsAny(name, `/\:`) {
		return "", fmt.Errorf("invalid project assembly name")
	}
	path := filepath.Join(project, ".godot", "mono", "temp", "bin", "Debug", name+".dll")
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return "", fmt.Errorf("csharp_build_required: build the project assembly before running C#")
	}
	return path, nil
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
