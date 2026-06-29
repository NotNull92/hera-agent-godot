extends RefCounted

const ToolResponse = preload("res://addons/hera_agent_godot/core/tool_response.gd")

static func export(params: Dictionary) -> Dictionary:
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
	var result := _build_library(root, params)
	root.free()
	return result

static func _build_library(root: Node, params: Dictionary) -> Dictionary:
	var scene_path := String(params.get("path", ""))
	var output := String(params.get("output", ""))
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
	if exported.is_empty():
		return ToolResponse.failure("no mesh items found in scene")
	var err := ResourceSaver.save(library, output)
	if err != OK:
		return ToolResponse.failure("save failed: %s" % error_string(err))
	return ToolResponse.success({ "source": scene_path, "output": output, "count": exported.size(), "items": exported })

static func _wanted_items(raw: Variant) -> Array:
	var out := []
	if typeof(raw) != TYPE_ARRAY:
		return out
	for value in raw:
		var name := String(value)
		if name != "":
			out.append(name)
	return out

static func _mesh_item_roots(root: Node) -> Array:
	var out := []
	for child in root.get_children():
		out.append(child)
	return out

static func _first_mesh_instance(node: Node) -> MeshInstance3D:
	if node is MeshInstance3D:
		return node
	for child in node.get_children():
		var found := _first_mesh_instance(child)
		if found != null:
			return found
	return null

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
