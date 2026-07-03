package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/NotNull92/hera-agent-godot/internal/protocol"
)

func runGuidance(args []string) int {
	params, err := parseGuidanceArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "guidance: %v\n", err)
		return 2
	}
	c, err := dialEditor()
	if err != nil {
		fmt.Fprintf(os.Stderr, "guidance: %v\n", err)
		return 1
	}
	resp, err := c.Post("guidance", params)
	if err != nil {
		fmt.Fprintf(os.Stderr, "guidance: %v\n", err)
		return 1
	}
	if resp.OK {
		return printData(resp)
	}
	if !strings.Contains(resp.Error, "unknown tool") {
		fmt.Fprintf(os.Stderr, "guidance: %s\n", resp.Error)
		return 1
	}
	statusResp, err := c.Post("status", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "guidance: %v\n", err)
		return 1
	}
	if !statusResp.OK {
		fmt.Fprintf(os.Stderr, "guidance: %s\n", statusResp.Error)
		return 1
	}
	return printData(&protocol.Response{OK: true, Data: guidanceDataFromStatus(statusResp.Data)})
}

func parseGuidanceArgs(args []string) (map[string]any, error) {
	if len(args) != 1 || args[0] != "ui" {
		return nil, fmt.Errorf("usage: guidance ui")
	}
	return map[string]any{"action": "ui"}, nil
}

func guidanceDataFromStatus(data any) map[string]any {
	status, _ := data.(map[string]any)
	gameFeelEnabled, _ := status["game_feel_ui_mode"].(bool)
	mode := "standard"
	if gameFeelEnabled {
		mode = "game_feel"
	}
	return map[string]any{
		"mode":              mode,
		"game_feel_ui_mode": gameFeelEnabled,
		"setting":           "hera_agent_godot/ui_juicy_mode",
		"instruction":       guidanceInstruction(gameFeelEnabled),
		"checklist":         guidanceChecklist(gameFeelEnabled),
		"source":            "status_fallback",
	}
}

func guidanceInstruction(enabled bool) string {
	if enabled {
		return "Build UI with Game Feel: immediate feedback, crisp transitions, expressive state changes, satisfying input response, and runtime visual QA."
	}
	return "Build UI with the standard Hera guidance: clear layout, readable state, predictable controls, and runtime visual QA."
}

func guidanceChecklist(enabled bool) []string {
	if enabled {
		return []string{
			"Give every primary input immediate pressed/hover/disabled feedback.",
			"Use brief motion, squash, pulse, flash, or count-up feedback where it clarifies state.",
			"Make success, failure, damage, reward, and progress changes visibly satisfying.",
			"Keep motion bounded and verify it does not cause clipping or layout shift.",
			"Capture runtime UI tree and screenshot analysis after interaction.",
		}
	}
	return []string{
		"Keep layout readable and stable.",
		"Expose clear state labels and reachable controls.",
		"Verify Control rects through runtime UI tree.",
		"Capture screenshot analysis for visual changes.",
	}
}
