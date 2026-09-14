extends SceneTree

const Inspector = preload("res://addons/hera_agent_godot/runtime/game_inspector.gd")
const GameTool = preload("res://addons/hera_agent_godot/tools/game_tool.gd")
var failed := false

class Counter extends Node:
	var count := 0
	func increment(delta: int = 1) -> int:
		count += delta
		return count

func _initialize() -> void:
	_run.call_deferred()

func _run() -> void:
	var inspector := Inspector.new()
	root.add_child(inspector)
	inspector.set_process(false)
	var counter := Counter.new()
	counter.name = "Counter"
	root.add_child(counter)
	var tool := GameTool.new()
	var id := "editor:%d:counter" % (int(Time.get_unix_time_from_system() * 1000) + 30000)
	var request := {"id": "first", "operation_id": id, "target_pid": OS.get_process_id(), "runtime_session_id": inspector.runtime_session_id, "action": "call", "path": "/root/Counter", "method": "increment", "args": [1]}
	await _deliver(tool, inspector, request)
	_check(counter.count == 1, "first mutation increments counter")
	# Fault injection stays in the test: erase only the reply, after the effect.
	DirAccess.remove_absolute(ProjectSettings.globalize_path(inspector._response_dir() + "/first.json"))
	request.id = "retry"
	await _deliver(tool, inspector, request)
	var response := tool._read_response(OS.get_process_id(), "retry")
	_check(counter.count == 1, "lost response same-ID retry cannot increment twice")
	_check(response.runtime_receipt.effect == "applied", "runtime preserves applied receipt")
	request.args = [2]
	request.id = "conflict"
	await _deliver(tool, inspector, request)
	response = tool._read_response(OS.get_process_id(), "conflict")
	_check(response.error.begins_with("operation_id_conflict:") and counter.count == 1, "different delta conflicts without effect")
	request.runtime_session_id = "previous-runtime"
	request.id = "old"
	await _deliver(tool, inspector, request)
	response = tool._read_response(OS.get_process_id(), "old")
	_check(response.error.begins_with("session_mismatch:") and counter.count == 1, "runtime replacement rejects old request")
	request.runtime_session_id = inspector.runtime_session_id
	request.operation_id = "editor:1:expired"
	request.id = "expired"
	await _deliver(tool, inspector, request)
	response = tool._read_response(OS.get_process_id(), "expired")
	_check(response.error.begins_with("operation_expired:") and counter.count == 1, "late file is consumed without effect")
	inspector.free()
	counter.free()
	print("runtime_operations_test: ", "FAIL" if failed else "PASS")
	quit(1 if failed else 0)

func _deliver(tool: RefCounted, inspector: Node, request: Dictionary) -> void:
	_check(tool._write_request(request, OS.get_process_id()) == "", "atomic request publish")
	await inspector._handle_file(inspector._request_dir() + "/%s.json" % request.id, inspector._response_dir() + "/%s.json" % request.id)

func _check(value: bool, message: String) -> void:
	if not value:
		failed = true
		push_error(message)
