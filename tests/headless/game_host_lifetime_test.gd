# Hera test: editor
extends SceneTree

const Game = preload("res://addons/hera_agent_godot/tools/game_tool.gd")
class FailedAssertion:
	extends Game
	func _read_response(_game_pid: int, _request_id: String) -> Dictionary:
		return {"ok": false, "id": "fixture", "error": "assert failed", "actual": "false", "waited_ms": 100, "timed_out": true}

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
	host = Node.new()
	root.add_child(host)
	var failing := FailedAssertion.new()
	failing.set_host(host)
	response = await failing.execute_async({"action": "assert", "pid": 424242, "within_ms": 100})
	if response.get("ok", true) or response.get("error") != "assert failed" or response.get("data", {}).has("error") or not response.get("data", {}).get("timed_out", false):
		push_error("assertion failure keeps the outer error without duplicating it in data")
		quit(1)
		return
	host.free()
	print("game_host_lifetime_test: PASS")
	quit(0)

func _invoke(tool: RefCounted) -> void:
	response = await tool.execute_async({"action": "call", "pid": 424242, "path": "/root/Main", "method": "increment"})
