extends SceneTree

const GameTool = preload("res://addons/hera_agent_godot/tools/game_tool.gd")

func _init() -> void:
	call_deferred("_run")

func _run() -> void:
	var tool := GameTool.new()
	if not tool.has_method("_select_target"):
		fail("GameTool must support explicit PID selection")
		quit(1)
		return
	var instances := [
		{ "pid": 101, "scene": "res://game/Main.tscn" },
		{ "pid": 202, "scene": "res://game/Main.tscn" },
	]
	check(tool._select_target(instances, { "pid": 202 }, "", false).get("pid") == 202, "explicit PID targets an external runtime")
	check(tool._select_target(instances, { "pid": 202.0 }, "", false).get("pid") == 202, "JSON numeric PID targets an external runtime")
	check(tool._select_target([instances[0]], {}, "res://game/Main.tscn", true).get("pid") == 101, "single editor-play runtime remains the default")
	check(tool._select_target(instances, {}, "res://game/Main.tscn", true).has("error"), "multiple editor-play instances stay ambiguous")
	check(tool._select_target(instances, { "pid": 303 }, "", false).has("error"), "missing PID fails")
	check(tool._select_target(instances, { "pid": "oops" }, "", false).has("error"), "malformed PID fails")
	check(tool._select_target([instances[0]], {}, "", false).has("error"), "external runtime requires explicit PID")
	if not _failed:
		print("PASS: game target selection")
	quit(1 if _failed else 0)

var _failed := false

func check(value: bool, message: String) -> void:
	if not value:
		fail(message)

func fail(message: String) -> void:
	_failed = true
	push_error("FAIL: %s" % message)
