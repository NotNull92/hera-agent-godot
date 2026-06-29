extends RefCounted

const ToolResponse = preload("res://addons/hera_agent_godot/core/tool_response.gd")

static func execute(root: Node, params: Dictionary, undo_redo: Variant) -> Dictionary:
	var scene_path := String(params.get("scene", ""))
	if scene_path == "":
		return ToolResponse.failure("instance requires a scene path")
	if not scene_path.begins_with("res://") or not _is_safe_res_path(scene_path):
		return ToolResponse.failure("scene path must be a safe res:// path")
	if not (scene_path.ends_with(".tscn") or scene_path.ends_with(".scn")):
		return ToolResponse.failure("scene path must end with .tscn or .scn")
	if not ResourceLoader.exists(scene_path, "PackedScene"):
		return ToolResponse.failure("packed scene not found: %s" % scene_path)
	var loaded := ResourceLoader.load(scene_path, "PackedScene")
	if loaded == null or not (loaded is PackedScene):
		return ToolResponse.failure("not a PackedScene: %s" % scene_path)
	var parent_path := String(params.get("parent", "."))
	var parent := _resolve(root, parent_path)
	if parent == null:
		return ToolResponse.failure("parent not found: %s" % parent_path)
	var packed_scene: PackedScene = loaded as PackedScene
	var instanced := packed_scene.instantiate()
	if instanced == null or not (instanced is Node):
		return ToolResponse.failure("PackedScene root is not a Node: %s" % scene_path)
	var node: Node = instanced as Node
	var requested_name := String(params.get("name", ""))
	if requested_name != "":
		node.name = requested_name

	if undo_redo != null:
		undo_redo.create_action("Hera: instance %s" % scene_path)
		undo_redo.add_do_method(parent, "add_child", node)
		undo_redo.add_do_method(node, "set_owner", root)
		undo_redo.add_do_reference(node)
		undo_redo.add_undo_method(parent, "remove_child", node)
		undo_redo.commit_action()
	else:
		parent.add_child(node)
		node.set_owner(root)

	return ToolResponse.success({
		"instanced": String(root.get_path_to(node)),
		"scene": scene_path,
		"type": node.get_class(),
		"name": String(node.name),
	})

static func _resolve(root: Node, path: String) -> Node:
	return root if path == "." else root.get_node_or_null(path)

static func _is_safe_res_path(path: String) -> bool:
	if path.find("\\") != -1:
		return false
	var rel := path.substr("res://".length())
	if rel == "" or rel.begins_with("/"):
		return false
	for part in rel.split("/", true):
		if part == "" or part == "." or part == "..":
			return false
	return true
