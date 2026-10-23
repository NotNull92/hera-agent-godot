package cmd

import (
	"bytes"
	"encoding/json"
	"maps"
	"reflect"
	"strings"
)

// slimData only visits response containers, never user values or evidence.
func slimData(raw any) any {
	data, ok := raw.(map[string]any)
	if !ok {
		return raw
	}
	out := maps.Clone(data)
	if out["truncated"] == false {
		delete(out, "truncated")
	}
	if count, ok := out["count"]; ok {
		for _, value := range out {
			list := reflect.ValueOf(value)
			if list.IsValid() && list.Kind() == reflect.Slice && equalJSON(count, list.Len()) {
				delete(out, "count")
				break
			}
		}
	}
	for _, key := range []string{"nodes", "controls"} {
		if nodes, ok := out[key].([]any); ok {
			entries := make([]any, len(nodes))
			for i, node := range nodes {
				entries[i] = slimNode(slimData(node))
			}
			out[key] = entries
		}
	}
	return out
}

func slimNode(raw any) any {
	data, ok := raw.(map[string]any)
	if !ok {
		return raw
	}
	path, ok := data["path"].(string)
	if !ok || path == "." || path == "" {
		return raw
	}
	name, ok := data["name"].(string)
	if !ok || name == "" || name != path[strings.LastIndex(path, "/")+1:] {
		return raw
	}
	out := maps.Clone(data)
	delete(out, "name")
	return out
}

func equalJSON(a, b any) bool {
	left, e1 := json.Marshal(a)
	right, e2 := json.Marshal(b)
	return e1 == nil && e2 == nil && bytes.Equal(left, right)
}

func reshapeOutput(tool string, params map[string]any, raw any) any {
	data, ok := raw.(map[string]any)
	if !ok {
		return raw
	}
	out := maps.Clone(data)
	switch tool {
	case "status":
		if params["capabilities"] == true {
			if caps, ok := out["capabilities"].(map[string]any); ok {
				out["capabilities"] = maps.Clone(caps)
			}
			out = statusWithSDK(out).(map[string]any)
		} else {
			out = compactStatusData(out).(map[string]any)
		}
		delete(out, "editor_session_id")
		delete(out, "project_path")
	case "output", "diagnostics":
		if out["source"] != "editor" {
			if out["available"] == true {
				delete(out, "log_path")
			}
			if out["source"] == "file" {
				delete(out, "source")
			}
		}
	case "node":
		if params["action"] == "get" {
			out = slimNode(out).(map[string]any)
		}
	case "game":
		if params["action"] == "get" {
			out = slimNode(out).(map[string]any)
		}
		if params["action"] == "input" {
			for _, key := range []string{"kind", "mode", "x", "y", "button", "key", "keycode", "physical", "route", "name", "strength", "text", "axis", "value", "device"} {
				if value, exists := params[key]; exists && equalJSON(value, out[key]) {
					delete(out, key)
				}
			}
		}
	case "batch":
		commands, _ := params["commands"].([]any)
		if results, ok := out["results"].([]any); ok {
			entries := make([]any, len(results))
			for i, result := range results {
				child, ok := result.(map[string]any)
				if !ok {
					entries[i] = result
					continue
				}
				entry := maps.Clone(child)
				childTool, _ := entry["tool"].(string)
				var childParams map[string]any
				if i < len(commands) {
					command, _ := commands[i].(map[string]any)
					childParams, _ = command["params"].(map[string]any)
				}
				if payload, exists := entry["data"]; exists {
					entry["data"] = reshapeOutput(childTool, childParams, payload)
				}
				entries[i] = entry
			}
			out["results"] = entries
		}
	}
	return slimData(out)
}
