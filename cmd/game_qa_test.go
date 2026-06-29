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
