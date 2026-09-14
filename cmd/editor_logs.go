package cmd

import (
	"fmt"
	"os"

	"github.com/NotNull92/hera-agent-godot/internal/client"
	"github.com/NotNull92/hera-agent-godot/internal/protocol"
)

func dialEditorLogPrint(tool string, params map[string]any) int {
	c, err := dialEditor()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", tool, err)
		return 1
	}
	status, err := c.Post("status", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", tool, err)
		return 1
	}
	if !status.OK {
		fmt.Fprintf(os.Stderr, "%s: %s\n", tool, status.Error)
		return 1
	}
	data, _ := status.Data.(map[string]any)
	capabilities, _ := data["capabilities"].(map[string]any)
	if capabilities["editor_log_cursor"] != "supported" {
		state := "unverified"
		if capabilities["editor_log_cursor"] == "unsupported" {
			state = "unsupported"
		}
		return printData(&protocol.Response{OK: true, Data: map[string]any{
			"source": "editor", "available": false, "clean": false,
			"reason": "evidence_unavailable", "capability": state,
			"editor_session_id": data["editor_session_id"],
			"hint":              "This addon has no verified editor log collector. Startup --log-file capture remains separate.",
		}})
	}
	return dialAndPostPrint(func() (*client.Client, error) { return c, nil }, postPrintRequest{tool: tool, params: params, label: tool})
}
