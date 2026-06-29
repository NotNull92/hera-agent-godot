package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/NotNull92/hera-agent-godot/internal/client"
	"github.com/NotNull92/hera-agent-godot/internal/protocol"
)

func TestSmokeRunnerRun_waitsForGameTreeBeforeInspectingRuntime(t *testing.T) {
	// Given
	const scene = "res://scenes/Main.tscn"
	gameTreeCalls := 0
	screenshotAnalyzed := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req protocol.Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		resp := protocol.Response{OK: true, Data: map[string]any{}}
		switch req.Tool {
		case "run":
			action, _ := req.Params["action"].(string)
			resp.Data = map[string]any{"playing": action != "stop", "scene": scene}
			if action == "state" {
				resp.Data = map[string]any{"playing": gameTreeCalls < 3, "scene": scene}
			}
		case "game":
			action, _ := req.Params["action"].(string)
			switch action {
			case "instances":
				resp.Data = map[string]any{"instances": []any{}}
			case "tree":
				gameTreeCalls++
				if gameTreeCalls == 1 {
					resp = protocol.Response{OK: false, Error: "game inspector not ready"}
					break
				}
				resp.Data = map[string]any{"scene": scene, "nodes": []any{}}
			case "screenshot":
				screenshotAnalyzed = req.Params["analyze"] == true
				resp.Data = map[string]any{
					"path":   "user://hera_game_screenshots/smoke.png",
					"width":  800,
					"height": 600,
					"analysis": map[string]any{
						"nonblank":          true,
						"possible_clipping": false,
					},
				}
			}
		}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	t.Cleanup(srv.Close)
	runner := smokeRunner{client: client.New(srv.URL), steps: make([]map[string]any, 0, 7)}

	// When
	steps, err := runner.run(smokeOptions{runGame: true})

	// Then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gameTreeCalls < 3 {
		t.Fatalf("gameTreeCalls = %d, want poll plus final tree call", gameTreeCalls)
	}
	if !screenshotAnalyzed {
		t.Fatalf("runtime screenshot analyze request was not sent")
	}
	if len(steps) != 8 {
		t.Fatalf("len(steps) = %d, want 8", len(steps))
	}
}
