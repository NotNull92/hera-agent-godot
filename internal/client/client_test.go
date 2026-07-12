package client

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/NotNull92/hera-agent-godot/internal/protocol"
)

func TestClientPost_sendsRequestToRpcAndDecodesResponse(t *testing.T) {
	var seen protocol.Request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rpc" {
			t.Errorf("path = %q, want /rpc", r.URL.Path)
		}
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &seen)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"data":{"scene":"res://Main.tscn"}}`))
	}))
	defer srv.Close()

	resp, err := New(srv.URL).Post("status", map[string]any{"k": "v"})
	if err != nil {
		t.Fatalf("Post error: %v", err)
	}

	if seen.Tool != "status" {
		t.Fatalf("server saw tool %q, want status", seen.Tool)
	}
	if !resp.OK {
		t.Fatalf("resp.OK = false, want true (error=%q)", resp.Error)
	}
	data, ok := resp.Data.(map[string]any)
	if !ok || data["scene"] != "res://Main.tscn" {
		t.Fatalf("resp.Data = %#v, want map with scene=res://Main.tscn", resp.Data)
	}
}

func TestClientPost_sendsTokenHeaderWhenSet(t *testing.T) {
	var gotToken string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotToken = r.Header.Get("X-Hera-Token")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := New(srv.URL)
	if _, err := c.Post("status", nil); err != nil {
		t.Fatalf("Post error: %v", err)
	}
	if gotToken != "" {
		t.Fatalf("header sent without token: %q", gotToken)
	}

	c.Token = "s3cret"
	if _, err := c.Post("status", nil); err != nil {
		t.Fatalf("Post error: %v", err)
	}
	if gotToken != "s3cret" {
		t.Fatalf("X-Hera-Token = %q, want s3cret", gotToken)
	}
}

func TestLoadSharedToken_envWinsThenFileThenEmpty(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	t.Setenv(tokenEnvVar, "")
	if got := LoadSharedToken(); got != "" {
		t.Fatalf("no env, no file: token = %q, want empty", got)
	}

	dir := filepath.Join(home, ".hera-agent-godot")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "token"), []byte("  from-file\n"), 0o600); err != nil {
		t.Fatalf("write token: %v", err)
	}
	if got := LoadSharedToken(); got != "from-file" {
		t.Fatalf("file token = %q, want from-file (trimmed)", got)
	}

	t.Setenv(tokenEnvVar, " from-env ")
	if got := LoadSharedToken(); got != "from-env" {
		t.Fatalf("env token = %q, want from-env (trimmed, wins over file)", got)
	}
}

func TestClientPost_setsDefaultTimeout_whenHTTPClientMissing(t *testing.T) {
	var seen protocol.Request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &seen)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL}

	resp, err := c.Post("status", nil)
	if err != nil {
		t.Fatalf("Post error: %v", err)
	}

	if seen.Tool != "status" {
		t.Fatalf("server saw tool %q, want status", seen.Tool)
	}
	if !resp.OK {
		t.Fatalf("resp.OK = false, want true (error=%q)", resp.Error)
	}
	if c.HTTP == nil {
		t.Fatal("Client.HTTP = nil, want default timeout client")
	}
	if c.HTTP.Timeout != defaultTimeout {
		t.Fatalf("Client.HTTP.Timeout = %s, want %s", c.HTTP.Timeout, defaultTimeout)
	}
}
