package cmd

// runStatus implements `hera-agent-godot status`: find a live editor, ask it for
// status, and print the result as compact JSON.
func runStatus(args []string) int {
	_ = args
	return dialPostPrint("status", nil, "status")
}
