package cmd

import (
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"github.com/NotNull92/hera-agent-godot/internal/client"
	"github.com/NotNull92/hera-agent-godot/internal/protocol"
)

func runGameQA(args []string, targetPID int) int {
	file, keepGoing, err := parseGameQAFlags(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "game qa: %v\n", err)
		return 2
	}
	scenario, err := readGameQAScenario(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "game qa: %v\n", err)
		return 2
	}
	for index := range scenario.Steps {
		scenario.Steps[index].targetPID = targetPID
	}
	c, err := dialMutationEditor()
	if err != nil {
		fmt.Fprintf(os.Stderr, "game qa: %v\n", err)
		return 1
	}
	results, stepsOK := executeGameQASteps(c, scenario.Steps, keepGoing)
	data, ok := gameQASummary(scenario, results, stepsOK)
	resp := &protocol.Response{OK: true, Data: data}
	printData(resp)
	if !ok {
		return 1
	}
	return 0
}

func parseGameQAFlags(args []string) (file string, keepGoing bool, err error) {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--file":
			if i+1 >= len(args) {
				return "", false, fmt.Errorf("--file requires a path")
			}
			i++
			file = args[i]
		case "--continue":
			keepGoing = true
		default:
			return "", false, fmt.Errorf("unknown flag %q", args[i])
		}
	}
	if file == "" {
		return "", false, fmt.Errorf("game qa requires --file")
	}
	return file, keepGoing, nil
}

func executeGameQASteps(c *client.Client, steps []gameQAStep, keepGoing bool) ([]gameQAResult, bool) {
	results := make([]gameQAResult, 0, len(steps))
	ok := true
	for index, step := range steps {
		result := executeGameQAStep(c, index+1, step)
		if !result.OK {
			ok = false
		}
		results = append(results, result)
		if !result.OK && !keepGoing {
			break
		}
	}
	return results, ok
}

func executeGameQAStep(c *client.Client, index int, step gameQAStep) gameQAResult {
	result := gameQAResult{Step: index, Tool: step.Tool, Covers: step.Covers}
	resp, err := postGameQAStep(c, step)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	if !resp.OK {
		result.Error = resp.Error
		return result
	}
	if step.Tool == "game.ui.audit" {
		passed, auditErr := gameUIAuditPassed(resp.Data)
		if auditErr != nil {
			result.Error = auditErr.Error()
			return result
		}
		if !passed {
			result.Error = gameUIAuditFailure(resp.Data)
			result.Data = resp.Data
			return result
		}
	}
	result.OK = true
	return result
}

func postGameQAStep(c *client.Client, step gameQAStep) (*protocol.Response, error) {
	switch step.Tool {
	case "wait":
		if step.DurationMS < 0 || int64(step.DurationMS) > maxDurationMilliseconds {
			return nil, fmt.Errorf("wait duration_ms must be between 0 and %d", maxDurationMilliseconds)
		}
		time.Sleep(time.Duration(step.DurationMS) * time.Millisecond)
		return &protocol.Response{OK: true, Data: map[string]any{"waited_ms": step.DurationMS}}, nil
	case "run", "stop":
		params := runParamsFromQAStep(step)
		if step.Tool == "stop" {
			params = map[string]any{"action": "stop"}
		}
		resp, err := c.Post("run", params)
		if err != nil || !step.Wait || !resp.OK || params["action"] == "state" {
			return resp, err
		}
		playing := params["action"] != "stop"
		_, waitErr := pollPlaying(c, playing, waitTimeout)
		if waitErr != nil {
			return resp, waitErr
		}
		if !playing {
			return resp, pollGameInstancesStopped(c, waitTimeout)
		}
		_, waitErr = pollGameReady(c, sceneFromResponse(resp), waitTimeout)
		return resp, waitErr
	case "game.node.get":
		return c.Post("game", targetGameParams(gameNodeGetParamsFromQAStep(step), step.targetPID))
	case "game.node.set":
		return c.Post("game", targetGameParams(map[string]any{"action": "set", "path": normalizeGameNodePath(step.Path), "prop": step.Prop, "value": step.Value}, step.targetPID))
	case "game.node.call":
		return c.Post("game", targetGameParams(map[string]any{"action": "call", "path": normalizeGameNodePath(step.Path), "method": step.Method, "args": step.Args}, step.targetPID))
	case "game.qa.discover":
		return c.Post("game", targetGameParams(qaDiscoverParamsFromQAStep(step), step.targetPID))
	case "game.click":
		return c.Post("game", targetGameParams(gameClickParamsFromQAStep(step), step.targetPID))
	case "game.input":
		return c.Post("game", targetGameParams(gameInputParamsFromQAStep(step), step.targetPID))
	case "game.clock":
		return c.Post("game", targetGameParams(gameClockParamsFromQAStep(step), step.targetPID))
	case "game.input_log":
		return c.Post("game", targetGameParams(gameInputLogParamsFromQAStep(step), step.targetPID))
	case "game.ui.tree":
		return c.Post("game", targetGameParams(gameUITreeParamsFromQAStep(step), step.targetPID))
	case "game.ui.audit":
		return c.Post("game", targetGameParams(gameUIAuditParamsFromQAStep(step), step.targetPID))
	case "game.assert":
		return c.Post("game", targetGameParams(map[string]any{"action": "assert", "path": normalizeGameNodePath(step.Path), "prop": step.Prop, "op": step.Op, "value": step.Value}, step.targetPID))
	case "screenshot.runtime":
		return c.Post("game", targetGameParams(screenshotParamsFromQAStep(step), step.targetPID))
	case "diagnostics":
		resp, err := c.Post("diagnostics", diagnosticsParamsFromQAStep(step))
		if err != nil || !resp.OK {
			return resp, err
		}
		return resp, validateDiagnosticsThresholds(resp, step)
	default:
		return nil, fmt.Errorf("unknown qa tool %q", step.Tool)
	}
}

func runParamsFromQAStep(step gameQAStep) map[string]any {
	if step.Action != "" {
		params := cloneJSONMap(step.Params)
		params["action"] = step.Action
		return params
	}
	if step.Current {
		return map[string]any{"action": "play_current"}
	}
	if step.Scene != "" {
		return map[string]any{"action": "play_custom", "scene": step.Scene}
	}
	return map[string]any{"action": "play_main"}
}

func gameNodeGetParamsFromQAStep(step gameQAStep) map[string]any {
	params := map[string]any{"action": "get", "path": normalizeGameNodePath(step.Path)}
	if len(step.Props) > 0 {
		params["props"] = step.Props
	} else if step.Prop != "" {
		params["prop"] = step.Prop
	}
	return params
}

func gameUITreeParamsFromQAStep(step gameQAStep) map[string]any {
	params := cloneJSONMap(step.Params)
	params["action"] = "ui_tree"
	if step.Path != "" {
		params["path"] = normalizeGameNodePath(step.Path)
	}
	if step.Text != "" {
		params["text"] = step.Text
	}
	return params
}

func gameUIAuditParamsFromQAStep(step gameQAStep) map[string]any {
	params := cloneJSONMap(step.Params)
	params["action"] = "ui_audit"
	if step.Path != "" {
		params["path"] = normalizeGameNodePath(step.Path)
	}
	return params
}

func screenshotParamsFromQAStep(step gameQAStep) map[string]any {
	params := map[string]any{"action": "screenshot", "analyze": true}
	if step.Path != "" {
		params["path"] = step.Path
	}
	return params
}

func diagnosticsParamsFromQAStep(step gameQAStep) map[string]any {
	if step.Lines > 0 {
		return map[string]any{"lines": step.Lines}
	}
	return map[string]any{}
}

func validateDiagnosticsThresholds(resp *protocol.Response, step gameQAStep) error {
	data, ok := resp.Data.(map[string]any)
	if !ok {
		return fmt.Errorf("diagnostics returned unexpected data")
	}
	maxWarnings := noGameQAWarningLimit
	if step.MaxWarnings != nil {
		maxWarnings = *step.MaxWarnings
	}
	_, issues := evaluateGameQADiagnostics(data, gameQADiagnoseOptions{maxErrors: step.MaxErrors, maxWarnings: maxWarnings})
	if len(issues) > 0 {
		return fmt.Errorf("%s", strings.Join(issues, "; "))
	}
	return nil
}

func numericField(values map[string]any, key string) (int, bool) {
	value, ok := values[key]
	if !ok {
		return 0, false
	}
	switch typed := value.(type) {
	case float64:
		if math.IsNaN(typed) || typed < 0 || typed >= -float64(math.MinInt) || math.Trunc(typed) != typed {
			return 0, false
		}
		return int(typed), true
	case int:
		return typed, typed >= 0
	default:
		return 0, false
	}
}
