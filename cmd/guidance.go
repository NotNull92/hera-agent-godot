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
	fallback := guidanceDataFromStatus(statusResp.Data)
	if params["action"] == "game_feel" {
		fallback = gameFeelGuidanceDataFromStatus(statusResp.Data)
	}
	return printData(&protocol.Response{OK: true, Data: fallback})
}

func parseGuidanceArgs(args []string) (map[string]any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("usage: guidance <ui|game-feel>")
	}
	switch args[0] {
	case "ui":
		return map[string]any{"action": "ui"}, nil
	case "game-feel", "game_feel":
		return map[string]any{"action": "game_feel"}, nil
	default:
		return nil, fmt.Errorf("usage: guidance <ui|game-feel>")
	}
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

func gameFeelGuidanceDataFromStatus(data any) map[string]any {
	status, _ := data.(map[string]any)
	gameFeelEnabled, _ := status["game_feel_mode"].(bool)
	mode := "standard"
	if gameFeelEnabled {
		mode = "game_feel"
	}
	return map[string]any{
		"mode":             mode,
		"game_feel_mode":   gameFeelEnabled,
		"setting":          "hera_agent_godot/game_feel_mode",
		"instruction":      gameFeelInstruction(gameFeelEnabled),
		"checklist":        gameFeelChecklist(gameFeelEnabled),
		"topics":           []string{"ethics_checklist", "control_feel", "screen_shake", "hit_stop", "camera", "sound", "particles", "tweening_easing"},
		"game_qa_patterns": gameFeelQAPatterns(gameFeelEnabled),
		"source":           "status_fallback",
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
			"Keep framed children inside padding bounds; verify text and disabled-state contrast.",
			"Keep sibling panel geometry shared and enforce content-density budgets.",
			"Use live viewport bounds for backgrounds/playfields/HUDs unless fixed resolution is explicit.",
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

func gameFeelInstruction(enabled bool) string {
	if enabled {
		return "Build gameplay feel with concrete Game Feel parameters: responsive controls, honest juice, camera/audio/particle feedback, accessibility options, and runtime QA."
	}
	return "Build gameplay normally; query game_feel topics when you need concrete feel parameters."
}

func gameFeelChecklist(enabled bool) []string {
	if enabled {
		return []string{
			"Start from `game_feel` topic lookup before tuning control, camera, impact, sound, particles, rewards, or presentation.",
			"Keep feedback proportional to real achievement: Honest Juice first.",
			"Add reduce-motion, shake/flash intensity, or off options for strong feedback.",
			"Verify the feel in Play Mode with runtime tree, interactions, screenshot analysis, and output errors.",
			"Before reporting done, validate against `game_feel ethics_checklist` and `game_feel checklist_all`.",
		}
	}
	return []string{
		"Use `game_feel list` to discover concrete topics when gameplay feel matters.",
		"Run the game and observe the player-facing result before calling work done.",
	}
}

func gameFeelQAPatterns(enabled bool) []string {
	if !enabled {
		return []string{}
	}
	return []string{
		"Scope: `guidance game-feel`=gameplay; `guidance ui`=Control layout/input.",
		"Node2D HUD: root Control uses `Control.MOUSE_FILTER_IGNORE`; keep buttons/sidebar interactive.",
		"Realtime/physics: expose restart/start, deterministic step, and targeted event helpers for score/removal/damage/lives.",
		"Delayed/locked state: trigger+step helpers; assert lock flags, visible state, disabled control counts.",
		"Hidden-state rules: avoid preconditions created by earlier QA steps or reset them explicitly.",
		"AI/automated turns: expose priority setup helpers and document/prove the undo boundary.",
		"Autonomous loops: provide a restart-paused helper, pause toggle, and one-step advancement before inspection.",
		"Collision/impact: forced-overlap helpers prove collision, feedback, end-state, and feel evidence.",
		"Wave/economy checks: isolate placement, spawn, reward, spend, leak, and loss; avoid full-run-only QA.",
		"Runtime QA order: state-changing runtime QA is ordered; do not parallelize clicks, input, or `qa_*` calls.",
		"primary input scheme: drive keyboard/mouse/touch/controller through `game input`, not helper-only QA.",
		"Stateful controls: read current text or use stable paths before semantic clicks on toggles/modes.",
		"Terminal states: preserve or append win/loss/pause/game-over terminal-state instruction text.",
		"Viewport layout: draw from the live viewport; keep playfield/HUD rects inside padded bounds.",
		"High-volume UI: scope `game ui tree` with `--type`, `--fields`, `--path`, `--text`, and `--depth`.",
	}
}
