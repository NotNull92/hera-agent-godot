package cmd

// runScene implements `hera-agent-godot scene <tree|open|save>`.
//
// Maps to the addon `scene` tool (EditorInterface: GetEditedSceneRoot,
// OpenSceneFromPath, SaveScene; GetOpenScenes).
//
// TODO(phase3/4): subcommand parsing + params.
func runScene(args []string) int {
	_ = args
	notImplemented("scene")
	return 1
}
