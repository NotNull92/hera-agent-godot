package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/NotNull92/hera-agent-godot/internal/dap"
)

func TestParseBreakpoint(t *testing.T) {
	path, line, err := parseBreakpoint("res://scripts/player.gd:42")
	if err != nil || path != "res://scripts/player.gd" || line != 42 {
		t.Fatalf("path=%q line=%d err=%v", path, line, err)
	}
	for _, bad := range []string{"res://a.gd", "res://a.gd:0", "res://a.gd:-3", "scripts/player.gd:12", "res://a.gd:line", "res://../escape.gd:3", "res:///abs.gd:3"} {
		if _, _, err := parseBreakpoint(bad); err == nil {
			t.Fatalf("expected error for %q", bad)
		}
	}
}

func TestResolveBreakpointPath(t *testing.T) {
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(project, "scripts", "player.gd")
	if err := os.WriteFile(script, []byte("extends Node\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	resolved, err := resolveBreakpointPath(project, "res://scripts/player.gd")
	if err != nil || !strings.HasPrefix(resolved, project) {
		t.Fatalf("resolved=%q err=%v", resolved, err)
	}
	if _, err := resolveBreakpointPath(project, "res://scripts/missing.gd"); err == nil {
		t.Fatal("an unreadable script must be refused before configuring breakpoints")
	}
	if _, err := resolveBreakpointPath("", "res://a.gd"); err == nil {
		t.Fatal("a res:// breakpoint without --project must fail")
	}
	if _, err := resolveBreakpointPath(project, "res://../escape.gd"); err == nil {
		t.Fatal("escaping res:// paths must fail")
	}
	absolute := filepath.Join(project, "a.gd")
	if resolved, err := resolveBreakpointPath(project, absolute); err != nil || resolved != absolute {
		t.Fatalf("absolute paths pass through: %q err=%v", resolved, err)
	}
	// A link inside the project that points outside it is refused.
	outside := filepath.Join(t.TempDir(), "secret.gd")
	if err := os.WriteFile(outside, []byte("extends Node\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(project, "scripts", "linked.gd")
	if err := os.Symlink(outside, link); err != nil {
		t.Logf("symlinks unavailable on this host: %v", err)
		return
	}
	if _, err := resolveBreakpointPath(project, "res://scripts/linked.gd"); err == nil {
		t.Fatal("a linked script resolving outside the project must be refused")
	}
}

func TestValidSessionID(t *testing.T) {
	for _, ok := range []string{"1234-1700000000", "abc.DEF-1", "a"} {
		if !validSessionID(ok) {
			t.Fatalf("%q should be a valid session id", ok)
		}
	}
	for _, bad := range []string{"", ".", "..", "../victim", "a/b", `a\b`, "a b", strings.Repeat("x", 65)} {
		if validSessionID(bad) {
			t.Fatalf("%q must be rejected as a session id", bad)
		}
	}
}

func TestParseDebugArgs(t *testing.T) {
	params, err := parseDebugArgs(nil, true)
	if err != nil || params["host"] != "127.0.0.1" || params["port"] != defaultDAPPort || params["wait_ms"] != defaultStopWaitMS {
		t.Fatalf("defaults changed: %v err=%v", params, err)
	}
	params, err = parseDebugArgs([]string{"--break", "res://a.gd:3", "--break=res://a.gd:9", "--break", "res://b.gd:1", "--dap-port", "6055", "--wait-ms", "2500", "--host", "192.168.0.5"}, true)
	if err != nil {
		t.Fatal(err)
	}
	breaks, ok := params["breaks"].([]sessionBreakpoint)
	if !ok || len(breaks) != 3 || breaks[0].Line != 3 || breaks[2].Path != "res://b.gd" {
		t.Fatalf("breakpoints were not collected in order: %v", params["breaks"])
	}
	if params["port"] != 6055 || params["wait_ms"] != 2500 || params["host"] != "192.168.0.5" {
		t.Fatalf("flags were not parsed: %v", params)
	}
	// Explicit values must be distinguishable from defaults.
	if !argsChanged(params, "host") || !argsChanged(params, "port") || argsChanged(params, "frame") || !argsChanged(params, "wait_ms") {
		t.Fatalf("flag presence tracking is wrong: %v", params)
	}
	if _, err := parseDebugArgs([]string{"--break", "res://a.gd:3"}, false); err == nil {
		t.Fatal("--break must be rejected outside attach")
	}
	for _, bad := range [][]string{
		{"--dap-port", "0"},
		{"--dap-port", "nope"},
		{"--wait-ms", "-1"},
		{"--wait-ms", "99999999"},
		{"--frame", "-1"},
		{"--session", "../escape"},
		{"--unknown"},
	} {
		if _, err := parseDebugArgs(bad, true); err == nil {
			t.Fatalf("expected error for %v", bad)
		}
	}
	if _, err := parseDebugArgs([]string{"--session", "../escape"}, true); err == nil {
		t.Fatal("--session must reject traversal")
	}
	// `--` carries an expression that would otherwise look like a flag, and it is
	// only accepted by evaluate.
	params, err = parseDebugArgsWithExpression([]string{"--", "-counter"}, false, true)
	if err != nil || params["expression"] != "-counter" {
		t.Fatalf("-- expression: %v err=%v", params, err)
	}
	if _, err := parseDebugArgs([]string{"--", "-counter"}, false); err == nil {
		t.Fatal("only evaluate may use the expression terminator")
	}
	if _, err := parseDebugArgsWithExpression([]string{"--", "a", "b"}, false, true); err == nil {
		t.Fatal("a second expression must be rejected")
	}
}

func TestDebugSessionRoundTrip(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	root, err := dapSessionRoot()
	if err != nil || !strings.HasPrefix(root, home) {
		t.Fatalf("session root must live under the user home: %q err=%v", root, err)
	}
	session := debugSession{
		ID: "1234-1", Host: "127.0.0.1", Port: 6066, Project: filepath.ToSlash(t.TempDir()),
		Breakpoints: []sessionBreakpoint{{Path: "res://a.gd", Line: 7}},
		AttachedAt:  "2026-10-15T00:00:00Z",
	}
	if err := writeDebugSession(session); err != nil {
		t.Fatal(err)
	}
	loaded, recordPath, err := loadDebugSession("")
	if err != nil {
		t.Fatalf("latest session must resolve: %v", err)
	}
	if loaded.ID != session.ID || loaded.Port != 6066 || len(loaded.Breakpoints) != 1 || loaded.Breakpoints[0].Line != 7 {
		t.Fatalf("session round trip lost data: %+v", loaded)
	}
	if !strings.HasSuffix(recordPath, session.ID+".json") {
		t.Fatalf("unexpected record path: %q", recordPath)
	}
	if _, _, err := loadDebugSession("missing-id"); err == nil {
		t.Fatal("an unknown session id must fail")
	}
	if _, _, err := loadDebugSession("../victim"); err == nil {
		t.Fatal("a traversal id must fail")
	}
	if err := removeDebugSession(recordPath); err != nil {
		t.Fatal(err)
	}
	if _, _, err := loadDebugSession(""); err == nil {
		t.Fatal("a detached record must not be loadable through latest")
	}
	// Detach never writes the shared pointer, including when it names this record.
	if data, err := os.ReadFile(filepath.Join(root, "latest")); err != nil || string(data) != session.ID {
		t.Fatalf("detach must leave latest untouched: %q err=%v", data, err)
	}
	session.ID = "1234-2"
	if err := writeDebugSession(session); err != nil {
		t.Fatal(err)
	}
	if err := removeDebugSession(recordPath); err != nil {
		t.Fatal(err)
	}
	if loaded, _, err := loadDebugSession(""); err != nil || loaded.ID != session.ID {
		t.Fatalf("an older detach must preserve the new latest session: %+v err=%v", loaded, err)
	}
}

func TestLoadDebugSessionRejectsMismatchedRecord(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	root, err := dapSessionRoot()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	// A record whose id does not match its file name must not be trusted: a
	// crafted id could otherwise make detach delete an arbitrary path.
	raw := []byte(`{"id":"../../victim","host":"127.0.0.1","port":6006}`)
	if err := os.WriteFile(filepath.Join(root, "honest.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := loadDebugSession("honest"); err == nil {
		t.Fatal("a record that does not describe itself correctly must be rejected")
	}
	if _, err := os.Stat(filepath.Join(root, "..", "victim.json")); err == nil {
		t.Fatal("no cleanup may have touched a path outside the session store")
	}
}

func TestFileChecksums(t *testing.T) {
	path := filepath.Join(t.TempDir(), "script.gd")
	if err := os.WriteFile(path, []byte("extends SceneTree\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	checksums, err := fileChecksums(path)
	if err != nil || len(checksums) != 2 {
		t.Fatalf("checksums=%v err=%v", checksums, err)
	}
	digests := map[string]string{}
	for _, entry := range checksums {
		digests[entry["algorithm"].(string)] = entry["checksum"].(string)
	}
	// Known digests for exactly "extends SceneTree\n".
	if digests["MD5"] != "86b64ca5f7c9247aab8e931488bbc6da" {
		t.Fatalf("unexpected md5: %v", digests["MD5"])
	}
	if digests["SHA256"] != "23b7461a614328b26bab82978120a8efb299cae65a94076ab259d6bb3d6504df" {
		t.Fatalf("unexpected sha256: %v", digests["SHA256"])
	}
	if _, err := fileChecksums(filepath.Join(t.TempDir(), "absent.gd")); err == nil {
		t.Fatal("missing files must not produce checksums")
	}
}

func TestValidateFrameID(t *testing.T) {
	for _, good := range []float64{0, 1, 4096} {
		if failure := validateFrameID(dap.Message{"id": good}, 0); failure != "" {
			t.Fatalf("id %v should be usable: %s", good, failure)
		}
	}
	bad := []any{nil, "7", -1.0, 0.5, float64(maxFrameID + 1)}
	for _, value := range bad {
		if failure := validateFrameID(dap.Message{"id": value}, 2); failure == "" {
			t.Fatalf("id %#v must be rejected", value)
		}
	}
	if failure := validateFrameID(dap.Message{}, 0); failure == "" {
		t.Fatal("a missing id must be rejected")
	}
}

func TestDapAddressHandlesIPv6(t *testing.T) {
	if got := dapAddress("::1", 6006); got != "[::1]:6006" {
		t.Fatalf("unexpected address: %q", got)
	}
	if got := dapAddress("127.0.0.1", 6006); got != "127.0.0.1:6006" {
		t.Fatalf("unexpected address: %q", got)
	}
}
