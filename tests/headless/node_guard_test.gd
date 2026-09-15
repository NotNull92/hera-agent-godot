# Hera test: editor
extends SceneTree

const NodeTool = preload("res://addons/hera_agent_godot/tools/node_tool.gd")
const Records = preload("res://addons/hera_agent_godot/core/operation_records.gd")
var passed: bool = true

func _initialize() -> void:
	call_deferred("_test")

func _check(condition: bool, message: String) -> void:
	if not condition:
		passed = false
		push_error(message)

func _test() -> void:
	var scene := Node2D.new()
	scene.name = "GuardScene"
	var packed := PackedScene.new()
	packed.pack(scene)
	ResourceSaver.save(packed, "res://guard_scene.tscn")
	scene.free()
	EditorInterface.open_scene_from_path("res://guard_scene.tscn")
	await process_frame
	await process_frame
	scene = EditorInterface.get_edited_scene_root() as Node2D
	var plugin := EditorPlugin.new()
	var manager: EditorUndoRedoManager = plugin.get_undo_redo()
	var tool := NodeTool.new()
	tool.set_undo_redo(manager)
	tool.set("editor_session_id", "session")
	var expected := {"editor_session_id": "old-session", "scene": scene.scene_file_path, "node_instance_id": str(scene.get_instance_id()), "prop": "visible", "type": "bool", "value": "true"}
	var result: Dictionary = tool.execute({"action": "set", "path": ".", "prop": "visible", "value": "false", "expected": expected, "verify": true})
	_check(not result.get("ok", false) and String(result.get("error", "")).begins_with("session_mismatch:"), "stale session must reject before mutation")
	_check(result.get("attempted") == false and result.get("error_code") == "session_mismatch", "pre-set session mismatch must be a rejected attempt")
	var records := Records.new()
	records.session = "session"
	var now := int(Time.get_unix_time_from_system() * 1000)
	var id := "session:%d:guard" % (now + 30000)
	var input := {"tool": "node", "params": {"action": "set_guarded", "path": ".", "prop": "visible", "value": "false", "expected": expected}}
	_check(records.accept(id, input).has("accepted"), "guarded set must be an admissible operation")
	_check(records.begin(id), "guarded rejection fixture begins")
	records.finish(id, result)
	var receipt: Dictionary = records.lookup(id)
	_check(receipt.lifecycle == "rejected" and receipt.effect == "not_applied" and receipt.error_code == "session_mismatch", "guarded pre-set conflict must not become outcome_unknown")
	var read: Dictionary = tool.execute({"action": "get", "path": ".", "prop": "visible"}).get("data", {})
	_check(read.get("properties", {}).get("visible") == "true", "conflict changed property")
	var history: UndoRedo = manager.get_history_undo_redo(manager.get_object_history_id(scene))
	_check(not history.has_undo(), "conflict registered undo")
	for field in ["scene", "node_instance_id", "prop", "type", "value"]:
		var mismatch: Dictionary = expected.duplicate()
		mismatch["editor_session_id"] = "session"
		mismatch[field] = "false" if field == "value" else "other"
		result = tool.execute({"action": "set", "path": ".", "prop": "visible", "value": "false", "expected": mismatch})
		_check(not result.get("ok", false) and result.get("attempted") == false and result.get("error_code") != "outcome_unknown", "mismatched " + field + " lost attempted=false")
		read = tool.execute({"action": "get", "path": ".", "prop": "visible"}).get("data", {})
		_check(read.get("properties", {}).get("visible") == "true" and not history.has_undo(), "conflict had side effects: " + field)
	expected["editor_session_id"] = "session"
	result = tool.execute({"action": "set", "path": ".", "prop": "visible", "value": "false", "expected": expected, "verify": true})
	_check(result.get("ok", false) and result.get("data", {}).get("verification") == "passed", "valid guarded mutation did not verify")
	_check(not scene.visible, "setter did not apply")
	history.undo()
	_check(scene.visible, "guarded mutation lost undo")
	manager.clear_history()
	read = tool.execute({"action": "get", "path": ".", "prop": "position", "snapshot": true}).get("data", {})
	var snapshot: Dictionary = read.get("expected", {})
	_check(snapshot.get("node_instance_id") is String and snapshot.get("type") == "Vector2", "snapshot lacks typed identity")
	result = tool.execute({"action": "set", "path": ".", "prop": "position", "value": "Vector2(5, 10)", "expected": snapshot, "verify": true})
	_check(result.get("ok", false) and scene.position == Vector2(5, 10), "typed Vector2 guard failed")
	manager.clear_history()
	var child := Node2D.new()
	child.name = "Target"
	scene.add_child(child)
	read = tool.execute({"action": "get", "path": "Target", "prop": "visible", "snapshot": true}).get("data", {})
	child.free()
	child = Node2D.new()
	child.name = "Target"
	scene.add_child(child)
	result = tool.execute({"action": "set", "path": "Target", "prop": "visible", "value": "false", "expected": read.get("expected", {})})
	_check(not result.get("ok", false) and String(result.get("error", "")).begins_with("state_conflict:"), "same-path replacement was mutated")
	_check(child.visible and not history.has_undo(), "replacement conflict had side effects")
	for malformed in [null, {}, {"editor_session_id": 3}, expected.merged({"extra": "ignored"})]:
		result = tool.execute({"action": "set", "path": ".", "prop": "visible", "value": "false", "expected": malformed})
		_check(not result.get("ok", false) and scene.visible and not history.has_undo(), "malformed expected bypassed guard")
	var custom := Node2D.new()
	custom.set_script(load("res://tests/headless/node_guard_fixture.gd"))
	custom.name = "Custom"
	scene.add_child(custom)
	for prop in ["amount", "integer", "text", "number"]:
		read = tool.execute({"action": "get", "path": "Custom", "prop": prop, "snapshot": true}).get("data", {})
		snapshot = read.get("expected", {})
		_check(not snapshot.is_empty(), "snapshot cannot roundtrip " + prop)
		result = tool.execute({"action": "set", "path": "Custom", "prop": prop, "value": snapshot.get("value"), "expected": snapshot, "verify": true})
		_check(result.get("ok", false), "typed snapshot failed for " + prop)
	read = tool.execute({"action": "get", "path": "Custom", "prop": "amount", "snapshot": true}).get("data", {})
	result = tool.execute({"action": "set", "path": "Custom", "prop": "amount", "value": "99", "expected": read.get("expected"), "verify": true})
	_check(String(result.get("error", "")).begins_with("verification_failed:"), "clamping setter was falsely verified")
	read = tool.execute({"action": "get", "path": "Custom", "prop": "amount"}).get("data", {})
	_check(read.get("properties", {}).get("amount") == "10.0", "failed verification incorrectly rolled back setter")
	read = tool.execute({"action": "get", "path": "Custom", "prop": "retire", "snapshot": true}).get("data", {})
	result = tool.execute({"action": "set", "path": "Custom", "prop": "retire", "value": "true", "expected": read.get("expected"), "verify": true})
	_check(String(result.get("error", "")).begins_with("verification_unavailable:"), "retiring target was falsely verified")
	manager.clear_history()
	read = tool.execute({"action": "get", "path": ".", "prop": "visible", "snapshot": true}).get("data", {})
	ResourceSaver.save(packed, "res://other_scene.tscn")
	EditorInterface.open_scene_from_path("res://other_scene.tscn")
	await process_frame
	await process_frame
	result = tool.execute({"action": "set", "path": ".", "prop": "visible", "value": "false", "expected": read.get("expected")})
	_check(String(result.get("error", "")).begins_with("state_conflict:"), "scene switch bypassed guard")
	read = tool.execute({"action": "get", "path": ".", "prop": "visible"}).get("data", {})
	_check(read.get("properties", {}).get("visible") == "true", "scene switch conflict changed new scene")
	read = tool.execute({"action": "get", "path": ".", "prop": "scene_file_path", "snapshot": true}).get("data", {})
	result = tool.execute({"action": "set", "path": ".", "prop": "scene_file_path", "value": "res://renamed.tscn", "expected": read.get("expected"), "verify": true})
	_check(String(result.get("error", "")).begins_with("verification_unavailable:"), "changed scene identity was falsely verified")
	var lifecycle: EditorPlugin = load("res://addons/hera_agent_godot/hera_agent_plugin.gd").new()
	var registry: RefCounted = load("res://addons/hera_agent_godot/core/tool_registry.gd").new()
	registry.call("register", tool)
	lifecycle.set("_registry", registry)
	lifecycle.call("_exit_tree")
	_check(tool.editor_session_id == "", "retired addon left an active node mutation session")
	lifecycle.free()
	manager.clear_history()
	plugin.free()
	print("node guard: ", "PASS" if passed else "FAIL")
	quit(0 if passed else 1)
