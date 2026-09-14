package cmd

import (
	"fmt"
	"os"
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
	params, err := parseOutputArgs(args)
	if err != nil {
		return nil, err
	}
	if _, hasType := params["type"]; hasType {
		return nil, fmt.Errorf("unknown flag %q", "--type")
	}
	return params, nil
}
