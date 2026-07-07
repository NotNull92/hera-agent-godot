package cmd

func gameClickParamsFromQAStep(step gameQAStep) map[string]any {
	params := map[string]any{"action": "click"}
	if step.Path != "" {
		params["path"] = normalizeGameNodePath(step.Path)
		return params
	}
	if step.Text != "" {
		params["text"] = step.Text
		return params
	}
	params["x"] = step.X
	params["y"] = step.Y
	return params
}

func gameInputParamsFromQAStep(step gameQAStep) map[string]any {
	params := cloneJSONMap(step.Params)
	params["action"] = "input"
	if step.X != 0 || step.Y != 0 {
		params["x"] = step.X
		params["y"] = step.Y
	}
	if step.Text != "" {
		params["text"] = step.Text
	}
	return params
}

func gameInputLogParamsFromQAStep(step gameQAStep) map[string]any {
	params := cloneJSONMap(step.Params)
	params["action"] = "input_log"
	if step.Lines > 0 {
		params["limit"] = step.Lines
	}
	return params
}
