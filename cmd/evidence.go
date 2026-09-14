package cmd

import (
	"encoding/hex"
	"fmt"
)

func prepareLinkedEvidence(tool string, params map[string]any) bool {
	if tool == "batch" {
		requested := false
		commands, _ := params["commands"].([]any)
		for _, command := range commands {
			entry, _ := command.(map[string]any)
			subTool, _ := entry["tool"].(string)
			subParams, _ := entry["params"].(map[string]any)
			if subTool != "batch" && prepareLinkedEvidence(subTool, subParams) {
				requested = true
			}
		}
		return requested
	}
	save := tool == "scene" && (params["action"] == "save" || params["action"] == "save_evidence")
	set := tool == "resource" && (params["action"] == "set" || params["action"] == "set_evidence")
	_, expectedHash := params["expected_sha256"]
	if params["evidence"] != true && !(expectedHash && (save || set)) {
		return false
	}
	params["evidence"] = true
	if save {
		params["action"] = "save_evidence"
	} else if set {
		params["action"] = "set_evidence"
	}
	return true
}

func linkedEvidenceAvailable(tool string, params map[string]any, raw any) bool {
	data, _ := raw.(map[string]any)
	if tool != "batch" {
		evidence, _ := data["evidence"].(map[string]any)
		return evidence["available"] == true
	}
	commands, _ := params["commands"].([]any)
	results, _ := data["results"].([]any)
	available := true
	for index, command := range commands {
		entry, _ := command.(map[string]any)
		subParams, _ := entry["params"].(map[string]any)
		if entry["tool"] == "batch" || subParams["evidence"] != true {
			continue
		}
		if index >= len(results) {
			available = false
			continue
		}
		result, _ := results[index].(map[string]any)
		if result["ok"] != true {
			available = false
			continue
		}
		subTool, _ := entry["tool"].(string)
		if result["tool"] != subTool || !linkedEvidenceAvailable(subTool, subParams, result["data"]) {
			result["ok"] = false
			result["error"] = "evidence_unavailable"
			available = false
		}
	}
	return available
}

func parseEvidenceOptions(args []string, capture bool) ([]string, map[string]any, error) {
	params := map[string]any{}
	rest := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--evidence":
			params["evidence"] = true
		case "--operation-id", "--runtime-session", "--expected-sha256":
			flag := args[i]
			if i+1 == len(args) {
				return nil, nil, fmt.Errorf("%s requires a value", flag)
			}
			i++
			value := args[i]
			switch flag {
			case "--operation-id":
				if !capture || !validOperationID(value) {
					return nil, nil, fmt.Errorf("--operation-id requires session:unix_ms:nonce on a capture")
				}
				params["correlation_operation_id"] = value
			case "--runtime-session":
				if !capture || len(value) == 0 || len(value) > 160 {
					return nil, nil, fmt.Errorf("--runtime-session requires a session on a capture")
				}
				params["runtime_session_id"] = value
			case "--expected-sha256":
				decoded, err := hex.DecodeString(value)
				if capture || err != nil || len(decoded) != 32 {
					return nil, nil, fmt.Errorf("--expected-sha256 requires a SHA-256 hash on a save")
				}
				params["expected_sha256"] = value
			}
			params["evidence"] = true
		default:
			rest = append(rest, args[i])
		}
	}
	return rest, params, nil
}
