extends SceneTree
# Hera test: editor

const NodeTool = preload("res://addons/hera_agent_godot/tools/node_tool.gd")
var passed: bool = true

func _initialize() -> void:
	call_deferred("_test")

func _check(condition: bool, message: String) -> void:
	if not condition:
		passed = false
		push_error(message)

func _test() -> void:
	var scene := Node.new()
	scene.name = "Scene"
	root.add_child(scene)
	var branch := Node.new()
	branch.name = "Branch"
	scene.add_child(branch)
	branch.owner = scene
	var leaf := Node.new()
	leaf.name = "Leaf"
	branch.add_child(leaf)
	leaf.owner = scene
	var nested := Node.new()
	nested.name = "Nested"
	branch.add_child(nested)
	nested.owner = branch
	var outside := Node.new()
	outside.name = "Outside"
	root.add_child(outside)
	var plugin := EditorPlugin.new()
	var manager: EditorUndoRedoManager = plugin.get_undo_redo()
	var tool := NodeTool.new()
	tool.set_undo_redo(manager)
	var result: Dictionary = tool._remove_node(scene, {"path": "Branch"})
	_check(bool(result.get("ok", false)), "remove failed")
	var history: UndoRedo = manager.get_history_undo_redo(manager.get_object_history_id(scene))
	for cycle in range(2):
		history.undo()
		_check(branch.get_parent() == scene and branch.owner == scene and leaf.owner == scene, "undo lost subtree ownership")
		_check(nested.owner == branch, "undo flattened nested ownership")
		var packed := PackedScene.new()
		_check(packed.pack(scene) == OK, "pack failed")
		_check(ResourceSaver.save(packed, "res://remove_undo_fixture.tscn") == OK, "save failed")
		var saved: PackedScene = ResourceLoader.load("res://remove_undo_fixture.tscn", "", ResourceLoader.CACHE_MODE_IGNORE)
		var restored := saved.instantiate()
		_check(restored.has_node("Branch/Leaf"), "saved scene lost descendant")
		restored.free()
		if cycle == 0:
			history.redo()
			_check(branch.get_parent() == null, "redo did not remove branch")
	manager.clear_history()
	for path in [".", "Branch/..", String(scene.get_path()), "..", "/root", "../Outside", "/root/Outside"]:
		result = tool._remove_node(scene, {"path": path})
		_check(not bool(result.get("ok", false)), "accepted root/outside remove: " + path)
		if bool(result.get("ok", false)):
			manager.get_history_undo_redo(manager.get_object_history_id(root)).undo()
		manager.clear_history()
	for path in ["Branch/..", String(scene.get_path()), "../Outside"]:
		result = tool._reparent_node(scene, {"path": path, "parent": "Branch"})
		_check(not bool(result.get("ok", false)), "accepted root/outside reparent: " + path)
	result = tool._reparent_node(scene, {"path": "Branch", "parent": "../Outside"})
	_check(not bool(result.get("ok", false)), "accepted out-of-scene parent")
	_check(tool._resolve(scene, "..") == null, "resolver escaped scene")
	_check(tool._resolve(scene, "Branch") == branch, "resolver rejected child")
	manager.clear_history()
	plugin.free()
	outside.free()
	scene.free()
	DirAccess.remove_absolute(ProjectSettings.globalize_path("res://remove_undo_fixture.tscn"))
	print("node remove undo: ", "PASS" if passed else "FAIL")
	quit(0 if passed else 1)
