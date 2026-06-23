extends RefCounted

# `run` — control the editor's play session via EditorInterface.
#
# params.action:
#   play_main     -> play_main_scene()
#   play_current  -> play_current_scene()
#   play_custom   -> play_custom_scene(params.scene)
#   stop          -> stop_playing_scene()
#   state         -> just report { playing, scene } (used by the CLI's --wait poll)
#
# Returns { playing: bool, scene: String }. Play spawns a separate game process,
# so `playing` may still be false in the immediate response right after a play
# call — the CLI polls `state` to wait for it to flip.

const ToolResponse = preload("res://addons/hera_agent_godot/core/tool_response.gd")

func get_name() -> String:
	return "run"

func execute(params: Dictionary) -> Dictionary:
	var action := String(params.get("action", ""))
	match action:
		"play_main":
			EditorInterface.play_main_scene()
			return _state()
		"play_current":
			EditorInterface.play_current_scene()
			return _state()
		"play_custom":
			var scene := String(params.get("scene", ""))
			if scene == "":
				return ToolResponse.failure("play_custom requires a 'scene' path")
			if not ResourceLoader.exists(scene, "PackedScene"):
				return ToolResponse.failure("scene not found or not a PackedScene: %s" % scene)
			EditorInterface.play_custom_scene(scene)
			return _state()
		"stop":
			EditorInterface.stop_playing_scene()
			return _state()
		"state":
			return _state()
		_:
			return ToolResponse.failure("unknown run action: %s (want play_main|play_current|play_custom|stop|state)" % action)

func _state() -> Dictionary:
	return ToolResponse.success({
		"playing": EditorInterface.is_playing_scene(),
		"scene": EditorInterface.get_playing_scene(),
	})
