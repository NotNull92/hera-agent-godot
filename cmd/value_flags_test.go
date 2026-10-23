package cmd

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestValueFlagEqualsAcrossCommandFamilies(t *testing.T) {
	cases := []struct {
		name  string
		parse func([]string) (map[string]any, error)
		args  []string
	}{
		{"node get", parseNodeArgs, []string{"get", ".", "--prop", "visible"}},
		{"node set", parseNodeArgs, []string{"set", ".", "--prop", "name", "--value", "A=B"}},
		{"node add", parseNodeArgs, []string{"add", "Node", "--name", "Child"}},
		{"scene", parseSceneArgs, []string{"create", "res://test.tscn", "--root", "Node2D"}},
		{"game node", parseGameArgs, []string{"node", "get", ".", "--props", "position,visible"}},
		{"game input", parseGameInputArgs, []string{"key", "--key", "A", "--keycode", "65", "--modifiers", "shift,ctrl", "--press"}},
		{"game mouse", parseGameInputArgs, []string{"mouse", "--x", "1", "--y", "2", "--dx", "-2", "--button", "left", "--click"}},
		{"game axis", parseGameInputArgs, []string{"axis", "--axis", "LEFT_X", "--value", "-0.5"}},
		{"game click", parseGameClickArgs, []string{"--text", "Play"}},
		{"game clock", parseGameClockArgs, []string{"--time-scale", "0.5"}},
		{"game ui", parseGameUIArgs, []string{"tree", "--depth", "2", "--type", "Button"}},
		{"game audit", parseGameUIArgs, []string{"audit", "--limit", "5"}},
		{"game log", parseGameInputLogArgs, []string{"--limit", "5"}},
		{"output", parseOutputArgs, []string{"--source", "editor", "--since", "session:1", "--lines", "5"}},
		{"diagnostics", parseDiagnosticsArgs, []string{"--lines", "5"}},
		{"project", parseProjectListFilesArgs, []string{"--type", "script", "--limit", "4"}},
		{"resource", parseResourceArgs, []string{"set", "res://x.tres", "--prop", "resource_name=x"}},
		{"resource create", parseResourceArgs, []string{"create", "Resource", "res://x.tres", "--prop", "resource_name=x", "--force"}},
		{"script", parseScriptArgs, []string{"create", "res://x.gd", "--extends", "Node", "--export", "hp:int=5"}},
		{"script open", parseScriptArgs, []string{"open", "res://x.gd", "--line", "2"}},
		{"theme", parseThemeArgs, []string{"set", "res://x.tres", "--type", "Button", "--constant", "outline_size=2"}},
		{"screenshot", parseScreenshotArgs, []string{"--path", "user://x.png", "--width", "100"}},
		{"capture evidence", parseScreenshotArgs, []string{"--runtime", "--runtime-session", "test"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			combined := joinValueFlags(tc.args)
			want, err := tc.parse(tc.args)
			if err != nil {
				t.Fatal(err)
			}
			got, err := tc.parse(combined)
			if err != nil || !reflect.DeepEqual(got, want) {
				t.Fatalf("%v: got %v, %v; want %v", combined, got, err, want)
			}
		})
	}
	for _, parse := range []func([]string) (string, bool, error){parseBatchFlags, parseGameQAFlags} {
		file, keep, err := parse([]string{"--file=scenario.json", "--continue"})
		if err != nil || file != "scenario.json" || !keep {
			t.Fatalf("file flags: %q %v %v", file, keep, err)
		}
	}
	params, wait, err := parseRunArgs([]string{"--scene=res://main.tscn", "--wait"})
	if err != nil || !wait || params["scene"] != "res://main.tscn" {
		t.Fatalf("run: %v %v", params, err)
	}
	_, _, threshold, err := parseScreenshotDiffArgs([]string{"a.png", "b.png", "--threshold=12"})
	if err != nil || threshold != 12 {
		t.Fatalf("diff: %d %v", threshold, err)
	}
	options, err := parseGameQADiagnoseArgs([]string{"--lines=5", "--max-errors=0", "--max-warnings=2", "--path=user://x.png"})
	if err != nil || options.lines != 5 || options.maxWarnings != 2 || options.screenshotPath != "user://x.png" {
		t.Fatalf("diagnose: %+v %v", options, err)
	}
	pid, rest, err := parseGameInvocation([]string{"--pid=123", "tree"})
	if err != nil || pid != 123 || !reflect.DeepEqual(rest, []string{"tree"}) {
		t.Fatalf("pid: %d %v %v", pid, rest, err)
	}
	path := filepath.Join(t.TempDir(), "events.json")
	if err := os.WriteFile(path, []byte(`[{"frame":0,"action":"jump","pressed":true}]`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := parseGameInputSequenceArgs([]string{"--file=" + path}); err != nil {
		t.Fatal(err)
	}
}

func joinValueFlags(args []string) []string {
	var result []string
	for i := 0; i < len(args); i++ {
		if strings.HasPrefix(args[i], "--") && i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
			result = append(result, args[i]+"="+args[i+1])
			i++
		} else {
			result = append(result, args[i])
		}
	}
	return result
}
