package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/NotNull92/hera-agent-godot/internal/client"
	"github.com/NotNull92/hera-agent-godot/internal/protocol"
)

func TestPostGameQAStep_returnsRejectedLifecycleStepWithoutPolling(t *testing.T) {
	tests := []struct {
		name string
		step gameQAStep
	}{
		{name: "run", step: gameQAStep{Tool: "run", Current: true, Wait: true}},
		{name: "stop", step: gameQAStep{Tool: "stop", Wait: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				if err := json.NewEncoder(w).Encode(protocol.Response{OK: false, Error: "play rejected"}); err != nil {
					t.Errorf("encode response: %v", err)
				}
			}))
			defer server.Close()

			resp, err := postGameQAStep(client.New(server.URL), tt.step)

			if err != nil {
				t.Fatalf("postGameQAStep() error = %v, want nil", err)
			}
			if resp == nil || resp.OK || resp.Error != "play rejected" {
				t.Fatalf("postGameQAStep() response = %#v, want rejected lifecycle response", resp)
			}
			if requests != 1 {
				t.Fatalf("request count = %d, want 1", requests)
			}
		})
	}
}

func TestPostGameQAStep_rejectsOverflowingWaitDuration(t *testing.T) {
	if strconv.IntSize < 64 {
		t.Skip("duration overflow requires a 64-bit int")
	}
	_, err := postGameQAStep(nil, gameQAStep{Tool: "wait", DurationMS: int(maxDurationMilliseconds + 1)})

	if err == nil {
		t.Fatal("postGameQAStep() accepted an overflowing wait duration")
	}
}

func TestPostGameQAStepTargetsSelectedRuntime(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request protocol.Request
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if request.Params["pid"] != float64(202) {
			t.Errorf("pid = %v, want 202", request.Params["pid"])
		}
		if err := json.NewEncoder(w).Encode(protocol.Response{OK: true}); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	t.Cleanup(server.Close)

	_, err := postGameQAStep(client.New(server.URL), gameQAStep{Tool: "game.node.get", Path: "/root/Main", targetPID: 202})
	if err != nil {
		t.Fatalf("postGameQAStep() error = %v", err)
	}
}

func TestPostGameQAStep_waitsForRuntimeInstancesAfterStop(t *testing.T) {
	expected := []struct {
		tool   string
		action string
		resp   protocol.Response
	}{
		{tool: "run", action: "stop", resp: protocol.Response{OK: true}},
		{tool: "run", action: "state", resp: protocol.Response{OK: true, Data: map[string]any{"playing": false}}},
		{tool: "game", action: "instances", resp: protocol.Response{OK: true, Data: map[string]any{"instances": []any{}}}},
	}
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request protocol.Request
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode request: %v", err)
			return
		}
		if requests >= len(expected) {
			t.Errorf("unexpected request %q", request.Tool)
			return
		}
		want := expected[requests]
		requests++
		if request.Tool != want.tool || request.Params["action"] != want.action {
			t.Errorf("request = %q/%v, want %q/%q", request.Tool, request.Params["action"], want.tool, want.action)
		}
		if err := json.NewEncoder(w).Encode(want.resp); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	defer server.Close()

	resp, err := postGameQAStep(client.New(server.URL), gameQAStep{Tool: "stop", Wait: true})

	if err != nil {
		t.Fatalf("postGameQAStep() error = %v, want nil", err)
	}
	if resp == nil || !resp.OK {
		t.Fatalf("postGameQAStep() response = %#v, want successful stop response", resp)
	}
	if requests != len(expected) {
		t.Fatalf("request count = %d, want %d", requests, len(expected))
	}
}

func TestPostGameQAStepWaitsForRequestedLifecycleAction(t *testing.T) {
	for _, tc := range []struct {
		name string
		step gameQAStep
		want []string
	}{
		{"stop alias", gameQAStep{Tool: "run", Action: "stop", Wait: true}, []string{"run/stop", "run/state", "game/instances"}},
		{"state snapshot", gameQAStep{Tool: "run", Action: "state", Wait: true}, []string{"run/state"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Given: a stopped editor and no runtime processes.
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var request protocol.Request
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Error(err)
					return
				}
				action, _ := request.Params["action"].(string)
				actual := request.Tool + "/" + action
				if requests >= len(tc.want) || actual != tc.want[requests] {
					t.Errorf("unexpected lifecycle request %s at index %d", actual, requests)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				requests++
				data := map[string]any{"playing": false, "instances": []any{}}
				if err := json.NewEncoder(w).Encode(protocol.Response{OK: true, Data: data}); err != nil {
					t.Error(err)
				}
			}))
			t.Cleanup(server.Close)
			// When
			resp, err := postGameQAStep(client.New(server.URL), tc.step)
			// Then
			if err != nil || resp == nil || !resp.OK || requests != len(tc.want) {
				t.Fatalf("response=%v error=%v requests=%d, want %d", resp, err, requests, len(tc.want))
			}
		})
	}
}
