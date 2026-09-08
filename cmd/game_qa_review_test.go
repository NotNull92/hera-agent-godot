package cmd

import (
	"math"
	"testing"

	"github.com/NotNull92/hera-agent-godot/internal/client"
	"github.com/NotNull92/hera-agent-godot/internal/protocol"
)

func TestGameQADiagnosticsRejectsUnreliableCounts(t *testing.T) {
	for name, data := range map[string]map[string]any{
		"unavailable":      {"available": false, "error_count": 0, "warning_count": 0},
		"missing":          {},
		"wrong type":       {"error_count": "0", "warning_count": 0},
		"fraction":         {"error_count": 0.5, "warning_count": 0},
		"negative":         {"error_count": -1, "warning_count": 0},
		"overflow":         {"error_count": math.MaxFloat64, "warning_count": 0},
		"bad availability": {"available": "false", "error_count": 0, "warning_count": 0},
	} {
		t.Run(name, func(t *testing.T) {
			// Given
			server := newGameQADiagnoseServer(t, map[string]protocol.Response{"diagnostics": {OK: true, Data: data}})
			t.Cleanup(server.Close)
			scenario := gameQAScenario{Requirements: []string{"clean"}, Steps: []gameQAStep{{Tool: "diagnostics", Covers: []string{"clean"}}}}
			// When
			results, stepsOK := executeGameQASteps(client.New(server.URL), scenario.Steps, false)
			_, ok := gameQASummary(scenario, results, stepsOK)
			check, issues := evaluateGameQADiagnostics(data, gameQADiagnoseOptions{})
			// Then
			if ok || check["ok"] == true || len(issues) == 0 {
				t.Fatalf("scenario=%v diagnosis=%v issues=%v", ok, check, issues)
			}
		})
	}
}

func TestGameQADiagnosticsWarningLimitDistinguishesOmittedAndZero(t *testing.T) {
	for _, tc := range []struct {
		raw     string
		wantErr bool
	}{
		{`[{"tool":"diagnostics"}]`, false},
		{`[{"tool":"diagnostics","max_warnings":0}]`, true},
		{`[{"tool":"diagnostics","max_warnings":1}]`, false},
	} {
		// Given
		scenario, err := parseGameQAScenario([]byte(tc.raw))
		if err != nil {
			t.Fatal(err)
		}
		// When
		err = validateDiagnosticsThresholds(&protocol.Response{Data: map[string]any{"error_count": 0, "warning_count": 1}}, scenario.Steps[0])
		// Then
		if (err != nil) != tc.wantErr {
			t.Fatalf("%s: err=%v", tc.raw, err)
		}
	}
}

func TestGameQAScenarioPreflightsInvalidLaterSteps(t *testing.T) {
	for _, invalid := range []string{
		`{"tool":"typo"}`, `{"tool":"wait","duration_ms":-1}`,
		`{"tool":"wait","duration_ms":9223372036854775807}`,
		`{"tool":"game.node.get"}`, `{"tool":"game.node.set","path":"/root/Main"}`,
		`{"tool":"game.node.call","path":"/root/Main"}`,
		`{"tool":"game.assert","path":"/root/Main","prop":"score","op":"typo"}`,
		`{"tool":"diagnostics","max_errors":-1}`, `{"tool":"diagnostics","max_warnings":-1}`,
		`{"tool":"run","action":"typo"}`, `{"tool":"run","action":"play_custom"}`,
	} {
		// Given
		file := writeGameQAScenario(t, `[{"tool":"run"},`+invalid+`]`)
		// When
		_, err := readGameQAScenario(file)
		// Then: argument failure occurs before editor discovery or any request.
		if err == nil {
			t.Fatalf("%s: expected preflight rejection", invalid)
		}
	}
}

func TestGameQAScenarioRequiresValuesButAcceptsExplicitNull(t *testing.T) {
	for _, tc := range []struct {
		raw     string
		wantErr bool
	}{
		{`[{"tool":"game.node.set","path":"/root/Main","prop":"target"}]`, true},
		{`[{"tool":"game.node.set","path":"/root/Main","prop":"target","value":null}]`, false},
		{`[{"tool":"game.assert","path":"/root/Main","prop":"target","op":"eq"}]`, true},
		{`[{"tool":"game.assert","path":"/root/Main","prop":"target","op":"eq","value":null}]`, false},
		{`[{"tool":"game.assert","path":"/root/Main","prop":"target","op":"exists"}]`, false},
	} {
		file := writeGameQAScenario(t, tc.raw)
		_, err := readGameQAScenario(file)
		if (err != nil) != tc.wantErr {
			t.Fatalf("%s: error=%v", tc.raw, err)
		}
	}
}
