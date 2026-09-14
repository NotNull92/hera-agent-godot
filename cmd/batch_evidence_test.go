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

func batchEvidenceFile(t *testing.T, tool string, params map[string]any) string {
	t.Helper()
	commands := []any{
		map[string]any{"tool": "scene", "params": map[string]any{"action": "tree"}},
		map[string]any{"tool": tool, "params": params},
	}
	raw, err := json.Marshal(commands)
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "batch.json")
	if err := os.WriteFile(file, raw, 0600); err != nil {
		t.Fatal(err)
	}
	return file
}

func TestBatchEvidenceGuardsLegacyAndReplacedAddon(t *testing.T) {
	for _, tool := range []string{"scene", "resource"} {
		for _, optIn := range []string{"evidence", "expected_sha256"} {
			for _, replaced := range []bool{false, true} {
				name := tool + "/" + optIn + "/legacy"
				if replaced {
					name = tool + "/" + optIn + "/replaced"
				}
				t.Run(name, func(t *testing.T) {
					action := "save"
					if tool == "resource" {
						action = "set"
					}
					var statusCalls, batchCalls, mutations atomic.Int32
					server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						var req protocol.Request
						if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
							t.Error(err)
							return
						}
						response := protocol.Response{OK: true, Data: map[string]any{}}
						if req.Tool == "status" {
							statusCalls.Add(1)
							if replaced {
								response.Data = map[string]any{"capabilities": map[string]any{"linked_evidence": "supported"}}
							}
						} else if req.Tool == "batch" {
							batchCalls.Add(1)
							commands := req.Params["commands"].([]any)
							params := commands[1].(map[string]any)["params"].(map[string]any)
							child := map[string]any{"tool": tool, "ok": true, "data": map[string]any{"effect": "applied"}}
							if params["action"] == action {
								mutations.Add(1)
							} else {
								if params["action"] != action+"_evidence" || params["evidence"] != true {
									t.Errorf("incorrect guarded child: %v", params)
								}
								child = map[string]any{"tool": tool, "ok": false, "error": "unknown action"}
							}
							response.Data = map[string]any{"count": 2, "stopped": true, "results": []any{map[string]any{"tool": "scene", "ok": true}, child}}
						}
						if err := json.NewEncoder(w).Encode(response); err != nil {
							t.Error(err)
						}
					}))
					t.Cleanup(server.Close)
					installContractHome(t, contractServerPort(t, server), true)
					params := map[string]any{"action": action, "path": "res://test.tres", "props": map[string]any{"resource_name": "changed"}}
					if optIn == "evidence" {
						params[optIn] = true
					} else {
						params[optIn] = strings.Repeat("0", 64)
					}
					stdout, stderr, code := captureContractRun([]string{"batch", "--file", batchEvidenceFile(t, tool, params)})
					wantBatch := int32(0)
					if replaced {
						wantBatch = 1
					}
					if code != 1 || statusCalls.Load() != 1 || batchCalls.Load() != wantBatch || mutations.Load() != 0 {
						t.Fatalf("exit=%d status=%d batch=%d mutations=%d stdout=%s stderr=%s", code, statusCalls.Load(), batchCalls.Load(), mutations.Load(), stdout, stderr)
					}
					if replaced && !strings.Contains(stdout, "unknown action") {
						t.Fatalf("lost child failure: %s", stdout)
					}
				})
			}
		}
	}
}

func TestBatchEvidenceResultEnforcement(t *testing.T) {
	for _, tool := range []string{"scene", "resource", "screenshot", "game"} {
		for _, shape := range []string{"missing", "unavailable", "available", "failed", "omitted_result", "legacy"} {
			t.Run(tool+"/"+shape, func(t *testing.T) {
				childData := map[string]any{"observed": "retain-me"}
				if shape == "available" || shape == "failed" || shape == "unavailable" {
					childData["evidence"] = map[string]any{"available": shape != "unavailable"}
				}
				child := map[string]any{"tool": tool, "ok": shape != "failed" && shape != "legacy", "data": childData}
				if shape == "failed" || shape == "legacy" {
					child["error"] = "state_conflict"
				}
				results := []any{map[string]any{"tool": "scene", "ok": true}, child}
				if shape == "omitted_result" {
					results = results[:1]
				}
				body, err := json.Marshal(protocol.Response{OK: true, Data: map[string]any{"count": len(results), "results": results, "stopped": false}})
				if err != nil {
					t.Fatal(err)
				}
				server := startContractEditor(t, map[string]string{
					"status": `{"ok":true,"data":{"capabilities":{"linked_evidence":"supported"}}}`,
					"batch":  string(body),
				})
				t.Cleanup(server.Close)
				installContractHome(t, contractServerPort(t, server), true)
				params := map[string]any{"evidence": shape != "legacy", "action": "save"}
				if tool == "resource" {
					params["action"] = "set"
				} else if tool == "game" {
					params["action"] = "node_get"
				}
				stdout, stderr, code := captureContractRun([]string{"batch", "--file", batchEvidenceFile(t, tool, params)})
				wantCode := 1
				if shape == "available" || shape == "legacy" {
					wantCode = 0
				}
				if code != wantCode {
					t.Fatalf("exit=%d want=%d stdout=%s stderr=%s", code, wantCode, stdout, stderr)
				}
				var data struct {
					Results []struct {
						OK    bool
						Error string
						Data  map[string]any
					}
				}
				if err := json.Unmarshal([]byte(stdout), &data); err != nil {
					t.Fatal(err)
				}
				if shape != "omitted_result" {
					result := data.Results[1]
					if result.OK != (shape == "available") || result.Data["observed"] != "retain-me" {
						t.Fatalf("incorrect child result: %+v", result)
					}
					if (shape == "missing" || shape == "unavailable") && result.Error != "evidence_unavailable" {
						t.Fatalf("missing explicit evidence failure: %+v", result)
					}
				}
				if shape == "legacy" {
					if stderr != "" || len(data.Results) != 2 || data.Results[1].Error != "state_conflict" {
						t.Fatalf("legacy output changed: %s %s", stdout, stderr)
					}
				}
			})
		}
	}
}
