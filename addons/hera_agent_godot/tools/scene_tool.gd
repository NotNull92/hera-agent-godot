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
				"open": _to_unique_strings(EditorInterface.get_open_scenes()),
				"current": _current_scene(),
			})
		"open":
			return _open(params)
		"save":
			return _save()
		"create":
			return _create(params)
		"save_as":
			return _save_as(params)
		_:
			return ToolResponse.failure("unknown scene action: %s (want tree|open_scenes|open|save|create|save_as)" % action)

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

func _create(params: Dictionary) -> Dictionary:
	var path := String(params.get("path", ""))
	var root_type := String(params.get("root", "Node2D"))
	var force := bool(params.get("force", false))
	var open_after_create := bool(params.get("open", false))
	var guard := _guard_scene_path(path, force)
	if not guard.is_empty():
		return guard
	if not ClassDB.can_instantiate(root_type) or not ClassDB.is_parent_class(root_type, "Node"):
		return ToolResponse.failure("not an instantiable Node class: %s" % root_type)

	var root: Node = ClassDB.instantiate(root_type)
	root.name = path.get_file().get_basename()
	var packed := PackedScene.new()
	var pack_err := packed.pack(root)
	root.free()
	if pack_err != OK:
		return ToolResponse.failure("pack failed: %s" % error_string(pack_err))
	var save_err := ResourceSaver.save(packed, path)
	if save_err != OK:
		return ToolResponse.failure("save failed: %s" % error_string(save_err))
	if open_after_create:
		EditorInterface.open_scene_from_path(path)
	return ToolResponse.success({ "created": path, "root": root_type, "opened": open_after_create })

func _save_as(params: Dictionary) -> Dictionary:
	var path := String(params.get("path", ""))
	var force := bool(params.get("force", false))
	var guard := _guard_scene_path(path, force)
	if not guard.is_empty():
		return guard
	var root := EditorInterface.get_edited_scene_root()
	if root == null:
		return ToolResponse.failure("no scene to save")
	if EditorInterface.has_method("save_scene_as"):
		var save_result: Variant = EditorInterface.call("save_scene_as", path)
		if typeof(save_result) == TYPE_INT and int(save_result) != OK:
			return ToolResponse.failure("save-as failed: %s" % error_string(int(save_result)))
		if not FileAccess.file_exists(path):
			return ToolResponse.failure("save-as did not create scene: %s" % path)
		return ToolResponse.success({ "saved": path })
	var packed := PackedScene.new()
	var err := packed.pack(root)
	if err == OK:
		err = ResourceSaver.save(packed, path)
	if err == OK:
		EditorInterface.open_scene_from_path(path)
	if err != OK:
		return ToolResponse.failure("save-as failed: %s" % error_string(err))
	return ToolResponse.success({ "saved": path })

func _guard_scene_path(path: String, force: bool) -> Dictionary:
	if path == "":
		return ToolResponse.failure("scene path is required")
	if not path.begins_with("res://"):
		return ToolResponse.failure("scene path must start with res://")
	if not path.ends_with(".tscn"):
		return ToolResponse.failure("scene path must end with .tscn")
	if not _is_safe_res_path(path):
		return ToolResponse.failure("scene path must stay inside res://")
	if FileAccess.file_exists(path) and not force:
		return ToolResponse.failure("scene already exists: %s (pass --force to overwrite)" % path)
	return {}

func _is_safe_res_path(path: String) -> bool:
	if path.find("\\") != -1:
		return false
	var rel := path.substr("res://".length())
	if rel == "" or rel.begins_with("/"):
		return false
	for part in rel.split("/", true):
		if part == "" or part == "." or part == "..":
			return false
	return true

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

func _to_unique_strings(values) -> Array:
	var result: Array = []
	var seen := {}
	for v in values:
		var text := String(v)
		if seen.has(text):
			continue
		seen[text] = true
		result.append(text)
	return result
