# Hera test: editor
extends SceneTree
const NodeTool = preload("res://addons/hera_agent_godot/tools/node_tool.gd")
const Instancer = preload("res://addons/hera_agent_godot/tools/node_scene_instancer.gd")
var failed: bool = false
func _initialize() -> void:
	call_deferred("_test")
func _check(ok: bool, message: String) -> void:
	if not ok:
		failed = true
		push_error(message)
func _test() -> void:
	var scene := Node.new()
	root.add_child(scene)
	var template := Node2D.new()
	template.name = "Template"
	var packed := PackedScene.new()
	packed.pack(template)
	ResourceSaver.save(packed, "res://template.tscn")
	template.free()
	var plugin: EditorPlugin = ClassDB.instantiate("EditorPlugin")
	var manager: EditorUndoRedoManager = plugin.get_undo_redo()
	var tool := NodeTool.new()
	tool.set_undo_redo(manager)
	for action: String in ["add", "instance"]:
		var params := {"type": "Node2D", "scene": "res://template.tscn", "name": "Child", "props": [["visible", "false"], ["position", "Vector2(5, 6)"]]}
		var result: Dictionary = tool._add_node(scene, params) if action == "add" else Instancer.execute(scene, params, manager)
		_check(result.get("ok", false) and result.data.values.visible == "false", "creation returns re-read values")
		var child: Node2D = scene.get_node("Child")
		_check(child.position == Vector2(5, 6) and not child.visible, "properties applied")
		var history: UndoRedo = manager.get_history_undo_redo(manager.get_object_history_id(scene))
		history.undo()
		_check(not scene.has_node("Child"), "one undo removes configured node")
		history.redo()
		_check(scene.get_node("Child") == child and not child.visible, "redo restores configured node")
		history.undo()
		manager.get_history_undo_redo(manager.get_object_history_id(scene)).clear_history()
		for props: Variant in [{"visible": "not_bool"}, {"missing": "value"}, [["visible"]], [[1, "false"]], 42]:
			params["props"] = props
			result = tool._add_node(scene, params) if action == "add" else Instancer.execute(scene, params, manager)
			_check(not result.get("ok", false) and scene.get_child_count() == 0, "bad properties add no node")
	var range_template := Range.new()
	range_template.name = "RangeTemplate"
	packed.pack(range_template)
	ResourceSaver.save(packed, "res://range.tscn")
	range_template.free()
	for action: String in ["add", "instance"]:
		for duplicate: bool in [false, true]:
			var pairs: Array = [["value", "150"], ["max_value", "200"]]
			if duplicate:
				pairs.append(["value", "180"])
			var params := {"type": "Range", "scene": "res://range.tscn", "name": "Ordered", "props": pairs}
			var result: Dictionary = tool._add_node(scene, params) if action == "add" else Instancer.execute(scene, params, manager)
			_check(result.get("ok", false), "ordered creation succeeds")
			var ordered: Range = scene.get_node("Ordered")
			_check(ordered.value == (180.0 if duplicate else 100.0) and ordered.max_value == 200.0, "setters follow caller order; last duplicate wins")
			var history: UndoRedo = manager.get_history_undo_redo(manager.get_object_history_id(scene))
			history.undo()
			manager.get_history_undo_redo(manager.get_object_history_id(scene)).clear_history()
	manager.get_history_undo_redo(manager.get_object_history_id(scene)).clear_history()
	plugin.free()
	scene.free()
	quit(1 if failed else 0)
