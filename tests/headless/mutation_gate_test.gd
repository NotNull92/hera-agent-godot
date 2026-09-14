extends SceneTree

const WorkQueue = preload("res://addons/hera_agent_godot/server/work_queue.gd")
const Registry = preload("res://addons/hera_agent_godot/core/tool_registry.gd")
const Batch = preload("res://addons/hera_agent_godot/tools/batch_tool.gd")
var failed := false
var finished: Dictionary = {}

class HeldTool extends RefCounted:
	signal released
	var calls := 0
	var result: Variant = {"ok": true, "data": {}}
	func get_name() -> String:
		return "held"
	func execute_async(_params: Dictionary) -> Variant:
		calls += 1
		await released
		return result

class CounterTool extends RefCounted:
	var calls := 0
	func get_name() -> String:
		return "counter"
	func execute(_params: Dictionary) -> Dictionary:
		calls += 1
		return {"ok": true, "data": {}}

class FinishingFilesystem extends Node:
	signal filesystem_changed
	signal sources_changed(changed: bool)
	func is_scanning() -> bool:
		return false

func _initialize() -> void:
	_run.call_deferred()

func _run() -> void:
	var queue := WorkQueue.new()
	queue.configure_operations("session")
	var first := {"request": {"tool": "project", "params": {"action": "scan"}}}
	_check(queue.begin(first), "first mutation starts")
	var collision := {"request": {"tool": "project", "params": {"action": "reimport"}}}
	_check(not queue.begin(collision), "reentrant mutation must be rejected while scan owns gate")
	_check(String(collision.get("response", {}).get("error", "")).begins_with("editor_busy:"), "collision reports editor_busy")
	queue.complete(first, {"ok": false, "error": "fixture failure"})
	for request: Dictionary in [
		{"tool": "node", "params": {"action": "get"}},
		{"tool": "scene", "params": {"action": "tree"}},
		{"tool": "resource", "params": {"action": "get"}},
		{"tool": "output", "params": {"source": "file"}},
	]:
		first = {"request": {"tool": "batch"}}
		_check(queue.begin(first), "failure releases gate")
		_check(not queue.begin({"request": request}), "sensitive read cannot overlap")
		for safe: Dictionary in [{"tool": "status"}, {"tool": "output", "params": {"source": "editor"}}, {"tool": "diagnostics", "params": {"source": "editor"}}]:
			var read := {"request": safe}
			_check(queue.begin(read), "independent buffered read remains admitted")
			queue.complete(read, {"ok": true})
		_check(not queue.begin({"request": request}), "safe read completion cannot release owner")
		queue.complete(first, {"ok": true})
	var registry := Registry.new()
	var held := HeldTool.new()
	var counter := CounterTool.new()
	var batch := Batch.new()
	registry.register(held)
	registry.register(counter)
	registry.register(batch)
	var commands := [{"tool": "held"}, {"tool": "counter"}, {"tool": "batch"}]
	_run_item(queue, registry, {"tool": "batch", "params": {"commands": commands}})
	await process_frame
	_check(finished.is_empty() and held.calls == 1, "async batch retains lifetime")
	_check(not queue.begin({"request": {"tool": "counter", "params": {"gate_owner": queue._owner}}}), "wire params cannot borrow parent ownership")
	var independent := WorkQueue.new()
	independent.configure_operations("other-project")
	var other := {"request": {"tool": "counter"}}
	_check(independent.begin(other), "independent project session remains runnable")
	independent.complete(other, {"ok": true})
	held.released.emit()
	_check(counter.calls == 1 and finished.data.stopped and finished.data.results[2].error == "batch cannot nest batch", "owned children execute once and recursion stops")
	_check(queue._owner.is_empty(), "batch recursion failure releases ownership")
	for invalid_result: Variant in [null, {"ok": false, "error": "target lost"}]:
		finished = {}
		held.result = invalid_result
		_run_item(queue, registry, {"tool": "held"})
		held.released.emit()
		_check(not finished.ok and queue._owner.is_empty(), "failed or missing async result releases ownership")
	var now := int(Time.get_unix_time_from_system() * 1000)
	var id := "session:%d:held" % (now + 30000)
	var input := {"tool": "game", "params": {"action": "call", "pid": 42, "runtime_session_id": "runtime", "method": "increment"}}
	var raw := JSON.stringify(input)
	first = {"request": {"tool": "batch"}}
	queue.begin(first)
	queue.enqueue({"request": {"tool": "operation", "params": {"action": "submit", "id": id, "request": raw, "digest": raw.sha256_text()}}})
	var wrapped: Dictionary = queue.drain()[0]
	_check(not queue.begin(wrapped) and wrapped.response.data.error_code == "editor_busy" and wrapped.response.data.effect == "not_applied", "receipt reports definite collision rejection")
	queue.enqueue({"request": {"tool": "operation", "params": {"action": "status", "id": id}}})
	_check(queue.drain()[0].response.data.error_code == "editor_busy", "operation status available during ownership")
	queue.complete(first, {"ok": true})
	var cancel_id := "session:%d:cancel" % (now + 30000)
	queue.enqueue({"request": {"tool": "operation", "params": {"action": "submit", "id": cancel_id, "request": raw, "digest": raw.sha256_text()}}})
	wrapped = queue.drain()[0]
	queue.operations.cancel(cancel_id)
	_check(not queue.begin(wrapped) and queue._owner.is_empty(), "pre-dispatch cancellation acquires no gate")
	finished = {}
	_run_item(queue, registry, {"tool": "batch", "params": {"commands": commands, "stop_on_error": false}})
	queue.retire()
	_check(queue._owner.is_empty(), "plugin teardown immediately retires ownership")
	held.released.emit()
	_check(counter.calls == 1 and not finished.ok, "teardown prevents later batch children even with continue")
	_check(not queue.begin({"request": {"tool": "counter"}}), "retired session cannot restart work")
	var fs_queue := WorkQueue.new()
	var fs := FinishingFilesystem.new()
	root.add_child(fs)
	fs_queue.set_filesystem(fs)
	fs.set_process(true)
	_check(fs_queue._import_busy(), "pending scan processing is busy")
	fs.filesystem_changed.emit()
	_check(fs_queue._import_busy(), "completion event does not prove still-processing filesystem idle")
	fs.set_process(false)
	_check(not fs_queue._import_busy(), "completion before processing stops cannot leak pending gate")
	fs_queue.retire()
	fs.free()
	print("mutation_gate_test: ", "FAIL" if failed else "PASS")
	quit(1 if failed else 0)

func _run_item(queue: RefCounted, registry: RefCounted, request: Dictionary) -> void:
	var item := {"request": request}
	if not queue.begin(item):
		finished = item.response
		return
	var result: Dictionary = await queue.execute_request(request, registry, item.get("gate_owner", ""))
	finished = queue.complete(item, result)

func _check(value: bool, message: String) -> void:
	if not value:
		failed = true
		push_error(message)
