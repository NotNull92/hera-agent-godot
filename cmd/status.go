package cmd

import (
	"fmt"
	"os"
)

// runStatus implements `hera-godot status`: find a live editor, ask it for status,
// and print the result as compact JSON. Experimental capability detail is
// omitted unless --capabilities is passed.
func runStatus(args []string) int {
	detail, err := parseStatusArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "status: %v\n", err)
		return 2
	}
	return dialAndPostPrint(dialEditor, postPrintRequest{tool: "status", params: map[string]any{"capabilities": detail}, label: "status"})
}

func parseStatusArgs(args []string) (bool, error) {
	detail := false
	for _, arg := range args {
		switch arg {
		case "--capabilities":
			detail = true
		default:
			return false, fmt.Errorf("unknown flag %q", arg)
		}
	}
	return detail, nil
}
