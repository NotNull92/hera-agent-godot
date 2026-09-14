package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/NotNull92/hera-agent-godot/internal/protocol"
)

const guardJSON = `{"editor_session_id":"session","scene":"res://Main.tscn","node_instance_id":"9007199254740993","prop":"visible","type":"bool","value":"true"}`

func TestNodeGuardArguments(t *testing.T) {
	if _, err := parseNodeArgs([]string{"get", ".", "--prop", "visible", "--snapshot"}); err != nil {
		t.Fatal(err)
	}
	if _, err := parseNodeArgs([]string{"get", ".", "--snapshot"}); err == nil {
		t.Fatal("snapshot without property accepted")
	}
	base := []string{"set", ".", "--prop", "visible", "--value", "false"}
	got, err := parseNodeArgs(append(append([]string{}, base...), "--expected", guardJSON, "--verify"))
	if err != nil || got["expected"] == nil || got["verify"] != true {
		t.Fatalf("guard parse = %v, %v", got, err)
	}
	for _, extra := range [][]string{{"--verify"}, {"--expected"}, {"--expected", "{}"}, {"--expected", "null"}, {"--expected", guardJSON + `{}`}, {"--expected", strings.Replace(guardJSON, `"9007199254740993"`, `9007199254740993`, 1)}, {"--expected", strings.Replace(guardJSON, `"value":"true"`, `"value":null`, 1)}, {"--expected", strings.Replace(guardJSON, `"prop":"visible"`, `"prop":"name"`, 1)}, {"--expected", strings.Replace(guardJSON, `"value":"true"`, `"value":"true","extra":"x"`, 1)}} {
		if _, err := parseNodeArgs(append(append([]string{}, base...), extra...)); err == nil {
			t.Errorf("accepted invalid guard: %v", extra)
		}
	}
}

func TestBatchGuardNegotiatesBeforeAnyCommand(t *testing.T) {
	var mutations atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req protocol.Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
			return
		}
		if req.Tool != "status" {
			mutations.Add(1)
		}
		if err := json.NewEncoder(w).Encode(protocol.Response{OK: true, Data: map[string]any{}}); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	installContractHome(t, contractServerPort(t, server), true)
	file := filepath.Join(t.TempDir(), "guard.json")
	if err := os.WriteFile(file, []byte(`[{"tool":"node","params":{"action":"set","path":".","prop":"visible","value":"false"}},{"tool":"node","params":{"action":"set","expected":`+guardJSON+`}}]`), 0600); err != nil {
		t.Fatal(err)
	}
	_, stderr, code := captureContractRun([]string{"batch", "--file", file})
	if code != 1 || mutations.Load() != 0 || !strings.Contains(stderr, "capability_unavailable") {
		t.Fatalf("batch bypassed guard: %d %s %d", code, stderr, mutations.Load())
	}
}

func TestGuardRejectsAddonReplacedAfterPreflight(t *testing.T) {
	var mutations atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req protocol.Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
			return
		}
		response := protocol.Response{OK: true, Data: map[string]any{"capabilities": map[string]string{"node_set_guard": "supported"}}}
		if req.Tool == "node" {
			if req.Params["action"] == "set" {
				mutations.Add(1)
			} else {
				response = protocol.Response{OK: false, Error: "unknown node action"}
			}
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	installContractHome(t, contractServerPort(t, server), true)
	_, _, code := captureContractRun([]string{"node", "set", ".", "--prop", "visible", "--value", "false", "--expected", guardJSON})
	if code != 1 || mutations.Load() != 0 {
		t.Fatalf("replacement addon ignored guard: exit=%d mutations=%d", code, mutations.Load())
	}
}

func TestNodeGuardRequiresExplicitEditorWhenMultipleLive(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if err := json.NewEncoder(w).Encode(protocol.Response{OK: true, Data: map[string]any{"capabilities": map[string]string{"node_set_guard": "supported"}}}); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	installContractHome(t, contractServerPort(t, server), true)
	dir := filepath.Join(os.Getenv("USERPROFILE"), ".hera-agent-godot", "instances")
	raw, err := os.ReadFile(filepath.Join(dir, "49928.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "49929.json"), []byte(strings.ReplaceAll(string(raw), "49928", "49929")), 0600); err != nil {
		t.Fatal(err)
	}
	args := []string{"node", "set", ".", "--prop", "visible", "--value", "false", "--expected", guardJSON}
	_, stderr, code := captureContractRun(args)
	if code != 1 || requests.Load() != 0 || !strings.Contains(stderr, "multiple live Godot editors") {
		t.Fatalf("ambiguous guard: %d %s %d", code, stderr, requests.Load())
	}
	_, stderr, code = captureContractRun(append([]string{"--instance", "49929"}, args...))
	if code != 0 || requests.Load() != 2 {
		t.Fatalf("explicit guard: %d %s %d", code, stderr, requests.Load())
	}
}

func TestNodeGuardNegotiatesBeforeMutation(t *testing.T) {
	for _, capability := range []string{"", "unsupported", "unverified", "supported", "status_error"} {
		t.Run(capability, func(t *testing.T) {
			var mutations atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var req protocol.Request
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Error(err)
					return
				}
				response := protocol.Response{OK: true, Data: map[string]any{"capabilities": map[string]string{"node_set_guard": capability}}}
				if req.Tool == "status" && capability == "status_error" {
					response = protocol.Response{OK: false, Error: "unauthorized"}
				} else if req.Tool != "status" {
					mutations.Add(1)
					if req.Params["expected"].(map[string]any)["node_instance_id"] != "9007199254740993" {
						t.Error("instance ID lost precision")
					}
					response.Data = map[string]string{"verification": "passed"}
				}
				if err := json.NewEncoder(w).Encode(response); err != nil {
					t.Error(err)
				}
			}))
			t.Cleanup(server.Close)
			installContractHome(t, contractServerPort(t, server), true)
			stdout, stderr, code := captureContractRun([]string{"node", "set", ".", "--prop", "visible", "--value", "false", "--expected", guardJSON, "--verify"})
			if capability == "supported" {
				if code != 0 || mutations.Load() != 1 || !strings.Contains(stdout, "passed") {
					t.Fatalf("missing guarded mutation: %d %s %s", code, stdout, stderr)
				}
			} else {
				want := "capability_unavailable"
				if capability == "status_error" {
					want = "unauthorized"
				}
				if code != 1 || mutations.Load() != 0 || stdout != "" || !strings.Contains(stderr, want) {
					t.Fatalf("unsafe negotiation: %d %s %s mutations=%d", code, stdout, stderr, mutations.Load())
				}
			}
		})
	}
}
