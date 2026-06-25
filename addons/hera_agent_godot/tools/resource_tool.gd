extends RefCounted

const ToolResponse = preload("res://addons/hera_agent_godot/core/tool_response.gd")

const MAX_VALUE_LEN := 200
const MAX_ERRORS := 20

func get_name() -> String:
	return "resource"

func execute(params: Dictionary) -> Dictionary:
	var action := String(params.get("action", ""))
	match action:
		"get":
			return _describe(params)
		"uid":
			return _uid(params)
		"resave":
			return _resave(params)
		"update_uids":
			return _update_uids()
		"export_mesh_library":
			return _export_mesh_library(params)
		_:
			return ToolResponse.failure("unknown resource action: %s (want get|uid|resave|update_uids|export_mesh_library)" % action)

func _describe(params: Dictionary) -> Dictionary:
	var path := String(params.get("path", ""))
	var guard := _guard_loadable_path(path)
	if guard != "":
		return ToolResponse.failure(guard)
	var res := ResourceLoader.load(path)
	if res == null:
		return ToolResponse.failure("failed to load resource: %s" % path)
	return ToolResponse.success({
		"path": path,
		"type": res.get_class(),
		"resource_name": res.resource_name,
		"uid": _resource_uid(path),
		"properties": _properties(res),
	})

func _uid(params: Dictionary) -> Dictionary:
	var path := String(params.get("path", ""))
	var guard := _guard_loadable_path(path)
	if guard != "":
		return ToolResponse.failure(guard)
	var uid_path := "%s.uid" % path
	var sidecar := ""
	if FileAccess.file_exists(uid_path):
		var file := FileAccess.open(uid_path, FileAccess.READ)
		if file != null:
			sidecar = file.get_as_text().strip_edges()
			file.close()
	return ToolResponse.success({
		"path": path,
		"uid": _resource_uid(path),
		"uid_path": uid_path,
		"sidecar_exists": FileAccess.file_exists(uid_path),
		"sidecar": sidecar,
	})

func _resave(params: Dictionary) -> Dictionary:
	var path := String(params.get("path", ""))
	var guard := _guard_loadable_path(path)
	if guard != "":
		return ToolResponse.failure(guard)
	var res := ResourceLoader.load(path)
	if res == null:
		return ToolResponse.failure("failed to load resource: %s" % path)
	var err := ResourceSaver.save(res, path)
	if err != OK:
		return ToolResponse.failure("save failed: %s" % error_string(err))
	_refresh_filesystem()
	return ToolResponse.success({ "resaved": path, "uid": _resource_uid(path) })

func _update_uids() -> Dictionary:
	var candidates := []
	_collect_resavable("res://", candidates)
	if candidates.is_empty():
		return ToolResponse.failure("no resavable resources found")
	var saved := 0
	var skipped := 0
	var errors := []
	for path in candidates:
		var file_path := String(path)
		if not ResourceLoader.exists(file_path):
			skipped += 1
			continue
		var res := ResourceLoader.load(file_path)
		if res == null:
			skipped += 1
			continue
		var err := ResourceSaver.save(res, file_path)
		if err == OK:
			saved += 1
		elif errors.size() < MAX_ERRORS:
			errors.append({ "path": file_path, "error": error_string(err) })
	_refresh_filesystem()
	return ToolResponse.success({
		"processed": candidates.size(),
		"saved": saved,
		"skipped": skipped,
		"errors": errors,
	})

func _export_mesh_library(params: Dictionary) -> Dictionary:
	var scene_path := String(params.get("path", ""))
	if not scene_path.begins_with("res://") or not scene_path.ends_with(".tscn"):
		return ToolResponse.failure("scene path must be a res:// .tscn file")
	if not _is_safe_res_path(scene_path) or not ResourceLoader.exists(scene_path, "PackedScene"):
		return ToolResponse.failure("scene not found or not safe: %s" % scene_path)
	var output := String(params.get("output", ""))
	if not output.begins_with("res://") or not (output.ends_with(".tres") or output.ends_with(".res")):
		return ToolResponse.failure("output must be a res:// .tres or .res path")
	if not _is_safe_res_path(output):
		return ToolResponse.failure("output path must stay inside res://")
	var packed: PackedScene = ResourceLoader.load(scene_path, "PackedScene")
	if packed == null:
		return ToolResponse.failure("failed to load scene: %s" % scene_path)
	var root := packed.instantiate()
	if root == null:
		return ToolResponse.failure("failed to instantiate scene: %s" % scene_path)
	var wanted := _wanted_items(params.get("items", []))
	var library := MeshLibrary.new()
	var exported := []
	var item_id := 0
	for node in _mesh_item_roots(root):
		var item_name := String(node.name)
		if not wanted.is_empty() and not wanted.has(item_name):
			continue
		var mesh_node := _first_mesh_instance(node)
		if mesh_node == null or mesh_node.mesh == null:
			continue
		library.create_item(item_id)
		library.set_item_name(item_id, item_name)
		library.set_item_mesh(item_id, mesh_node.mesh)
		library.set_item_mesh_transform(item_id, mesh_node.transform)
		exported.append({ "id": item_id, "name": item_name, "mesh": String(root.get_path_to(mesh_node)) })
		item_id += 1
	root.free()
	if exported.is_empty():
		return ToolResponse.failure("no mesh items found in scene")
	var err := ResourceSaver.save(library, output)
	if err != OK:
		return ToolResponse.failure("save failed: %s" % error_string(err))
	_refresh_filesystem()
	return ToolResponse.success({ "source": scene_path, "output": output, "count": exported.size(), "items": exported })

func _properties(res: Resource) -> Dictionary:
	var result := {}
	for prop in res.get_property_list():
		var usage := int(prop.get("usage", 0))
		if not (usage & PROPERTY_USAGE_EDITOR):
			continue
		if usage & (PROPERTY_USAGE_CATEGORY | PROPERTY_USAGE_GROUP | PROPERTY_USAGE_SUBGROUP):
			continue
		var pname := String(prop.get("name", ""))
		if pname == "":
			continue
		var text := str(res.get(pname))
		if text.length() > MAX_VALUE_LEN:
			text = text.substr(0, MAX_VALUE_LEN) + "..."
		result[pname] = text
	return result

func _guard_loadable_path(path: String) -> String:
	if not (path.begins_with("res://") or path.begins_with("user://")):
		return "path must start with res:// or user:// : %s" % path
	if path.begins_with("res://") and not _is_safe_res_path(path):
		return "path must stay inside res://"
	if not ResourceLoader.exists(path):
		return "resource not found: %s" % path
	return ""

func _resource_uid(path: String) -> String:
	if ResourceLoader.has_method("get_resource_uid"):
		var uid: int = ResourceLoader.call("get_resource_uid", path)
		if uid >= 0 and ResourceUID.has_method("id_to_text"):
			return String(ResourceUID.call("id_to_text", uid))
	return ""

func _collect_resavable(dir_path: String, out: Array) -> void:
	var dir := DirAccess.open(dir_path)
	if dir == null:
		return
	dir.list_dir_begin()
	var name := dir.get_next()
	while name != "":
		if name != "." and name != "..":
			var child_path := dir_path.path_join(name)
			if dir.current_is_dir():
				if not name.begins_with("."):
					_collect_resavable(child_path, out)
			elif _is_resavable_extension(child_path):
				out.append(child_path)
		name = dir.get_next()

func _is_resavable_extension(path: String) -> bool:
	return ["tscn", "scn", "tres", "res", "gdshader", "shader", "gd", "cs"].has(path.get_extension().to_lower())

func _wanted_items(raw: Variant) -> Array:
	var out := []
	if typeof(raw) != TYPE_ARRAY:
		return out
	for value in raw:
		var name := String(value)
		if name != "":
			out.append(name)
	return out

func _mesh_item_roots(root: Node) -> Array:
	var out := []
	for child in root.get_children():
		out.append(child)
	return out

func _first_mesh_instance(node: Node) -> MeshInstance3D:
	if node is MeshInstance3D:
		return node
	for child in node.get_children():
		var found := _first_mesh_instance(child)
		if found != null:
			return found
	return null

func _refresh_filesystem() -> void:
	var fs := EditorInterface.get_resource_filesystem()
	if fs != null:
		fs.scan()

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
