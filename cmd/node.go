package cmd

// runNode implements `hera-agent-godot node <find|add|set|remove>`.
//
// Maps to the addon `node` tool, operating on the edited scene tree. Godot
// concepts: Node + NodePath + property names (not GameObject/Component).
//
// TODO(phase3 read / phase4 write): subcommand + flag parsing.
func runNode(args []string) int {
	_ = args
	notImplemented("node")
	return 1
}
