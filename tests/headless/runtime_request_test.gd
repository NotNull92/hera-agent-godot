extends SceneTree

const Inspector = preload("res://addons/hera_agent_godot/runtime/game_inspector.gd")
const GameTool = preload("res://addons/hera_agent_godot/tools/game_tool.gd")
var _failed := false

class Target extends Node:
	var calls := 0
	func increment() -> int:
		calls += 1
		return calls

func _initialize() -> void:
	call_deferred("_run")

func _run() -> void:
	var inspector := Inspector.new()
	root.add_child(inspector)
	inspector.set_process(false)
	var target := Target.new()
	target.name = "RequestTarget"
	root.add_child(target)
	var request_path := inspector._request_dir() + "/partial.json"
	var response_path := inspector._response_dir() + "/partial.json"
	var request := {"id": "partial", "target_pid": OS.get_process_id(), "action": "call", "path": String(target.get_path()), "method": "increment"}
	_write(request_path + ".tmp", "{\"id\":")
	inspector._process(0.0)
	_check(FileAccess.file_exists(request_path + ".tmp") and target.calls == 0, "poll ignores unfinished temporary publication")
	DirAccess.remove_absolute(ProjectSettings.globalize_path(request_path + ".tmp"))
	_write(request_path, "{\"id\":")
	await inspector._handle_file(request_path, response_path)
	_check(FileAccess.file_exists(request_path), "partial request survives a poll")
	_check(target.calls == 0, "partial request does not execute")
	_write(request_path, JSON.stringify(request))
	await inspector._handle_file(request_path, response_path)
	await inspector._handle_file(request_path, response_path)
	_check(target.calls == 1, "completed request executes once")
	var tool := GameTool.new()
	_check(tool._read_response(OS.get_process_id(), "partial").get("id") == "partial", "response keeps request identity")
	request.id = "published"
	_check(tool._write_request(request, OS.get_process_id()) == "", "request publication succeeds")
	await inspector._handle_file(inspector._request_dir() + "/published.json", inspector._response_dir() + "/published.json")
	_check(target.calls == 2, "published request executes")
	tool._read_response(OS.get_process_id(), "published")
	request.target_pid = OS.get_process_id() + 1
	_write(request_path, JSON.stringify(request))
	await inspector._handle_file(request_path, response_path)
	_check(target.calls == 2 and not FileAccess.file_exists(response_path), "wrong PID never executes or responds")
	request.target_pid = OS.get_process_id()
	request.method = "free"
	request.id = "partial"
	_write(request_path, JSON.stringify(request))
	await inspector._handle_file(request_path, response_path)
	var response: Dictionary = tool._read_response(OS.get_process_id(), "partial")
	_check(bool(response.get("ok", false)) and response.get("path") == "/root/RequestTarget", "self-free call retains response path")
	inspector.free()
	print("runtime_request_test: ", "FAIL" if _failed else "PASS")
	quit(1 if _failed else 0)

func _write(path: String, content: String) -> void:
	var file := FileAccess.open(path, FileAccess.WRITE)
	file.store_string(content)
	file.close()

func _check(value: bool, message: String) -> void:
	if not value:
		_failed = true
		push_error(message)
