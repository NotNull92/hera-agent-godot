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
