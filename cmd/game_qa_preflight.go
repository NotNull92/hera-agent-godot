package cmd

import "fmt"

func validateGameQAStep(step gameQAStep) error {
	if step.valueMissing {
		return fmt.Errorf("%s requires value (explicit null is accepted)", step.Tool)
	}
	switch step.Tool {
	case "wait":
		if step.DurationMS < 0 || int64(step.DurationMS) > maxDurationMilliseconds {
			return fmt.Errorf("wait duration_ms must be between 0 and %d", maxDurationMilliseconds)
		}
	case "run":
		params := runParamsFromQAStep(step)
		switch params["action"] {
		case "play_main", "play_current", "stop", "state":
		case "play_custom":
			if scene, ok := params["scene"].(string); !ok || scene == "" {
				return fmt.Errorf("run play_custom requires scene")
			}
		default:
			return fmt.Errorf("unknown run action %q", step.Action)
		}
	case "game.node.get", "game.node.set", "game.node.call", "game.assert":
		if step.Path == "" {
			return fmt.Errorf("%s requires path", step.Tool)
		}
		if (step.Tool == "game.node.set" || step.Tool == "game.assert") && step.Prop == "" {
			return fmt.Errorf("%s requires prop", step.Tool)
		}
		if step.Tool == "game.node.call" && step.Method == "" {
			return fmt.Errorf("game.node.call requires method")
		}
		if step.Tool == "game.assert" {
			if !validGameAssertOp(step.Op) {
				return fmt.Errorf("unknown assertion op %q", step.Op)
			}
		}
	case "diagnostics":
		if step.MaxErrors < 0 || (step.MaxWarnings != nil && *step.MaxWarnings < 0) {
			return fmt.Errorf("diagnostics thresholds must be non-negative")
		}
	case "stop", "game.qa.discover", "game.click", "game.input", "game.clock", "game.input_log", "game.ui.tree", "game.ui.audit", "screenshot.runtime":
	default:
		return fmt.Errorf("unknown qa tool %q", step.Tool)
	}
	if step.Lines < 0 {
		return fmt.Errorf("lines must be non-negative")
	}
	return nil
}
