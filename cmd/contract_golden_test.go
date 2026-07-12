package cmd

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/NotNull92/hera-agent-godot/internal/protocol"
)

// updateGolden rewrites the golden files under testdata/contract from the
// current CLI behavior: go test ./cmd -run TestContract -update
var updateGolden = flag.Bool("update", false, "rewrite contract golden files")

// contractCase drives the public CLI contract end-to-end: argv in, stdout +
// stderr + exit code out. The editor is faked by an httptest /rpc server that
// the CLI discovers through a heartbeat file planted in a temp home dir, so
// the real discovery → dial → print path runs unmodified.
//
// Response values starting with "@" are read from
// testdata/contract/responses/<name>.json; anything else is served verbatim.
type contractCase struct {
	name             string
	args             []string
	responses        map[string]string // "tool" or "tool:action" → raw /rpc response body
	noEditor         bool              // plant no heartbeat: "no live editor" path
	wantExit         int
	wantStderrPrefix string
	golden           string // golden file name; "" asserts empty stdout
	normalize        func(string) string
}

var (
	contractPortRE = regexp.MustCompile(`"port":\d+`)
	contractTSRE   = regexp.MustCompile(`"ts":\d+`)
)

// normalizeInstances pins the two fields of `instances` output that legitimately
// vary per run (the mock server's random port and the heartbeat timestamp).
func normalizeInstances(s string) string {
	s = contractPortRE.ReplaceAllString(s, `"port":8770`)
	return contractTSRE.ReplaceAllString(s, `"ts":0`)
}

func contractCases() []contractCase {
	const (
		gameTree    = `{"ok":true,"data":{"count":1,"nodes":[{"name":"Main","path":".","type":"Node2D"}],"scene":"res://scenes/Main.tscn","truncated":false}}`
		gameNodeGet = `{"ok":true,"data":{"name":"Main","path":".","properties":{"position":"(0.0, 0.0)","visible":"true"},"type":"Node2D"}}`
		gameAssert  = `{"ok":true,"data":{"actual":"true","expected":"true","op":"eq","prop":"visible"}}`
		runPlaying  = `{"ok":true,"data":{"playing":true,"scene":"res://scenes/Main.tscn"}}`
		runStopped  = `{"ok":true,"data":{"playing":false,"scene":""}}`

		diagClean      = `{"ok":true,"data":{"error_count":0,"warning_count":0}}`
		gameOneProc    = `{"ok":true,"data":{"instances":[{"pid":42}]}}`
		gameNoProc     = `{"ok":true,"data":{"instances":[]}}`
		gameTreeSmall  = `{"ok":true,"data":{"count":3,"scene":"res://scenes/Main.tscn","truncated":false}}`
		gameUISmall    = `{"ok":true,"data":{"count":2,"truncated":false}}`
		gameShotClean  = `{"ok":true,"data":{"analysis":{"nonblank":true,"low_detail":false,"possible_clipping":false}}}`
		gameNotRunning = `{"ok":false,"error":"no game is running"}`
	)

	return []contractCase{
		// Stable commands, live-captured fixtures (Godot 4.7-stable).
		{name: "status", args: []string{"status"}, responses: map[string]string{"status": "@status"}, golden: "status"},
		{name: "status_json", args: []string{"--json", "status"}, responses: map[string]string{"status": "@status"}, golden: "status_json"},
		{name: "scene_tree", args: []string{"scene", "tree"}, responses: map[string]string{"scene": "@scene_tree"}, golden: "scene_tree"},
		{name: "scene_tree_ids", args: []string{"--ids", "scene", "tree"}, responses: map[string]string{"scene": "@scene_tree"}, golden: "scene_tree_ids"},
		{name: "scene_list", args: []string{"scene", "list"}, responses: map[string]string{"scene": "@scene_list"}, golden: "scene_list"},
		{name: "editor_state", args: []string{"editor", "state"}, responses: map[string]string{"editor": "@editor_state"}, golden: "editor_state"},
		{name: "node_find", args: []string{"node", "find", "Main"}, responses: map[string]string{"node": "@node_find"}, golden: "node_find"},
		{name: "node_get", args: []string{"node", "get", "."}, responses: map[string]string{"node": "@node_get"}, golden: "node_get"},
		{name: "signal_list", args: []string{"signal", "list", "."}, responses: map[string]string{"signal": "@signal_list"}, golden: "signal_list"},
		{name: "project_info", args: []string{"project", "info"}, responses: map[string]string{"project": "@project_info"}, golden: "project_info"},
		{name: "classdb_info", args: []string{"classdb", "info", "Node2D"}, responses: map[string]string{"classdb": "@classdb_info"}, golden: "classdb_info"},
		{name: "resource_uid", args: []string{"resource", "uid", "res://scenes/Main.tscn"}, responses: map[string]string{"resource": "@resource_uid"}, golden: "resource_uid"},
		{name: "output", args: []string{"output", "--type", "all", "--lines", "3"}, responses: map[string]string{"output": "@output"}, golden: "output"},
		{name: "diagnostics", args: []string{"diagnostics", "--lines", "5"}, responses: map[string]string{"diagnostics": "@diagnostics"}, golden: "diagnostics"},
		{name: "eval", args: []string{"eval", "1+1"}, responses: map[string]string{"eval": "@eval"}, golden: "eval"},
		{name: "batch", args: []string{"batch", "--file", filepath.Join("testdata", "contract", "batch_input.json")}, responses: map[string]string{"batch": "@batch"}, golden: "batch"},

		// Stable commands, synthesized fixtures (shapes from the addon source;
		// avoids starting a real play session to capture them).
		{name: "run_current", args: []string{"run", "--current"}, responses: map[string]string{"run": runPlaying}, golden: "run_current"},
		{name: "stop", args: []string{"stop"}, responses: map[string]string{"run": runStopped}, golden: "stop"},
		{name: "game_tree", args: []string{"game", "tree"}, responses: map[string]string{"game": gameTree}, golden: "game_tree"},
		{name: "game_node_get", args: []string{"game", "node", "get", "/root/Main"}, responses: map[string]string{"game": gameNodeGet}, golden: "game_node_get"},
		{name: "game_assert_pass", args: []string{"game", "assert", "/root/Main", "visible", "eq", "true"}, responses: map[string]string{"game": gameAssert}, golden: "game_assert_pass"},

		// Commands that answer without an /rpc round trip.
		{name: "version", args: []string{"version"}, golden: "version"},
		{name: "help", args: []string{"help"}, golden: "help"},
		{name: "instances", args: []string{"instances"}, golden: "instances", normalize: normalizeInstances},

		// Verdict command: stdout report + exit code mirroring ok.
		{
			name: "game_qa_diagnose_ok",
			args: []string{"game", "qa", "diagnose"},
			responses: map[string]string{
				"diagnostics":     diagClean,
				"game:instances":  gameOneProc,
				"game:tree":       gameTreeSmall,
				"game:ui_tree":    gameUISmall,
				"game:screenshot": gameShotClean,
			},
			golden: "game_qa_diagnose_ok",
		},
		{
			name: "game_qa_diagnose_issues",
			args: []string{"game", "qa", "diagnose"},
			responses: map[string]string{
				"diagnostics":     diagClean,
				"game:instances":  gameNoProc,
				"game:tree":       gameNotRunning,
				"game:ui_tree":    gameNotRunning,
				"game:screenshot": gameNotRunning,
			},
			wantExit: 1,
			golden:   "game_qa_diagnose_issues",
		},

		// Failure semantics: exit 1 = runtime failure, exit 2 = usage error;
		// diagnostics go to stderr as "<command>: <message>", stdout stays empty.
		{name: "game_assert_fail", args: []string{"game", "assert", "/root/Main", "visible", "eq", "true"}, responses: map[string]string{"game": `{"ok":false,"error":"assert failed: visible eq true (actual: false)"}`}, wantExit: 1, wantStderrPrefix: "game: assert failed"},
		{name: "tool_error", args: []string{"node", "get", "/nonexistent"}, responses: map[string]string{"node": `{"ok":false,"error":"node not found: /nonexistent"}`}, wantExit: 1, wantStderrPrefix: "node: "},
		{name: "no_editor", args: []string{"status"}, noEditor: true, wantExit: 1, wantStderrPrefix: "status: "},
		{name: "unknown_command", args: []string{"bogus"}, noEditor: true, wantExit: 2, wantStderrPrefix: "unknown command"},
		{name: "invalid_instance", args: []string{"--instance", "abc", "status"}, noEditor: true, wantExit: 2, wantStderrPrefix: "--instance: invalid pid"},
		{name: "invalid_timeout", args: []string{"--timeout", "abc", "status"}, noEditor: true, wantExit: 2, wantStderrPrefix: "--timeout: invalid milliseconds"},
		{name: "missing_timeout_value", args: []string{"--timeout"}, noEditor: true, wantExit: 2, wantStderrPrefix: "--timeout requires"},
		{name: "timeout_eq_form", args: []string{"--timeout=1500", "status"}, responses: map[string]string{"status": "@status"}, golden: "status"},
	}
}

func TestContract_goldenOutputs(t *testing.T) {
	for _, tc := range contractCases() {
		t.Run(tc.name, func(t *testing.T) {
			responses := make(map[string]string, len(tc.responses))
			for key, value := range tc.responses {
				if name, ok := strings.CutPrefix(value, "@"); ok {
					value = contractFixture(t, name)
				}
				responses[key] = value
			}
			srv := startContractEditor(t, responses)
			t.Cleanup(srv.Close)
			installContractHome(t, contractServerPort(t, srv), !tc.noEditor)

			stdout, stderr, code := captureContractRun(tc.args)

			if code != tc.wantExit {
				t.Errorf("exit = %d, want %d (stderr: %q)", code, tc.wantExit, stderr)
			}
			if tc.normalize != nil {
				stdout = tc.normalize(stdout)
			}
			if tc.golden != "" {
				assertContractGolden(t, tc.golden, stdout)
			} else if stdout != "" {
				t.Errorf("stdout = %q, want empty", stdout)
			}
			if tc.wantStderrPrefix != "" && !strings.HasPrefix(stderr, tc.wantStderrPrefix) {
				t.Errorf("stderr = %q, want prefix %q", stderr, tc.wantStderrPrefix)
			}
		})
	}
}

func TestContract_timeoutBoundsRequests(t *testing.T) {
	status := contractFixture(t, "status")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(250 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, status)
	}))
	t.Cleanup(srv.Close)
	installContractHome(t, contractServerPort(t, srv), true)

	stdout, stderr, code := captureContractRun([]string{"--timeout", "50", "status"})
	if code != 1 {
		t.Errorf("short timeout: exit = %d, want 1 (stdout: %q)", code, stdout)
	}
	if !strings.HasPrefix(stderr, "status: ") {
		t.Errorf("short timeout: stderr = %q, want prefix %q", stderr, "status: ")
	}

	stdout, stderr, code = captureContractRun([]string{"--timeout", "5000", "status"})
	if code != 0 {
		t.Errorf("ample timeout: exit = %d, want 0 (stderr: %q)", code, stderr)
	}
	if !strings.HasSuffix(stdout, "\n") || stdout == "\n" {
		t.Errorf("ample timeout: stdout = %q, want one JSON line", stdout)
	}
}

// startContractEditor serves canned /rpc responses keyed by "tool:action" with
// a fallback to the bare tool name.
func startContractEditor(t *testing.T, responses map[string]string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request protocol.Request
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode request: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		key := request.Tool
		if action, ok := request.Params["action"].(string); ok {
			key = request.Tool + ":" + action
		}
		body, ok := responses[key]
		if !ok {
			body, ok = responses[request.Tool]
		}
		if !ok {
			t.Errorf("unexpected request %q", key)
			body = `{"ok":false,"error":"contract test: unexpected request"}`
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, body)
	}))
}

// installContractHome points discovery at a temp home dir; with heartbeat=true
// it plants one fresh instance file advertising the given port.
func installContractHome(t *testing.T, port int, heartbeat bool) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("HERA_AGENT_GODOT_TOKEN", "") // keep opt-in auth out of golden runs
	if !heartbeat {
		return
	}
	dir := filepath.Join(home, ".hera-agent-godot", "instances")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir instances: %v", err)
	}
	entry := fmt.Sprintf(
		`{"pid":49928,"port":%d,"project_path":"C:/Users/PC/Desktop/Cowork/hera-agent-godot/","godot_version":"4.7-stable (official)","scene":"res://scenes/Main.tscn","ts":%d}`,
		port, time.Now().Unix())
	if err := os.WriteFile(filepath.Join(dir, "49928.json"), []byte(entry), 0o644); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}
}

func contractServerPort(t *testing.T, srv *httptest.Server) int {
	t.Helper()
	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("parse server url %q: %v", srv.URL, err)
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil {
		t.Fatalf("parse server port %q: %v", u.Port(), err)
	}
	return port
}

// captureContractRun runs Execute with stdout/stderr redirected to pipes.
func captureContractRun(args []string) (stdout, stderr string, code int) {
	oldOut, oldErr := os.Stdout, os.Stderr
	outR, outW, _ := os.Pipe()
	errR, errW, _ := os.Pipe()
	os.Stdout, os.Stderr = outW, errW
	outCh := make(chan string, 1)
	errCh := make(chan string, 1)
	go drainContractPipe(outR, outCh)
	go drainContractPipe(errR, errCh)

	code = Execute(args)

	os.Stdout, os.Stderr = oldOut, oldErr
	_ = outW.Close()
	_ = errW.Close()
	return <-outCh, <-errCh, code
}

func drainContractPipe(r *os.File, ch chan<- string) {
	b, _ := io.ReadAll(r)
	_ = r.Close()
	ch <- string(b)
}

func contractFixture(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "contract", "responses", name+".json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	return string(b)
}

func assertContractGolden(t *testing.T, name string, got string) {
	t.Helper()
	path := filepath.Join("testdata", "contract", name+".golden")
	if *updateGolden {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("update golden: %v", err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden (create with: go test ./cmd -run TestContract -update): %v", err)
	}
	if w := strings.ReplaceAll(string(want), "\r\n", "\n"); w != got {
		t.Errorf("stdout does not match %s\n got: %q\nwant: %q", path, got, w)
	}
}
