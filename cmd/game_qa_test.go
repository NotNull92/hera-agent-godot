package cmd

import "testing"

func TestScreenshotParamsFromQAStep_enablesAnalysisByDefault_whenRuntimeScreenshot(t *testing.T) {
	// Given
	step := gameQAStep{Tool: "screenshot.runtime", Path: "user://shot.png"}

	// When
	params := screenshotParamsFromQAStep(step)

	// Then
	if params["action"] != "screenshot" {
		t.Fatalf("action = %v, want screenshot", params["action"])
	}
	if params["path"] != "user://shot.png" {
		t.Fatalf("path = %v, want user://shot.png", params["path"])
	}
	if params["analyze"] != true {
		t.Fatalf("analyze = %v, want true", params["analyze"])
	}
}

func TestGameClickParamsFromQAStep_targetsNode_whenPathProvided(t *testing.T) {
	// Given
	step := gameQAStep{Tool: "game.click", Path: "C:/Program Files/Git/root/Main/Restart"}

	// When
	params := gameClickParamsFromQAStep(step)

	// Then
	if params["action"] != "click" {
		t.Fatalf("action = %v, want click", params["action"])
	}
	if params["path"] != "/root/Main/Restart" {
		t.Fatalf("path = %v, want /root/Main/Restart", params["path"])
	}
	if _, ok := params["x"]; ok {
		t.Fatalf("x should be omitted when path is provided: %v", params)
	}
}

func TestGameClickParamsFromQAStep_targetsText_whenTextProvided(t *testing.T) {
	// Given
	step := gameQAStep{Tool: "game.click", Text: "Restart"}

	// When
	params := gameClickParamsFromQAStep(step)

	// Then
	if params["action"] != "click" {
		t.Fatalf("action = %v, want click", params["action"])
	}
	if params["text"] != "Restart" {
		t.Fatalf("text = %v, want Restart", params["text"])
	}
}

func TestGameInputParamsFromQAStep_passesParamsThrough(t *testing.T) {
	// Given
	step := gameQAStep{
		Tool: "game.input",
		X:    320,
		Y:    240,
		Params: map[string]any{
			"kind":      "mouse",
			"mode":      "release",
			"button":    "right",
			"modifiers": []any{"shift", "ctrl"},
		},
	}

	// When
	params := gameInputParamsFromQAStep(step)

	// Then
	if params["action"] != "input" {
		t.Fatalf("action = %v, want input", params["action"])
	}
	if params["kind"] != "mouse" || params["mode"] != "release" || params["button"] != "right" {
		t.Fatalf("input params = %v", params)
	}
	if params["x"] != 320 || params["y"] != 240 {
		t.Fatalf("coordinate params = %v", params)
	}
}

func TestGameInputLogParamsFromQAStep_usesLinesAsLimit(t *testing.T) {
	// Given
	step := gameQAStep{Tool: "game.input_log", Lines: 5, Params: map[string]any{"clear": true}}

	// When
	params := gameInputLogParamsFromQAStep(step)

	// Then
	if params["action"] != "input_log" {
		t.Fatalf("action = %v, want input_log", params["action"])
	}
	if params["limit"] != 5 || params["clear"] != true {
		t.Fatalf("input log params = %v", params)
	}
}

func TestGameNodeGetParamsFromQAStep_passesSelectedProps_whenProvided(t *testing.T) {
	// Given
	step := gameQAStep{Tool: "game.node.get", Path: "/root/Main", Props: []string{"score", "player.position"}}

	// When
	params := gameNodeGetParamsFromQAStep(step)

	// Then
	if params["action"] != "get" {
		t.Fatalf("action = %v, want get", params["action"])
	}
	if params["path"] != "/root/Main" {
		t.Fatalf("path = %v, want /root/Main", params["path"])
	}
	props, ok := params["props"].([]string)
	if !ok {
		t.Fatalf("props = %T, want []string", params["props"])
	}
	if len(props) != 2 || props[0] != "score" || props[1] != "player.position" {
		t.Fatalf("props = %v, want [score player.position]", props)
	}
	if _, ok := params["prop"]; ok {
		t.Fatalf("prop should be omitted when props is provided: %v", params)
	}
}

func TestGameUITreeParamsFromQAStep_passesScopedFilters(t *testing.T) {
	// Given
	step := gameQAStep{
		Tool: "game.ui.tree",
		Path: "C:/Program Files/Git/root/Main/HUD",
		Text: "Restart",
		Params: map[string]any{
			"depth":  2,
			"type":   "Button",
			"fields": []any{"name", "path", "text", "rect", "disabled"},
		},
	}

	// When
	params := gameUITreeParamsFromQAStep(step)

	// Then
	if params["action"] != "ui_tree" {
		t.Fatalf("action = %v, want ui_tree", params["action"])
	}
	if params["path"] != "/root/Main/HUD" {
		t.Fatalf("path = %v, want /root/Main/HUD", params["path"])
	}
	if params["text"] != "Restart" || params["type"] != "Button" || params["depth"] != 2 {
		t.Fatalf("ui tree filters = %v", params)
	}
	fields, ok := params["fields"].([]any)
	if !ok {
		t.Fatalf("fields = %T, want []any", params["fields"])
	}
	if len(fields) != 5 || fields[0] != "name" || fields[4] != "disabled" {
		t.Fatalf("fields = %v", fields)
	}
}
