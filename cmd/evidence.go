package cmd

import (
	"encoding/hex"
	"fmt"
)

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
