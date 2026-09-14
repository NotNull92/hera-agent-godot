package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/NotNull92/hera-agent-godot/internal/protocol"
)

func TestLogSourceArguments(t *testing.T) {
	for name, parse := range map[string]func([]string) (map[string]any, error){"output": parseOutputArgs, "diagnostics": parseDiagnosticsArgs} {
		t.Run(name, func(t *testing.T) {
			for _, args := range [][]string{{"--source", "editor", "--since", "session:12"}, {"--since", "session:12", "--source", "editor"}} {
				got, err := parse(args)
				if err != nil || !reflect.DeepEqual(got, map[string]any{"source": "editor", "since": "session:12"}) {
					t.Fatalf("parse(%v) = %v, %v", args, got, err)
				}
			}
			for _, args := range [][]string{{"--source"}, {"--source", "other"}, {"--since"}, {"--since", "x"}, {"--source", "file", "--since", "x"}, {"--source", "editor", "--since", ""}} {
				if got, err := parse(args); err == nil {
					t.Errorf("accepted invalid args %v: %v", args, got)
				}
			}
		})
	}
}

func TestEditorLogsNegotiateBeforeSendingToLegacyAddon(t *testing.T) {
	for _, command := range []string{"output", "diagnostics"} {
		for _, capability := range []string{"", "unsupported", "unverified", "supported"} {
			t.Run(command+"/"+capability, func(t *testing.T) {
				var logRequests atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					var req protocol.Request
					if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
						t.Error(err)
						return
					}
					data := map[string]any{"editor_session_id": "session", "capabilities": map[string]string{"editor_log_cursor": capability}}
					if req.Tool != "status" {
						logRequests.Add(1)
						if req.Tool != command || req.Params["source"] != "editor" || req.Params["since"] != "session:12" {
							t.Errorf("unexpected request %+v", req)
						}
						data = map[string]any{"available": true, "source": "editor", "cursor": "session:13"}
					}
					if err := json.NewEncoder(w).Encode(protocol.Response{OK: true, Data: data}); err != nil {
						t.Error(err)
					}
				}))
				t.Cleanup(server.Close)
				installContractHome(t, contractServerPort(t, server), true)
				stdout, stderr, code := captureContractRun([]string{command, "--source", "editor", "--since", "session:12"})
				var data map[string]any
				if err := json.Unmarshal([]byte(stdout), &data); err != nil || code != 0 {
					t.Fatalf("exit=%d stdout=%s stderr=%s err=%v", code, stdout, stderr, err)
				}
				if capability == "supported" {
					if logRequests.Load() != 1 || data["cursor"] != "session:13" {
						t.Fatalf("missing editor request: %v", data)
					}
				} else if logRequests.Load() != 0 || data["available"] != false || data["clean"] != false || data["reason"] != "evidence_unavailable" {
					t.Fatalf("legacy addon falsely used as editor evidence: %v", data)
				}
				if capability == "" && data["capability"] != "unverified" {
					t.Fatalf("missing legacy capability must be unverified: %v", data)
				}
			})
		}
	}
}
