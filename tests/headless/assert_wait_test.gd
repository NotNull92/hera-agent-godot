extends SceneTree
const Inspector = preload("res://addons/hera_agent_godot/runtime/game_inspector.gd")
const GameTool = preload("res://addons/hera_agent_godot/tools/game_tool.gd")
var failed: bool = false
func _initialize() -> void:
	call_deferred("_test")
func _check(ok: bool, message: String) -> void:
	if not ok:
		failed = true
		push_error(message)
func _test() -> void:
	var inspector := Inspector.new()
	root.add_child(inspector)
	inspector.set_process(false)
	var target := Node2D.new()
	target.name = "Target"
	target.visible = false
	root.add_child(target)
	var request := {"id": "wait-test", "action": "assert", "path": "/root/Target", "prop": "visible", "op": "eq", "value": "true", "within_ms": 500}
	create_timer(0.03).timeout.connect(target.set.bind("visible", true))
	var tool := GameTool.new()
	_check(tool._write_request(request, OS.get_process_id()) == "", "write request")
	inspector._handle_file(tool._request_path(OS.get_process_id(), "wait-test"), tool._response_path(OS.get_process_id(), "wait-test"))
	var response: Dictionary = {}
	var deadline: int = Time.get_ticks_msec() + 1500
	while response.is_empty() and Time.get_ticks_msec() < deadline:
		await process_frame
		response = tool._read_response(OS.get_process_id(), "wait-test")
	_check(response.get("ok", false) and response.get("actual") == "true" and int(response.get("waited_ms", 0)) > 0, "file dispatch waits for later frame success")
	request["value"] = "false"
	request["within_ms"] = 30
	response = await inspector._assert_node_within(request)
	_check(not response.get("ok", true) and response.get("timed_out", false) and response.get("actual") == "true" and int(response.get("waited_ms", 0)) >= 30, "timeout includes last observation")
	for invalid: Variant in [0, -1, 3600001, 0.5, "100"]:
		request["within_ms"] = invalid
		response = await inspector._assert_node_within(request)
		_check(not response.get("ok", true), "invalid deadline rejected")
	request.erase("within_ms")
	request["value"] = "true"
	response = inspector._handle(request)
	_check(response.get("ok", false) and not response.has("waited_ms"), "default remains immediate")
	target.free()
	inspector.free()
	quit(1 if failed else 0)
