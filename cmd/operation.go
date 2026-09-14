package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func runOperation(args []string) int {
	params, err := parseOperation(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, "operation:", err)
		return 2
	}
	return dialMutationPostPrint("operation", params, "operation")
}

func parseOperation(args []string) (map[string]any, error) {
	if len(args) < 2 || !validOperationID(args[1]) {
		return nil, fmt.Errorf("usage: operation status|cancel <session:unix_ms:nonce>; operation submit <id> --request JSON")
	}
	params := map[string]any{"action": args[0], "id": args[1]}
	switch args[0] {
	case "status", "cancel":
		if len(args) != 2 {
			return nil, fmt.Errorf("%s requires only an operation ID", args[0])
		}
	case "submit":
		if len(args) != 4 || args[2] != "--request" {
			return nil, fmt.Errorf("submit requires <id> --request JSON")
		}
		raw := args[3]
		var input struct {
			Tool   string          `json:"tool"`
			Params json.RawMessage `json:"params"`
		}
		if len(raw) > 16384 || json.Unmarshal([]byte(raw), &input) != nil || input.Tool == "" || len(input.Params) == 0 || input.Params[0] != '{' {
			return nil, fmt.Errorf("request must be a JSON object with tool and params, at most 16384 bytes")
		}
		digest := sha256.Sum256([]byte(raw))
		params["request"] = raw
		params["digest"] = hex.EncodeToString(digest[:])
	default:
		return nil, fmt.Errorf("unknown action %q (want submit|status|cancel)", args[0])
	}
	return params, nil
}

func validOperationID(id string) bool {
	parts := strings.Split(id, ":")
	if len(id) > 160 || len(parts) != 3 || parts[0] == "" || parts[2] == "" {
		return false
	}
	deadline, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || deadline <= 0 || deadline > 9007199254740991 || strconv.FormatInt(deadline, 10) != parts[1] {
		return false
	}
	for _, c := range parts[0] + parts[2] {
		if !strings.ContainsRune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-", c) {
			return false
		}
	}
	return true
}
