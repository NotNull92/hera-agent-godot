package cmd

import (
	"encoding/json"
	"github.com/NotNull92/hera-agent-godot/internal/client"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGameAssertWithin(t *testing.T) {
	for _, args := range [][]string{{".", "visible", "eq", "true", "--within-ms=100"}, {".", "visible", "exists", "--within-ms", "100"}} {
		p, err := parseGameAssertArgs(args)
		if err != nil || p["within_ms"] != 100 {
			t.Fatalf("%v %v", p, err)
		}
	}
	for _, n := range []string{"0", "-1", "30001", "1.5", ""} {
		if _, err := parseGameAssertArgs([]string{".", "visible", "exists", "--within-ms=" + n}); err == nil {
			t.Fatal(n)
		}
	}
	for _, raw := range []string{`[{"tool":"game.assert","path":".","prop":"visible","op":"exists","within_ms":0}]`, `[{"tool":"wait","within_ms":2}]`} {
		scenario, err := parseGameQAScenario([]byte(raw))
		if err == nil && validateGameQAScenario(scenario) == nil {
			t.Fatal(raw)
		}
	}
}
func TestAssertionWaitExtendsOnlyItsRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(20 * time.Millisecond)
		w.Write([]byte(`{"ok":false,"error":"assert failed","data":{"actual":"false","waited_ms":100,"timed_out":true}}`))
	}))
	defer server.Close()
	c := client.NewWithTimeout(server.URL, time.Millisecond)
	var step gameQAStep
	if err := json.Unmarshal([]byte(`{"tool":"game.assert","path":".","prop":"visible","op":"eq","value":true,"within_ms":100}`), &step); err != nil {
		t.Fatal(err)
	}
	result := executeGameQAStep(c, 1, step)
	if result.OK || result.Error != "assert failed" || result.Data.(map[string]any)["actual"] != "false" {
		t.Fatalf("%+v", result)
	}
	if c.HTTP.Timeout != time.Millisecond {
		t.Fatal("changed shared client timeout")
	}
}
