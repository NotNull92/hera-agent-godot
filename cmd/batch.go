package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// runBatch implements `hera batch [--file <path>] [--continue]`.
//
// Reads a JSON array of {tool, params} from --file (or stdin) and runs them in a
// single request. By default it stops at the first failing command; --continue
// runs them all. Treated as a mutation command (requires exactly one live
// editor) since a batch may contain mutations.
func runBatch(args []string) int {
	file, keepGoing, err := parseBatchFlags(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "batch: %v\n", err)
		return 2
	}

	var raw []byte
	if file != "" {
		raw, err = os.ReadFile(file)
	} else {
		raw, err = io.ReadAll(os.Stdin)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "batch: %v\n", err)
		return 1
	}

	var commands []any
	if err := json.Unmarshal(raw, &commands); err != nil {
		fmt.Fprintf(os.Stderr, "batch: invalid JSON commands array: %v\n", err)
		return 2
	}

	params := map[string]any{"commands": commands, "stop_on_error": !keepGoing}
	return dialMutationPostPrint("batch", params, "batch")
}

func parseBatchFlags(args []string) (file string, keepGoing bool, err error) {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--file":
			if i+1 >= len(args) {
				return "", false, fmt.Errorf("--file requires a path")
			}
			i++
			file = args[i]
		case "--continue":
			keepGoing = true
		default:
			return "", false, fmt.Errorf("unknown flag %q", args[i])
		}
	}
	return file, keepGoing, nil
}
