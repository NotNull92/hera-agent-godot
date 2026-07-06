package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type gameQAScenario struct {
	Requirements []string     `json:"requirements"`
	Steps        []gameQAStep `json:"steps"`
}

type gameQAStep struct {
	Tool        string         `json:"tool"`
	Path        string         `json:"path"`
	Prop        string         `json:"prop"`
	Props       []string       `json:"props"`
	Op          string         `json:"op"`
	Value       any            `json:"value"`
	X           int            `json:"x"`
	Y           int            `json:"y"`
	Text        string         `json:"text"`
	Action      string         `json:"action"`
	Scene       string         `json:"scene"`
	Current     bool           `json:"current"`
	Wait        bool           `json:"wait"`
	Method      string         `json:"method"`
	Args        []any          `json:"args"`
	Analyze     bool           `json:"analyze"`
	Lines       int            `json:"lines"`
	MaxErrors   int            `json:"max_errors"`
	MaxWarnings int            `json:"max_warnings"`
	DurationMS  int            `json:"duration_ms"`
	Covers      []string       `json:"covers"`
	Params      map[string]any `json:"params"`
}

type gameQAResult struct {
	Step   int      `json:"step"`
	Tool   string   `json:"tool"`
	OK     bool     `json:"ok"`
	Covers []string `json:"covers,omitempty"`
	Error  string   `json:"error,omitempty"`
}

func readGameQAScenario(file string) (gameQAScenario, error) {
	raw, err := os.ReadFile(file)
	if err != nil {
		return gameQAScenario{}, err
	}
	scenario, err := parseGameQAScenario(raw)
	if err != nil {
		return gameQAScenario{}, err
	}
	return scenario, validateGameQAScenario(scenario)
}

func parseGameQAScenario(raw []byte) (gameQAScenario, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return gameQAScenario{}, fmt.Errorf("invalid scenario JSON: empty file")
	}
	if trimmed[0] == '[' {
		var steps []gameQAStep
		if err := json.Unmarshal(trimmed, &steps); err != nil {
			return gameQAScenario{}, fmt.Errorf("invalid scenario JSON: %w", err)
		}
		return gameQAScenario{Steps: steps}, nil
	}
	var scenario gameQAScenario
	if err := json.Unmarshal(trimmed, &scenario); err != nil {
		return gameQAScenario{}, fmt.Errorf("invalid scenario JSON: %w", err)
	}
	return scenario, nil
}

func validateGameQAScenario(scenario gameQAScenario) error {
	if len(scenario.Steps) == 0 {
		return fmt.Errorf("scenario must contain at least one step")
	}
	plannedCoverage := map[string]bool{}
	for index, step := range scenario.Steps {
		if step.Tool == "" {
			return fmt.Errorf("step %d: tool is required", index+1)
		}
		for _, requirement := range step.Covers {
			plannedCoverage[requirement] = true
		}
	}
	for _, requirement := range scenario.Requirements {
		if strings.TrimSpace(requirement) == "" {
			return fmt.Errorf("requirement names must not be empty")
		}
		if !plannedCoverage[requirement] {
			return fmt.Errorf("requirement %q is not covered by any step", requirement)
		}
	}
	return nil
}

func gameQASummary(scenario gameQAScenario, results []gameQAResult, stepsOK bool) (map[string]any, bool) {
	missing := missingRequirements(scenario.Requirements, successfulCoverage(results))
	ok := stepsOK && len(missing) == 0
	data := map[string]any{
		"ok":      ok,
		"steps":   len(scenario.Steps),
		"results": results,
	}
	if len(scenario.Requirements) > 0 {
		data["requirements"] = len(scenario.Requirements)
		data["requirements_covered"] = coveredRequirements(scenario.Requirements, missing)
		data["requirements_missing"] = missing
	}
	return data, ok
}

func successfulCoverage(results []gameQAResult) map[string]bool {
	covered := map[string]bool{}
	for _, result := range results {
		if !result.OK {
			continue
		}
		for _, requirement := range result.Covers {
			covered[requirement] = true
		}
	}
	return covered
}

func missingRequirements(requirements []string, covered map[string]bool) []string {
	missing := make([]string, 0)
	for _, requirement := range requirements {
		if !covered[requirement] {
			missing = append(missing, requirement)
		}
	}
	return missing
}

func coveredRequirements(requirements []string, missing []string) []string {
	missingSet := map[string]bool{}
	for _, requirement := range missing {
		missingSet[requirement] = true
	}
	covered := make([]string, 0, len(requirements)-len(missing))
	for _, requirement := range requirements {
		if !missingSet[requirement] {
			covered = append(covered, requirement)
		}
	}
	return covered
}
