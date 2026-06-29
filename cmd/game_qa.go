package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/NotNull92/hera-agent-godot/internal/client"
	"github.com/NotNull92/hera-agent-godot/internal/protocol"
)

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
	Params      map[string]any `json:"params"`
}

type gameQAResult struct {
	Step  int    `json:"step"`
	Tool  string `json:"tool"`
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

func runGameQA(args []string) int {
	file, keepGoing, err := parseGameQAFlags(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "game qa: %v\n", err)
		return 2
	}
	steps, err := readGameQASteps(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "game qa: %v\n", err)
		return 2
	}
	c, err := dialMutationEditor()
	if err != nil {
		fmt.Fprintf(os.Stderr, "game qa: %v\n", err)
		return 1
	}
	results, ok := executeGameQASteps(c, steps, keepGoing)
	resp := &protocol.Response{OK: true, Data: map[string]any{"ok": ok, "steps": len(steps), "results": results}}
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

func readGameQASteps(file string) ([]gameQAStep, error) {
	raw, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var steps []gameQAStep
	if err := json.Unmarshal(raw, &steps); err != nil {
		return nil, fmt.Errorf("invalid scenario JSON: %w", err)
	}
	if len(steps) == 0 {
		return nil, fmt.Errorf("scenario must contain at least one step")
	}
	for index, step := range steps {
		if step.Tool == "" {
			return nil, fmt.Errorf("step %d: tool is required", index+1)
		}
	}
	return steps, nil
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
	resp, err := postGameQAStep(c, step)
	if err != nil {
		return gameQAResult{Step: index, Tool: step.Tool, OK: false, Error: err.Error()}
	}
	if !resp.OK {
		return gameQAResult{Step: index, Tool: step.Tool, OK: false, Error: resp.Error}
	}
	return gameQAResult{Step: index, Tool: step.Tool, OK: true}
}

func postGameQAStep(c *client.Client, step gameQAStep) (*protocol.Response, error) {
	switch step.Tool {
	case "wait":
		if step.DurationMS < 0 {
			return nil, fmt.Errorf("wait duration_ms must be non-negative")
		}
		time.Sleep(time.Duration(step.DurationMS) * time.Millisecond)
		return &protocol.Response{OK: true, Data: map[string]any{"waited_ms": step.DurationMS}}, nil
	case "run":
		params := runParamsFromQAStep(step)
		resp, err := c.Post("run", params)
		if err != nil || !step.Wait {
			return resp, err
		}
		_, waitErr := pollPlaying(c, true, waitTimeout)
		if waitErr != nil {
			return resp, waitErr
		}
		_, waitErr = pollGameReady(c, sceneFromResponse(resp), waitTimeout)
		return resp, waitErr
	case "stop":
		params := map[string]any{"action": "stop"}
		resp, err := c.Post("run", params)
		if err != nil || !step.Wait {
			return resp, err
		}
		_, waitErr := pollPlaying(c, false, waitTimeout)
		return resp, waitErr
	case "game.node.get":
		return c.Post("game", gameNodeGetParamsFromQAStep(step))
	case "game.node.set":
		return c.Post("game", map[string]any{"action": "set", "path": normalizeGameNodePath(step.Path), "prop": step.Prop, "value": step.Value})
	case "game.node.call":
		return c.Post("game", map[string]any{"action": "call", "path": normalizeGameNodePath(step.Path), "method": step.Method, "args": step.Args})
	case "game.click":
		return c.Post("game", gameClickParamsFromQAStep(step))
	case "game.ui.tree":
		return c.Post("game", map[string]any{"action": "ui_tree"})
	case "game.assert":
		return c.Post("game", map[string]any{"action": "assert", "path": normalizeGameNodePath(step.Path), "prop": step.Prop, "op": step.Op, "value": step.Value})
	case "screenshot.runtime":
		return c.Post("game", screenshotParamsFromQAStep(step))
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
	errorCount, _ := numericField(data, "error_count")
	warningCount, _ := numericField(data, "warning_count")
	if errorCount > step.MaxErrors {
		return fmt.Errorf("diagnostics errors = %d, want <= %d", errorCount, step.MaxErrors)
	}
	if step.MaxWarnings > 0 && warningCount > step.MaxWarnings {
		return fmt.Errorf("diagnostics warnings = %d, want <= %d", warningCount, step.MaxWarnings)
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
		return int(typed), true
	case int:
		return typed, true
	default:
		return 0, false
	}
}
