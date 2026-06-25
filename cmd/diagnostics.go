package cmd

import (
	"fmt"
	"os"
	"strconv"
)

func runDiagnostics(args []string) int {
	params, err := parseDiagnosticsArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "diagnostics: %v\n", err)
		return 2
	}
	return dialPostPrint("diagnostics", params, "diagnostics")
}

func parseDiagnosticsArgs(args []string) (map[string]any, error) {
	params := map[string]any{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--lines":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--lines requires a value")
			}
			i++
			n, err := strconv.Atoi(args[i])
			if err != nil || n <= 0 {
				return nil, fmt.Errorf("invalid --lines %q (want a positive integer)", args[i])
			}
			params["lines"] = n
		default:
			return nil, fmt.Errorf("unknown flag %q", args[i])
		}
	}
	return params, nil
}
