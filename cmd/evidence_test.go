package cmd

import (
	"github.com/NotNull92/hera-agent-godot/internal/client"
	"github.com/NotNull92/hera-agent-godot/internal/protocol"
	"testing"
)

func TestEvidenceParsers(t *testing.T) {
	for index, parse := range []func([]string) (map[string]any, error){parseScreenshotArgs, parseGameScreenshotArgs} {
		args := []string{"--evidence", "--operation-id", "session:123:nonce", "--runtime-session", "runtime"}
		if index == 0 {
			args = append(args, "--runtime")
		}
		p, err := parse(args)
		if err != nil || p["evidence"] != true || p["correlation_operation_id"] != "session:123:nonce" || p["runtime_session_id"] != "runtime" {
			t.Fatalf("capture params=%v err=%v", p, err)
		}
	}
	if _, err := parseSceneArgs([]string{"save", "--evidence", "--expected-sha256", "abc"}); err == nil {
		t.Fatal("invalid hash accepted")
	}
	if p, err := parseResourceArgs([]string{"set", "res://test.tres", "--prop", "resource_name=New", "--evidence"}); err != nil || p["evidence"] != true {
		t.Fatalf("resource evidence: %v %v", p, err)
	}
}

func TestQAEvidenceUnavailableFails(t *testing.T) {
	server := newGameQADiagnoseServer(t, map[string]protocol.Response{"game:screenshot": {OK: true, Data: map[string]any{"evidence": map[string]any{"available": false}}}})
	defer server.Close()
	c := client.New(server.URL)
	result := executeGameQAStep(c, 1, gameQAStep{Tool: "screenshot.runtime", Evidence: true, Covers: []string{"visible"}})
	if result.OK || result.Data == nil {
		t.Fatalf("unavailable evidence must fail and remain inspectable: %+v", result)
	}
	_, ok := gameQASummary(gameQAScenario{Requirements: []string{"visible"}}, []gameQAResult{result}, false)
	if ok {
		t.Fatal("unavailable coverage passed")
	}
}
