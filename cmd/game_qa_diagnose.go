package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/NotNull92/hera-agent-godot/internal/client"
	"github.com/NotNull92/hera-agent-godot/internal/protocol"
)

const (
	defaultGameQADiagnoseLines = 40
	noGameQAWarningLimit       = -1
)

type gameQADiagnoseOptions struct {
	lines          int
	maxErrors      int
	maxWarnings    int
	screenshotPath string
}

func runGameQADiagnose(args []string) int {
	options, err := parseGameQADiagnoseArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "game qa diagnose: %v\n", err)
		return 2
	}
	c, err := dialEditor()
	if err != nil {
		fmt.Fprintf(os.Stderr, "game qa diagnose: %v\n", err)
		return 1
	}
	data, ok := executeGameQADiagnosis(c, options)
	printData(&protocol.Response{OK: true, Data: data})
	if !ok {
		return 1
	}
	return 0
}

func parseGameQADiagnoseArgs(args []string) (gameQADiagnoseOptions, error) {
	options := gameQADiagnoseOptions{
		lines:       defaultGameQADiagnoseLines,
		maxErrors:   0,
		maxWarnings: noGameQAWarningLimit,
	}
	for index := 0; index < len(args); index++ {
		switch args[index] {
		case "--lines":
			value, next, err := gameQADiagnosePositiveInt(args, index, "--lines")
			if err != nil {
				return gameQADiagnoseOptions{}, err
			}
			options.lines = value
			index = next
		case "--max-errors":
			value, next, err := gameQADiagnoseNonNegativeInt(args, index, "--max-errors")
			if err != nil {
				return gameQADiagnoseOptions{}, err
			}
			options.maxErrors = value
			index = next
		case "--max-warnings":
			value, next, err := gameQADiagnoseNonNegativeInt(args, index, "--max-warnings")
			if err != nil {
				return gameQADiagnoseOptions{}, err
			}
			options.maxWarnings = value
			index = next
		case "--path":
			if index+1 >= len(args) || args[index+1] == "" {
				return gameQADiagnoseOptions{}, fmt.Errorf("--path requires a non-empty value")
			}
			index++
			options.screenshotPath = args[index]
		default:
			return gameQADiagnoseOptions{}, fmt.Errorf("unknown flag %q", args[index])
		}
	}
	return options, nil
}

func gameQADiagnosePositiveInt(args []string, index int, flag string) (int, int, error) {
	value, next, err := gameQADiagnoseNonNegativeInt(args, index, flag)
	if err != nil {
		return 0, index, err
	}
	if value == 0 {
		return 0, index, fmt.Errorf("%s must be greater than zero", flag)
	}
	return value, next, nil
}

func gameQADiagnoseNonNegativeInt(args []string, index int, flag string) (int, int, error) {
	if index+1 >= len(args) {
		return 0, index, fmt.Errorf("%s requires a value", flag)
	}
	value, err := strconv.Atoi(args[index+1])
	if err != nil || value < 0 {
		return 0, index, fmt.Errorf("invalid %s %q (want a non-negative integer)", flag, args[index+1])
	}
	return value, index + 1, nil
}

func executeGameQADiagnosis(c *client.Client, options gameQADiagnoseOptions) (map[string]any, bool) {
	checks := make([]map[string]any, 0, 5)
	issues := make([]string, 0)

	diagnostics, err := gameQADiagnoseData(c, "diagnostics", map[string]any{"lines": options.lines})
	if err != nil {
		checks = append(checks, gameQADiagnoseFailure("editor_diagnostics", err))
		issues = append(issues, fmt.Sprintf("editor diagnostics unavailable: %v", err))
	} else {
		check, findings := evaluateGameQADiagnostics(diagnostics, options)
		checks = append(checks, check)
		issues = append(issues, findings...)
	}

	instances, err := gameQADiagnoseData(c, "game", map[string]any{"action": "instances"})
	if err != nil {
		checks = append(checks, gameQADiagnoseFailure("runtime_instances", err))
		issues = append(issues, fmt.Sprintf("runtime instances unavailable: %v", err))
	} else {
		check, findings := evaluateGameQAInstances(instances)
		checks = append(checks, check)
		issues = append(issues, findings...)
	}

	tree, err := gameQADiagnoseData(c, "game", map[string]any{"action": "tree"})
	if err != nil {
		checks = append(checks, gameQADiagnoseFailure("runtime_tree", err))
		issues = append(issues, fmt.Sprintf("runtime tree unavailable: %v", err))
	} else {
		check, findings := evaluateGameQATree(tree)
		checks = append(checks, check)
		issues = append(issues, findings...)
	}

	ui, err := gameQADiagnoseData(c, "game", map[string]any{"action": "ui_tree"})
	if err != nil {
		checks = append(checks, gameQADiagnoseFailure("runtime_ui", err))
		issues = append(issues, fmt.Sprintf("runtime UI tree unavailable: %v", err))
	} else {
		check, findings := evaluateGameQAUI(ui)
		checks = append(checks, check)
		issues = append(issues, findings...)
	}

	screenshotParams := map[string]any{"action": "screenshot", "analyze": true}
	if options.screenshotPath != "" {
		screenshotParams["path"] = options.screenshotPath
	}
	screenshot, err := gameQADiagnoseData(c, "game", screenshotParams)
	if err != nil {
		checks = append(checks, gameQADiagnoseFailure("runtime_screenshot", err))
		issues = append(issues, fmt.Sprintf("runtime screenshot unavailable: %v", err))
	} else {
		check, findings := evaluateGameQAScreenshot(screenshot)
		checks = append(checks, check)
		issues = append(issues, findings...)
	}

	ok := len(issues) == 0
	return map[string]any{"ok": ok, "checks": checks, "issues": issues}, ok
}

func gameQADiagnoseData(c *client.Client, tool string, params map[string]any) (map[string]any, error) {
	resp, err := c.Post(tool, params)
	if err != nil {
		return nil, err
	}
	if !resp.OK {
		return nil, fmt.Errorf("%s", resp.Error)
	}
	data, ok := resp.Data.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected response data")
	}
	return data, nil
}
