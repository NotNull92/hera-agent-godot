package cmd

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/NotNull92/hera-agent-godot/internal/client"
)

func TestParseRunArgs(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantAction string
		wantScene  string
		wantWait   bool
		wantErr    bool
	}{
		{name: "default plays main", args: nil, wantAction: "play_main"},
		{name: "current", args: []string{"--current"}, wantAction: "play_current"},
		{name: "custom scene", args: []string{"--scene", "res://Main.tscn"}, wantAction: "play_custom", wantScene: "res://Main.tscn"},
		{name: "scene with wait", args: []string{"--scene", "res://Main.tscn", "--wait"}, wantAction: "play_custom", wantScene: "res://Main.tscn", wantWait: true},
		{name: "wait alone", args: []string{"--wait"}, wantAction: "play_main", wantWait: true},
		{name: "scene and current conflict", args: []string{"--scene", "res://X.tscn", "--current"}, wantErr: true},
		{name: "scene without value", args: []string{"--scene"}, wantErr: true},
		{name: "unknown flag", args: []string{"--nope"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params, wait, err := parseRunArgs(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got params=%v", params)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := params["action"]; got != tt.wantAction {
				t.Errorf("action = %v, want %v", got, tt.wantAction)
			}
			if tt.wantScene != "" && params["scene"] != tt.wantScene {
				t.Errorf("scene = %v, want %v", params["scene"], tt.wantScene)
			}
			if wait != tt.wantWait {
				t.Errorf("wait = %v, want %v", wait, tt.wantWait)
			}
		})
	}
}

func TestPollPlaying_returnsTimeoutError_whenDesiredStateNotReached(t *testing.T) {
	// Given
	c := newRunStateClient(t, `{"ok":true,"data":{"playing":false,"scene":""}}`)

	// When
	resp, err := pollPlaying(c, true, 0)

	// Then
	if err == nil {
		t.Fatalf("err = nil, want timeout error with last response %#v", resp)
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("err = %q, want timeout error", err)
	}
	if resp == nil || playingFlag(resp) {
		t.Fatalf("resp = %#v, want last non-playing state", resp)
	}
}

func TestPollPlaying_returnsToolError_whenStateRequestFails(t *testing.T) {
	// Given
	c := newRunStateClient(t, `{"ok":false,"error":"state failed"}`)

	// When
	resp, err := pollPlaying(c, true, time.Second)

	// Then
	if err == nil {
		t.Fatalf("err = nil, want tool error with response %#v", resp)
	}
	if !strings.Contains(err.Error(), "state failed") {
		t.Fatalf("err = %q, want tool error", err)
	}
}

func newRunStateClient(t *testing.T, response string) *client.Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(response))
	}))
	t.Cleanup(srv.Close)
	return client.New(srv.URL)
}
