package cmd

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestReadGameQAScenario_acceptsRequirementCoverage(t *testing.T) {
	// Given
	file := writeGameQAScenario(t, `{
		"requirements": ["launch-flow", "restart-flow"],
		"steps": [
			{"tool": "game.click", "text": "Launch", "covers": ["launch-flow"]},
			{"tool": "game.click", "text": "Restart", "covers": ["restart-flow"]}
		]
	}`)

	// When
	got, err := readGameQAScenario(file)

	// Then
	if err != nil {
		t.Fatalf("readGameQAScenario error: %v", err)
	}
	if fmt.Sprint(got.Requirements) != "[launch-flow restart-flow]" {
		t.Fatalf("requirements = %v, want [launch-flow restart-flow]", got.Requirements)
	}
	if fmt.Sprint(got.Steps[0].Covers) != "[launch-flow]" {
		t.Fatalf("covers = %v, want [launch-flow]", got.Steps[0].Covers)
	}
}

func TestReadGameQAScenario_acceptsLegacyStepArray(t *testing.T) {
	// Given
	file := writeGameQAScenario(t, `[
		{"tool": "diagnostics", "max_errors": 0}
	]`)

	// When
	got, err := readGameQAScenario(file)

	// Then
	if err != nil {
		t.Fatalf("readGameQAScenario error: %v", err)
	}
	if len(got.Requirements) != 0 {
		t.Fatalf("requirements = %v, want none", got.Requirements)
	}
	if len(got.Steps) != 1 || got.Steps[0].Tool != "diagnostics" {
		t.Fatalf("steps = %v, want one diagnostics step", got.Steps)
	}
}

func TestReadGameQAScenario_rejectsUncoveredRequirement(t *testing.T) {
	// Given
	file := writeGameQAScenario(t, `{
		"requirements": ["launch-flow", "restart-flow"],
		"steps": [
			{"tool": "game.click", "text": "Launch", "covers": ["launch-flow"]}
		]
	}`)

	// When
	_, err := readGameQAScenario(file)

	// Then
	if err == nil {
		t.Fatal("expected uncovered requirement error")
	}
	if !strings.Contains(err.Error(), "restart-flow") {
		t.Fatalf("error = %q, want missing requirement name", err.Error())
	}
}

func TestGameQASummary_countsOnlySuccessfulCoveredSteps(t *testing.T) {
	// Given
	scenario := gameQAScenario{
		Requirements: []string{"launch-flow"},
		Steps: []gameQAStep{
			{Tool: "game.click", Covers: []string{"launch-flow"}},
		},
	}
	results := []gameQAResult{
		{Step: 1, Tool: "game.click", OK: false, Covers: []string{"launch-flow"}},
	}

	// When
	data, ok := gameQASummary(scenario, results, false)

	// Then
	if ok {
		t.Fatal("summary ok = true, want false")
	}
	if fmt.Sprint(data["requirements_missing"]) != "[launch-flow]" {
		t.Fatalf("requirements_missing = %v, want [launch-flow]", data["requirements_missing"])
	}
}

func writeGameQAScenario(t *testing.T, content string) string {
	t.Helper()
	path := t.TempDir() + "/scenario.json"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write scenario: %v", err)
	}
	return path
}
