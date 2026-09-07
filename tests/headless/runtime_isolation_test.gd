extends SceneTree

const GameTool = preload("res://addons/hera_agent_godot/tools/game_tool.gd")
const GameViewportActions = preload("res://addons/hera_agent_godot/runtime/game_viewport_actions.gd")

var _failed := false


func _initialize() -> void:
	_check(GameViewportActions.sizes_match(1080, 1920, 1080, 1920), "matching project size is accepted")
	_check(not GameViewportActions.sizes_match(927, 1649, 1080, 1920), "smaller capture is not treated as the project size")
	_check(not GameViewportActions.sizes_match(1080, 1920, 0, 0), "unknown project size does not match")
	var metrics: Dictionary = GameViewportActions.geometry(root)
	_check(metrics.has("visible_width") and metrics.has("window_width") and metrics.has("project_width"), "geometry reports window, visible, and project size")
	_check(metrics.has("size_matches_project"), "geometry reports size_matches_project")

	var tool := GameTool.new()
	var instances := [
		{ "pid": 1, "user_data_dir": "C:/shared" },
		{ "pid": 2, "user_data_dir": "C:/shared" },
		{ "pid": 3, "user_data_dir": "C:/other" },
	]
	tool._mark_shared_user_data(instances)
	_check(instances[0].get("shared_user_data") == true, "shared user data is marked")
	_check(instances[1].get("shared_user_data") == true, "second shared process is marked")
	_check(not bool(instances[2].get("shared_user_data", false)), "isolated process stays unmarked")
	quit(1 if _failed else 0)


func _check(value: bool, message: String) -> void:
	if not value:
		_fail(message)


func _fail(message: String) -> void:
	_failed = true
	push_error("FAIL: %s" % message)
