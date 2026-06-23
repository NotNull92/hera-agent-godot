package cmd

import (
	"fmt"
	"os"
)

// runStatus implements `hera-agent-godot status`: find a live editor, ask it for
// status, and print the result as compact JSON.
func runStatus(args []string) int {
	_ = args

	c, err := dialEditor()
	if err != nil {
		fmt.Fprintf(os.Stderr, "status: %v\n", err)
		return 1
	}

	resp, err := c.Post("status", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "status: %v\n", err)
		return 1
	}
	if !resp.OK {
		fmt.Fprintf(os.Stderr, "status: %s\n", resp.Error)
		return 1
	}
	return printData(resp)
}
