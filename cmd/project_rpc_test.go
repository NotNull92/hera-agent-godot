package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/NotNull92/hera-agent-godot/internal/protocol"
)

func TestMainSceneSetterUsesEditorRPC(t *testing.T) {
	for _, mode := range []string{"", "--json"} {
		t.Run(mode, func(t *testing.T) {
			project := t.TempDir()
			original := "[application]\nconfig/name=\"Unterminated\""
			if err := os.WriteFile(filepath.Join(project, "project.godot"), []byte(original), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(project, "New.tscn"), []byte("[gd_scene format=3]\n[node name=\"New\" type=\"Node\"]"), 0600); err != nil {
				t.Fatal(err)
			}
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var request protocol.Request
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Error(err)
					return
				}
				requests++
				if request.Tool != "project" || request.Params["action"] != "set_main_scene" || request.Params["path"] != "res://New.tscn" {
					t.Errorf("request=%+v", request)
				}
				if err := json.NewEncoder(w).Encode(protocol.Response{OK: true, Data: map[string]any{"main_scene": "res://New.tscn", "project_path": project}}); err != nil {
					t.Error(err)
				}
			}))
			t.Cleanup(server.Close)
			installMainSceneContractHome(t, contractServerPort(t, server), project)
			args := []string{"project", "set-main-scene", "res://New.tscn"}
			if mode != "" {
				args = append([]string{mode}, args...)
			}
			stdout, stderr, code := captureContractRun(args)
			var data map[string]any
			if err := json.Unmarshal([]byte(stdout), &data); err != nil {
				t.Fatalf("output=%q error=%v", stdout, err)
			}
			if code != 0 || requests != 1 || data["main_scene"] != "res://New.tscn" || data["project_path"] != project {
				t.Fatalf("code=%d requests=%d data=%v stderr=%s", code, requests, data, stderr)
			}
			disk, err := os.ReadFile(filepath.Join(project, "project.godot"))
			if err != nil || string(disk) != original {
				t.Fatalf("CLI bypassed editor to write disk: %q err=%v", disk, err)
			}
		})
	}
}

func TestBareRunUsesEditorMainScene(t *testing.T) {
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "project.godot"), []byte("[application]\nrun/main_scene=\"res://Old.tscn\"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request protocol.Request
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		if request.Tool != "run" || request.Params["action"] != "play_main" {
			t.Errorf("request=%+v, want native play_main", request)
		}
		if err := json.NewEncoder(w).Encode(protocol.Response{OK: true, Data: map[string]any{"playing": true, "scene": "res://New.tscn"}}); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	installMainSceneContractHome(t, contractServerPort(t, server), project)
	_, stderr, code := captureContractRun([]string{"run"})
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr)
	}
}

func installMainSceneContractHome(t *testing.T, port int, project string) {
	t.Helper()
	installContractHome(t, port, true)
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(map[string]any{"pid": 49928, "port": port, "project_path": project, "ts": time.Now().Unix()})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".hera-agent-godot", "instances", fmt.Sprintf("%d.json", 49928)), raw, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestMainSceneSetterPropagatesEditorRejection(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if err := json.NewEncoder(w).Encode(protocol.Response{OK: false, Error: "scene path must stay inside res://"}); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	installMainSceneContractHome(t, contractServerPort(t, server), t.TempDir())
	stdout, stderr, code := captureContractRun([]string{"project", "set-main-scene", "res://../outside.tscn"})
	if code != 1 || requests != 1 || stdout != "" || stderr == "" {
		t.Fatalf("code=%d requests=%d stdout=%q stderr=%q", code, requests, stdout, stderr)
	}
}

func TestMainSceneSetterKeepsMultipleEditorGuard(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)
	installMainSceneContractHome(t, contractServerPort(t, server), t.TempDir())
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(map[string]any{"pid": 49929, "port": contractServerPort(t, server), "project_path": t.TempDir(), "ts": time.Now().Unix()})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".hera-agent-godot", "instances", "49929.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	_, stderr, code := captureContractRun([]string{"project", "set-main-scene", "res://New.tscn"})
	if code != 1 || requests != 0 || stderr == "" {
		t.Fatalf("code=%d requests=%d stderr=%q", code, requests, stderr)
	}
}
