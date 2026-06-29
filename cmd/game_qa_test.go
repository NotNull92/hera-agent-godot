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
