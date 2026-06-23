package cmd

// runRun implements `hera-agent-godot run [--scene <res://...>] [--current] [--wait]`.
//
// Maps to the addon `run` tool, which wraps EditorInterface.PlayMainScene /
// PlayCurrentScene / PlayCustomScene.
//
// TODO(phase2): parse --scene/--current/--wait; build params; post; optionally
// poll heartbeat state until play starts.
func runRun(args []string) int {
	_ = args
	notImplemented("run")
	return 1
}

// runStop implements `hera-agent-godot stop` (addon `run` tool, StopPlayingScene).
//
// TODO(phase2).
func runStop(args []string) int {
	_ = args
	notImplemented("stop")
	return 1
}
