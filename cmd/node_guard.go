package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"
)

type nodeExpected struct {
	EditorSessionID string `json:"editor_session_id"`
	Scene           string `json:"scene"`
	NodeInstanceID  string `json:"node_instance_id"`
	Prop            string `json:"prop"`
	Type            string `json:"type"`
	Value           string `json:"value"`
}

func parseNodeExpected(raw string) (nodeExpected, error) {
	var fields map[string]*string
	if err := json.Unmarshal([]byte(raw), &fields); err != nil {
		return nodeExpected{}, fmt.Errorf("invalid --expected JSON: %w", err)
	}
	keys := []string{"editor_session_id", "scene", "node_instance_id", "prop", "type", "value"}
	if len(fields) != len(keys) {
		return nodeExpected{}, fmt.Errorf("--expected requires exactly editor_session_id, scene, node_instance_id, prop, type, value")
	}
	for _, key := range keys {
		value := fields[key]
		if value == nil || (*value == "" && key != "scene" && key != "value") {
			return nodeExpected{}, fmt.Errorf("--expected requires string field %s", key)
		}
	}
	id, err := strconv.ParseUint(*fields["node_instance_id"], 10, 64)
	if err != nil || id == 0 || strconv.FormatUint(id, 10) != *fields["node_instance_id"] {
		return nodeExpected{}, fmt.Errorf("expected.node_instance_id must be a positive decimal string")
	}
	return nodeExpected{*fields["editor_session_id"], *fields["scene"], *fields["node_instance_id"], *fields["prop"], *fields["type"], *fields["value"]}, nil
}

func requiresNodeGuard(tool string, params map[string]any) bool {
	if tool == "batch" {
		needed := false
		forEachBatchChild(params, func(sub string, subParams map[string]any) {
			if requiresNodeGuard(sub, subParams) {
				needed = true
			}
		})
		return needed
	}
	if tool != "node" {
		return false
	}
	_, expected := params["expected"]
	_, verify := params["verify"]
	_, snapshot := params["snapshot"]
	return expected || verify || snapshot
}

func useGuardedNodeAction(tool string, params map[string]any) {
	if tool == "node" && params["action"] == "set" {
		_, expected := params["expected"]
		_, verify := params["verify"]
		if expected || verify {
			params["action"] = "set_guarded"
		}
	}
}
