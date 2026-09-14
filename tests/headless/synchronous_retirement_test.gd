# Hera test: editor
extends SceneTree

const Queue = preload("res://addons/hera_agent_godot/server/work_queue.gd")

class Server extends RefCounted:
	func poll(_queue: RefCounted) -> void:
		pass

class RetiringPlugin extends "res://addons/hera_agent_godot/hera_agent_plugin.gd":
	var calls: Array[int] = [0]
	func _handle(_item: Dictionary) -> void:
		calls[0] += 1
		_queue.operations.retire()

func _initialize() -> void:
	var plugin := RetiringPlugin.new()
	var calls: Array[int] = plugin.calls
	plugin._server = Server.new()
	plugin._queue = Queue.new()
	plugin._queue.enqueue({"request": {"tool": "eval"}})
	plugin._queue.enqueue({"request": {"tool": "eval"}})
	plugin._process(0.01)
	plugin.free()
	if calls[0] != 1:
		push_error("retirement must stop remaining drained requests")
		quit(1)
		return
	print("synchronous_retirement_test: PASS")
	quit(0)
