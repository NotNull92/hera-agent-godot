extends RefCounted

# `scene` — read scene structure via EditorInterface.
#   tree         -> flat node list of the edited scene { path, type, name }
#   open_scenes  -> open scene paths + the current one

const ToolResponse = preload("res://addons/hera_agent_godot/core/tool_response.gd")

const MAX_NODES := 1000

func get_name() -> String:
	return "scene"

func execute(params: Dictionary) -> Dictionary:
	var action := String(params.get("action", "tree"))
	match action:
		"tree":
			return _tree()
		"open_scenes":
			return ToolResponse.success({
				"open": _to_strings(EditorInterface.get_open_scenes()),
				"current": _current_scene(),
			})
		_:
			return ToolResponse.failure("unknown scene action: %s (want tree|open_scenes)" % action)

func _tree() -> Dictionary:
	var root := EditorInterface.get_edited_scene_root()
	if root == null:
		return ToolResponse.success({ "scene": "", "count": 0, "truncated": false, "nodes": [] })
	var nodes: Array = []
	_collect(root, root, nodes)
	var truncated := nodes.size() > MAX_NODES
	if truncated:
		nodes = nodes.slice(0, MAX_NODES)
	return ToolResponse.success({
		"scene": root.scene_file_path,
		"count": nodes.size(),
		"truncated": truncated,
		"nodes": nodes,
	})

func _collect(node: Node, root: Node, out: Array) -> void:
	if out.size() > MAX_NODES:
		return
	out.append({
		"path": String(root.get_path_to(node)),
		"type": node.get_class(),
		"name": String(node.name),
	})
	for child in node.get_children():
		_collect(child, root, out)
		if out.size() > MAX_NODES:
			return

func _current_scene() -> String:
	var root := EditorInterface.get_edited_scene_root()
	return root.scene_file_path if root != null else ""

func _to_strings(values) -> Array:
	var result: Array = []
	for v in values:
		result.append(String(v))
	return result
