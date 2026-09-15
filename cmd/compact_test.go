package cmd

import (
	"reflect"
	"testing"
)

func TestCompactStatusDataOmitsExperimentalDetail(t *testing.T) {
	in := map[string]any{
		"pid":               1,
		"godot_version":     "4.7",
		"godot_commit":      "abc",
		"capabilities":      map[string]any{"dap": "unverified"},
		"csharp_supported":  true,
		"editor_session_id": "session",
	}
	got, _ := compactStatusData(in).(map[string]any)
	if _, ok := got["capabilities"]; ok {
		t.Fatal("compact status still has capabilities")
	}
	if _, ok := got["godot_commit"]; ok {
		t.Fatal("compact status still has godot_commit")
	}
	if got["pid"] != 1 || got["csharp_supported"] != true || got["editor_session_id"] != "session" {
		t.Fatalf("compact status dropped stable fields: %v", got)
	}
	if _, ok := in["capabilities"]; !ok {
		t.Fatal("compact status mutated the original map")
	}
}

func TestCompactReceiptDataOmitsRetentionAndEchoedResponse(t *testing.T) {
	in := map[string]any{
		"id":        "session:1:x",
		"lifecycle": "completed",
		"retention": map[string]any{"durable": false, "expired_count": 1},
		"evidence":  map[string]any{"complete": true, "response": map[string]any{"ok": true}},
	}
	got, _ := compactReceiptData(in).(map[string]any)
	if _, ok := got["retention"]; ok {
		t.Fatal("compact receipt still has retention")
	}
	evidence, _ := got["evidence"].(map[string]any)
	if evidence["complete"] != true {
		t.Fatalf("compact receipt dropped evidence.complete: %v", evidence)
	}
	if _, ok := evidence["response"]; ok {
		t.Fatal("compact receipt still echoes evidence.response")
	}
	if !reflect.DeepEqual(in["retention"], map[string]any{"durable": false, "expired_count": 1}) {
		t.Fatal("compact receipt mutated the original map")
	}
}

func TestParseStatusArgs(t *testing.T) {
	detail, err := parseStatusArgs(nil)
	if err != nil || detail {
		t.Fatalf("empty args: detail=%v err=%v", detail, err)
	}
	detail, err = parseStatusArgs([]string{"--capabilities"})
	if err != nil || !detail {
		t.Fatalf("capabilities: detail=%v err=%v", detail, err)
	}
	if _, err := parseStatusArgs([]string{"--verbose"}); err == nil {
		t.Fatal("accepted unknown flag")
	}
}

func TestPrintIDsWritesControlPaths(t *testing.T) {
	// Covered via printPathList through printIDs; keep the extractor explicit.
	data := map[string]any{"controls": []any{
		map[string]any{"path": "/root/Main/Start", "name": "Start"},
		map[string]any{"path": "/root/Main/Quit"},
	}}
	if !hasPathList(data, "controls") {
		t.Fatal("controls path list not detected")
	}
	if hasPathList(data, "nodes") {
		t.Fatal("nodes list should be absent")
	}
}
