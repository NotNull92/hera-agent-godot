package cmd

import (
	"github.com/NotNull92/hera-agent-godot/internal/protocol"
)

func editorLogUnavailable(status *protocol.Response, caps map[string]any) *protocol.Response {
	state := "unverified"
	if capabilityValue(caps, "editor_log_cursor") == "unsupported" {
		state = "unsupported"
	}
	data, _ := status.Data.(map[string]any)
	return &protocol.Response{OK: true, Data: map[string]any{
		"source": "editor", "available": false, "clean": false,
		"reason": "evidence_unavailable", "capability": state,
		"editor_session_id": data["editor_session_id"],
		"hint":              "This addon has no verified editor log collector. Startup --log-file capture remains separate.",
	}}
}
