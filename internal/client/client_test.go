package client

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
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
