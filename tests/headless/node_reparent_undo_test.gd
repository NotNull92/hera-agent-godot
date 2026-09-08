extends SceneTree
# Hera test: editor

const NodeTool = preload("res://addons/hera_agent_godot/tools/node_tool.gd")


func _initialize() -> void:
	call_deferred("_test")


func _test() -> void:
	var passed := _test_ownership(false)
	passed = _test_ownership(true) and passed
	print("reparent undo: ", "PASS" if passed else "FAIL")
	quit(0 if passed else 1)


func _test_ownership(owner_is_parent: bool) -> bool:
	var scene := Node2D.new()
	root.add_child(scene)
	var left := Node2D.new()
	left.name = "Left"
	scene.add_child(left)
	left.owner = scene
	var right := Node2D.new()
	right.name = "Right"
	right.position = Vector2(50, 20)
	scene.add_child(right)
	right.owner = scene
	var item := Node2D.new()
	item.name = "Item"
	left.add_child(item)
	item.owner = left if owner_is_parent else scene
	var descendant := Node2D.new()
	descendant.name = "Descendant"
	item.add_child(descendant)
	descendant.owner = item.owner
	var nested := Node2D.new()
	item.add_child(nested)
	nested.owner = item
	item.position = Vector2(5, 10)
	var sibling := Node2D.new()
	left.add_child(sibling)
	var collision := Node2D.new()
	collision.name = "Item"
	right.add_child(collision)
	var plugin := EditorPlugin.new()
	var manager: EditorUndoRedoManager = plugin.get_undo_redo()
	var tool := NodeTool.new()
	tool.set_undo_redo(manager)
	var result: Dictionary = tool._reparent_node(scene, {"path": "Left/Item", "parent": "Right"})
	var history: UndoRedo = manager.get_history_undo_redo(manager.get_object_history_id(item))
	var passed: bool = bool(result.get("ok", false)) and item.get_parent() == right
	passed = passed and item.global_position.is_equal_approx(Vector2(5, 10))
	history.undo()
	passed = passed and item.get_parent() == left and item.name == &"Item"
	passed = passed and item.get_index() == 0 and item.owner == (left if owner_is_parent else scene)
	passed = passed and item.position.is_equal_approx(Vector2(5, 10))
	passed = passed and descendant.owner == item.owner and nested.owner == item
	var saved := PackedScene.new()
	passed = (saved.pack(left if owner_is_parent else scene) == OK) and passed
	var reopened := saved.instantiate()
	passed = passed and reopened.has_node("Item/Descendant" if owner_is_parent else "Left/Item/Descendant")
	reopened.free()
	history.redo()
	passed = passed and item.get_parent() == right
	history.undo()
	passed = passed and item.name == &"Item" and item.get_index() == 0
	passed = passed and descendant.owner == item.owner and nested.owner == item
	manager.clear_history()
	plugin.free()
	scene.free()
	if not passed:
		push_error("FAIL: reparent undo must restore name, parent, index, owner and transform after a name collision")
	return passed
