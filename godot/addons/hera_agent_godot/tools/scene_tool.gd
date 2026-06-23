extends RefCounted

# `scene` — read and manage scenes via EditorInterface.
#   tree         -> flat node list of the edited scene { path, type, name }
#   open_scenes  -> open scene paths + the current one
#   open         -> open a scene by path
#   save         -> save the edited scene

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
		"open":
			return _open(params)
		"save":
			return _save()
		_:
			return ToolResponse.failure("unknown scene action: %s (want tree|open_scenes|open|save)" % action)

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

func _open(params: Dictionary) -> Dictionary:
	var path := String(params.get("path", ""))
	if path == "":
		return ToolResponse.failure("open requires a 'path'")
	if not ResourceLoader.exists(path, "PackedScene"):
		return ToolResponse.failure("scene not found or not a PackedScene: %s" % path)
	EditorInterface.open_scene_from_path(path)
	return ToolResponse.success({ "requested_open": path, "current": _current_scene() })

func _save() -> Dictionary:
	var root := EditorInterface.get_edited_scene_root()
	if root == null:
		return ToolResponse.failure("no scene to save")
	if root.scene_file_path == "":
		return ToolResponse.failure("scene has no path yet; save it once from the editor first")
	var err := EditorInterface.save_scene()
	if err != OK:
		return ToolResponse.failure("save failed: %s" % error_string(err))
	return ToolResponse.success({ "saved": root.scene_file_path })

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
