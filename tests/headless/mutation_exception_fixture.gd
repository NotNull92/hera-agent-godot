extends SceneTree

const Queue = preload("res://addons/hera_agent_godot/server/work_queue.gd")
const Registry = preload("res://addons/hera_agent_godot/core/tool_registry.gd")
const Batch = preload("res://addons/hera_agent_godot/tools/batch_tool.gd")

class FaultTool extends RefCounted:
	func get_name() -> String:
		return "fault"
	func execute(_params: Dictionary) -> Variant:
		assert(false, "intentional gate exception")
		return null

func _initialize() -> void:
	_run.call_deferred()

func _run() -> void:
	var queue := Queue.new()
	queue.configure_operations("exception")
	var registry := Registry.new()
	registry.register(FaultTool.new())
	registry.register(Batch.new())
	var item := {"request": {"tool": "batch", "params": {"commands": [{"tool": "fault"}, {"tool": "fault"}]}}}
	if not queue.begin(item):
		quit(1)
		return
	var result: Dictionary = await queue.execute_request(item.request, registry, item.gate_owner)
	queue.complete(item, result)
	if not queue._owner.is_empty() or not result.data.stopped or result.data.count != 1 or not result.data.results[0].error.begins_with("outcome_unknown:"):
		quit(1)
		return
	var next := {"request": {"tool": "fault"}}
	if not queue.begin(next):
		quit(1)
		return
	queue.complete(next, {"ok": false, "error": "fixture failure"})
	print("MUTATION_EXCEPTION_PASS")
	quit(0)
