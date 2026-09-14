# Hera test: editor
extends SceneTree

const Game = preload("res://addons/hera_agent_godot/tools/game_tool.gd")
var response: Variant

func _initialize() -> void:
	_run.call_deferred()

func _run() -> void:
	var host := Node.new()
	root.add_child(host)
	var tool := Game.new()
	tool.set_host(host)
	DirAccess.make_dir_recursive_absolute("user://hera_game_instances")
	var file := FileAccess.open("user://hera_game_instances/424242.json", FileAccess.WRITE)
	file.store_string(JSON.stringify({"pid": 424242, "scene": "res://Main.tscn", "ts": Time.get_unix_time_from_system()}))
	file.close()
	_invoke(tool)
	host.free()
	var deadline := Time.get_ticks_msec() + 1000
	while response == null and Time.get_ticks_msec() < deadline:
		await process_frame
	if not response is Dictionary or not String(response.get("error", "")).begins_with("outcome_unknown:"):
		push_error("freed host must terminate polling with unknown dispatched effect")
		quit(1)
		return
	response = await tool.execute_async({"action": "instances"})
	if response.ok:
		push_error("retired host cannot accept another request")
		quit(1)
		return
	print("game_host_lifetime_test: PASS")
	quit(0)

func _invoke(tool: RefCounted) -> void:
	response = await tool.execute_async({"action": "call", "pid": 424242, "path": "/root/Main", "method": "increment"})
