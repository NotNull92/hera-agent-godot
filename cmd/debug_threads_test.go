package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/NotNull92/hera-agent-godot/internal/dap"
)

func threadAdapter(t *testing.T, handle func(dap.Message) []map[string]any) *dap.Client {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
		reader := bufio.NewReader(conn)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			var size int
			if _, err := fmt.Sscanf(line, "Content-Length: %d", &size); err != nil {
				t.Error(err)
				return
			}
			if _, err := reader.ReadString('\n'); err != nil {
				return
			}
			body := make([]byte, size)
			if _, err := io.ReadFull(reader, body); err != nil {
				return
			}
			var req dap.Message
			if err := json.Unmarshal(body, &req); err != nil {
				t.Error(err)
				return
			}
			for _, message := range handle(req) {
				if message["type"] == nil {
					message["type"], message["request_seq"], message["command"], message["success"] = "response", req["seq"], req["command"], true
				}
				raw, err := json.Marshal(message)
				if err != nil {
					t.Error(err)
					return
				}
				if _, err := fmt.Fprintf(conn, "Content-Length: %d\r\n\r\n%s", len(raw), raw); err != nil {
					return
				}
			}
		}
	}()
	client, err := dap.Dial(listener.Addr().String(), time.Second)
	if err != nil {
		listener.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() { client.Close(); listener.Close(); <-done })
	return client
}

func TestDebugThreadDiscovery(t *testing.T) {
	for _, tc := range []struct {
		name                  string
		ids                   []int
		selected, event, want int
		fail                  bool
	}{
		{name: "non-one", ids: []int{7}, want: 7},
		{name: "explicit", ids: []int{7, 9}, selected: 9, want: 9},
		{name: "stopped event", ids: []int{7, 9}, event: 9, want: 9},
		{name: "ambiguous", ids: []int{7, 9}, fail: true},
		{name: "missing selection", ids: []int{7}, selected: 9, fail: true},
		{name: "empty", fail: true},
		{name: "duplicate", ids: []int{7, 7}, fail: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := threadAdapter(t, func(req dap.Message) []map[string]any {
				switch req.String("command") {
				case "threads":
					ids := []map[string]any{}
					for _, id := range tc.ids {
						ids = append(ids, map[string]any{"id": id, "name": "worker"})
					}
					messages := []map[string]any{}
					if tc.event != 0 {
						messages = append(messages, map[string]any{"type": "event", "event": "stopped", "body": map[string]any{"threadId": tc.event}})
					}
					return append(messages, map[string]any{"body": map[string]any{"threads": ids}})
				case "stackTrace":
					if tc.fail || req.Map("arguments").Int("threadId") != tc.want {
						t.Errorf("wrong stack target: %v", req)
					}
					return []map[string]any{{"body": map[string]any{"stackFrames": []map[string]any{{"id": 42, "name": "frame", "line": 1}}}}}
				default:
					t.Errorf("unexpected request %v", req)
					return nil
				}
			})
			state, err := observeStop(client, time.Second, tc.selected)
			if tc.fail {
				if err == nil {
					t.Fatal("ambiguous or missing thread accepted")
				}
				return
			}
			if err != nil || state.ThreadID != tc.want || !state.Stopped {
				t.Fatalf("state=%+v err=%v", state, err)
			}
		})
	}
}

func TestDebugResumeUsesStoppedEventThread(t *testing.T) {
	client := threadAdapter(t, func(req dap.Message) []map[string]any {
		switch req.String("command") {
		case "threads":
			return []map[string]any{{"body": map[string]any{"threads": []map[string]any{{"id": 7}, {"id": 9}}}}}
		case "next":
			if req.Map("arguments").Int("threadId") != 7 {
				t.Errorf("wrong step target: %v", req)
			}
			return []map[string]any{{}, {"type": "event", "event": "continued", "body": map[string]any{"threadId": 7}}, {"type": "event", "event": "stopped", "body": map[string]any{"threadId": 9, "reason": "breakpoint"}}}
		case "stackTrace":
			if req.Map("arguments").Int("threadId") != 9 {
				t.Errorf("read wrong stopped thread: %v", req)
			}
			return []map[string]any{{"body": map[string]any{"stackFrames": []map[string]any{{"id": 99}}}}}
		default:
			t.Errorf("unexpected request: %v", req)
			return nil
		}
	})
	state, err := resumeAndObserve(client, "next", time.Second, 7)
	if err != nil || state.ThreadID != 9 || !state.ResumeConfirmed {
		t.Fatalf("state=%+v err=%v", state, err)
	}
}

func TestParseDebugThread(t *testing.T) {
	params, err := parseDebugArgs([]string{"--thread", "9"}, false)
	if err != nil || intValue(params, "thread", 0) != 9 {
		t.Fatalf("%v %v", params, err)
	}
	for _, value := range []string{"0", "-1", "1.5", strings.Repeat("9", 30)} {
		if _, err := parseDebugArgs([]string{"--thread", value}, false); err == nil {
			t.Fatalf("accepted %q", value)
		}
	}
}

func TestDebugThreadIDRejectsMalformedEvidence(t *testing.T) {
	for _, raw := range []any{nil, "9", float64(-1), float64(0), 9.5, float64(1 << 31)} {
		if _, err := debugThreadID(raw); err == nil {
			t.Fatalf("accepted malformed id %v", raw)
		}
	}
	if id, err := debugThreadID(float64(maxDebugThreadID)); err != nil || id != maxDebugThreadID {
		t.Fatalf("valid int32 rejected: %d %v", id, err)
	}
}

func TestVariablesRetryKeepsExplicitThreadAndNeverResumes(t *testing.T) {
	scopes := 0
	client := threadAdapter(t, func(req dap.Message) []map[string]any {
		switch req.String("command") {
		case "threads":
			return []map[string]any{{"body": map[string]any{"threads": []map[string]any{{"id": 7}, {"id": 9}}}}}
		case "stackTrace":
			if req.Map("arguments").Int("threadId") != 9 {
				t.Errorf("wrong variable thread: %v", req)
			}
			return []map[string]any{{"body": map[string]any{"stackFrames": []map[string]any{{"id": 99}}}}}
		case "scopes":
			if req.Map("arguments").Int("frameId") != 99 {
				t.Errorf("wrong variable frame: %v", req)
			}
			scopes++
			if scopes == 1 {
				return []map[string]any{{"type": "response", "command": "scopes", "request_seq": req["seq"], "success": false, "message": "stale"}}
			}
			return []map[string]any{{"body": map[string]any{"scopes": []map[string]any{{"name": "locals", "variablesReference": 42}}}}}
		case "variables":
			return []map[string]any{{"body": map[string]any{"variables": []map[string]any{{"name": "counter", "value": "3"}}}}}
		default:
			t.Errorf("retry must not resume: %v", req)
			return nil
		}
	})
	values, truncated, err := readVariables(client, 0, time.Second, 9)
	if err != nil || truncated || len(values) != 1 || values[0]["value"] != "3" {
		t.Fatalf("values=%v truncated=%v err=%v", values, truncated, err)
	}
}
