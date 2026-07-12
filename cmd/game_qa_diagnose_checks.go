package cmd

import "fmt"

func evaluateGameQADiagnostics(data map[string]any, options gameQADiagnoseOptions) (map[string]any, []string) {
	errors, errorsOK := numericField(data, "error_count")
	warnings, warningsOK := numericField(data, "warning_count")
	if !errorsOK || !warningsOK {
		return gameQADiagnoseFailure("editor_diagnostics", fmt.Errorf("missing error or warning counts")), []string{"editor diagnostics response is incomplete"}
	}
	check := map[string]any{"name": "editor_diagnostics", "ok": true, "errors": errors, "warnings": warnings}
	issues := make([]string, 0, 2)
	if errors > options.maxErrors {
		check["ok"] = false
		issues = append(issues, fmt.Sprintf("editor diagnostics report %d errors (want <= %d)", errors, options.maxErrors))
	}
	if options.maxWarnings >= 0 && warnings > options.maxWarnings {
		check["ok"] = false
		issues = append(issues, fmt.Sprintf("editor diagnostics report %d warnings (want <= %d)", warnings, options.maxWarnings))
	}
	return check, issues
}

func evaluateGameQAInstances(data map[string]any) (map[string]any, []string) {
	instances, ok := data["instances"].([]any)
	if !ok {
		return gameQADiagnoseFailure("runtime_instances", fmt.Errorf("missing instances")), []string{"runtime instance response is incomplete"}
	}
	check := map[string]any{"name": "runtime_instances", "ok": len(instances) == 1, "count": len(instances)}
	if len(instances) == 1 {
		return check, nil
	}
	return check, []string{fmt.Sprintf("expected exactly one live game process, found %d", len(instances))}
}

func evaluateGameQATree(data map[string]any) (map[string]any, []string) {
	count, countOK := numericField(data, "count")
	truncated, truncatedOK := data["truncated"].(bool)
	if !countOK || !truncatedOK {
		return gameQADiagnoseFailure("runtime_tree", fmt.Errorf("missing node count or truncation state")), []string{"runtime tree response is incomplete"}
	}
	check := map[string]any{"name": "runtime_tree", "ok": !truncated, "nodes": count, "truncated": truncated}
	if scene, ok := data["scene"].(string); ok && scene != "" {
		check["scene"] = scene
	}
	if truncated {
		return check, []string{"runtime node tree was truncated"}
	}
	return check, nil
}

func evaluateGameQAUI(data map[string]any) (map[string]any, []string) {
	count, countOK := numericField(data, "count")
	truncated, truncatedOK := data["truncated"].(bool)
	if !countOK || !truncatedOK {
		return gameQADiagnoseFailure("runtime_ui", fmt.Errorf("missing control count or truncation state")), []string{"runtime UI tree response is incomplete"}
	}
	check := map[string]any{"name": "runtime_ui", "ok": !truncated, "controls": count, "truncated": truncated}
	if truncated {
		return check, []string{"runtime UI tree was truncated"}
	}
	return check, nil
}

func evaluateGameQAScreenshot(data map[string]any) (map[string]any, []string) {
	analysis, ok := data["analysis"].(map[string]any)
	if !ok {
		return gameQADiagnoseFailure("runtime_screenshot", fmt.Errorf("missing image analysis")), []string{"runtime screenshot analysis is unavailable"}
	}
	nonblank, nonblankOK := analysis["nonblank"].(bool)
	lowDetail, lowDetailOK := analysis["low_detail"].(bool)
	clipped, clippedOK := analysis["possible_clipping"].(bool)
	if !nonblankOK || !lowDetailOK || !clippedOK {
		return gameQADiagnoseFailure("runtime_screenshot", fmt.Errorf("incomplete image analysis")), []string{"runtime screenshot analysis is incomplete"}
	}
	check := map[string]any{
		"name":              "runtime_screenshot",
		"ok":                nonblank && !lowDetail && !clipped,
		"nonblank":          nonblank,
		"low_detail":        lowDetail,
		"possible_clipping": clipped,
	}
	issues := make([]string, 0, 3)
	if !nonblank {
		issues = append(issues, "runtime screenshot is blank")
	}
	if lowDetail {
		issues = append(issues, "runtime screenshot has too little visual detail")
	}
	if clipped {
		issues = append(issues, "runtime screenshot may be clipped")
	}
	return check, issues
}

func gameQADiagnoseFailure(name string, err error) map[string]any {
	return map[string]any{"name": name, "ok": false, "error": err.Error()}
}
