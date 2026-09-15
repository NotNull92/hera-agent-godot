package cmd

import "maps"

func compactStatusData(raw any) any {
	data, ok := raw.(map[string]any)
	if !ok {
		return raw
	}
	out := maps.Clone(data)
	delete(out, "capabilities")
	delete(out, "godot_commit")
	return out
}

func compactReceiptData(raw any) any {
	data, ok := raw.(map[string]any)
	if !ok {
		return raw
	}
	out := maps.Clone(data)
	delete(out, "retention")
	if evidence, ok := out["evidence"].(map[string]any); ok {
		trimmed := maps.Clone(evidence)
		delete(trimmed, "response")
		out["evidence"] = trimmed
	}
	return out
}
