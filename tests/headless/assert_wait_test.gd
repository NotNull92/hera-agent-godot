extends SceneTree
const Inspector = preload("res://addons/hera_agent_godot/runtime/game_inspector.gd")
const GameTool = preload("res://addons/hera_agent_godot/tools/game_tool.gd")
class ObservedInspector extends Inspector:
	var observations: int = 0
	var after_failure: Callable
	func _assert_node(request: Dictionary) -> Dictionary:
		var result: Dictionary = super._assert_node(request)
		observations += 1
		if not result.get("ok", false) and after_failure.is_valid():
			# Order mutations after an actual failed observation, independent of frame duration.
			after_failure.call_deferred()
			after_failure = Callable()
		return result

var failed: bool = false
func _initialize() -> void:
	call_deferred("_test")
func _check(ok: bool, message: String) -> void:
	if not ok:
		failed = true
		push_error(message)
func _test() -> void:
	var inspector := ObservedInspector.new()
	root.add_child(inspector)
	inspector.set_process(false)
	var target := Node2D.new()
	target.name = "Target"
	target.visible = false
	root.add_child(target)
	var request := {"id": "wait-test", "action": "assert", "path": "/root/Target", "prop": "visible", "op": "eq", "value": "true", "within_ms": 2000}
	inspector.after_failure = target.set.bind("visible", true)
	var tool := GameTool.new()
	_check(tool._write_request(request, OS.get_process_id()) == "", "write request")
	inspector._handle_file(tool._request_path(OS.get_process_id(), "wait-test"), tool._response_path(OS.get_process_id(), "wait-test"))
	var response: Dictionary = {}
	var deadline: int = Time.get_ticks_msec() + 4000
	while response.is_empty() and Time.get_ticks_msec() < deadline:
		await process_frame
		response = tool._read_response(OS.get_process_id(), "wait-test")
	_check(response.get("ok", false) and response.get("actual") == "true" and response.has("waited_ms") and inspector.observations >= 2, "file dispatch waits for later frame success: %s" % response)
	request["value"] = "false"
	request["within_ms"] = 30
	response = await inspector._assert_node_within(request)
	_check(not response.get("ok", true) and response.get("timed_out", false) and response.get("actual") == "true" and int(response.get("waited_ms", 0)) >= 30, "timeout includes last observation")
	request["prop"] = "missing_property"
	request["within_ms"] = 30000
	response = await inspector._assert_node_within(request)
	_check(not response.get("ok", true) and not response.has("timed_out") and int(response.get("waited_ms", 30000)) < 100, "missing property fails without yielding")
	var spawned := Node2D.new()
	spawned.name = "Spawned"
	request["path"] = "/root/Spawned"
	request["prop"] = "visible"
	request["value"] = "true"
	request["within_ms"] = 2000
	inspector.observations = 0
	inspector.after_failure = root.add_child.bind(spawned)
	response = await inspector._assert_node_within(request)
	_check(response.get("ok", false) and response.has("waited_ms") and inspector.observations >= 2, "missing node waits for spawn")
	spawned.free()
	request["path"] = "/root/Target"
	for invalid: Variant in [0, -1, 30001, 0.5, "100"]:
		request["within_ms"] = invalid
		response = await inspector._assert_node_within(request)
		_check(not response.get("ok", true), "invalid deadline rejected")
	request.erase("within_ms")
	request["value"] = "true"
	response = inspector._handle(request)
	_check(response.get("ok", false) and not response.has("waited_ms"), "default remains immediate")
	request["within_ms"] = 100
	request["value"] = "false"
	inspector.observations = 0
	inspector.after_failure = target.queue_free
	response = await inspector._assert_node_within(request)
	_check(response.get("timed_out", false) and inspector.observations >= 2 and not response.has("actual") and String(response.get("error", "")).begins_with("node not found:"), "timeout reports only the last frame after node removal")
	if is_instance_valid(target):
		target.free()
	inspector.free()
	quit(1 if failed else 0)
