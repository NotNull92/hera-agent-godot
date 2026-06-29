extends RefCounted

const ToolResponse = preload("res://addons/hera_agent_godot/core/tool_response.gd")

const MAX_LIST_LIMIT := 500
const MAX_LIST_SCAN := 5000

var _type_filter := ""
var _pattern := ""
var _limit := 100
var _scanned := 0

func list_resources(params: Dictionary) -> Dictionary:
	var path := String(params.get("path", "res://"))
	if path == "":
		path = "res://"
	if not path.begins_with("res://") or not _is_safe_res_container(path):
		return ToolResponse.failure("list path must stay inside res://")
	_limit = clampi(int(params.get("limit", 100)), 1, MAX_LIST_LIMIT)
	_type_filter = String(params.get("type", ""))
	_pattern = String(params.get("pattern", "")).to_lower()
	_scanned = 0
	var resources := []
	if ResourceLoader.exists(path):
		_append_resource(path, resources)
		_scanned = 1
	else:
		var guard := _collect_resource_files(path, resources)
		if guard != "":
			return ToolResponse.failure(guard)
	return ToolResponse.success({
		"path": path,
		"count": resources.size(),
		"scanned": _scanned,
		"limit": _limit,
		"resources": resources,
	})

func _collect_resource_files(path: String, resources: Array) -> String:
	var dir := DirAccess.open(path)
	if dir == null:
		return "resource directory not found: %s" % path
	dir.list_dir_begin()
	var name := dir.get_next()
	while name != "":
		if resources.size() >= _limit or _scanned >= MAX_LIST_SCAN:
			break
		if name != "." and name != "..":
			var child_path := path.path_join(name)
			if dir.current_is_dir():
				if not name.begins_with("."):
					var guard := _collect_resource_files(child_path, resources)
					if guard != "":
						return guard
			elif _is_resource_extension(child_path):
				_scanned += 1
				_append_resource(child_path, resources)
		name = dir.get_next()
	return ""

func _append_resource(path: String, resources: Array) -> void:
	if _pattern != "" and path.to_lower().find(_pattern) == -1:
		return
	var res := ResourceLoader.load(path)
	if res == null:
		return
	var type := res.get_class()
	if _type_filter != "":
		if not res.is_class(_type_filter):
			return
		type = res.get_class()
	resources.append({ "path": path, "type": type })

func _is_resource_extension(path: String) -> bool:
	return ["tscn", "scn", "tres", "res", "gdshader", "shader", "gd", "cs"].has(path.get_extension().to_lower())

func _is_safe_res_container(path: String) -> bool:
	if path == "res://":
		return true
	return _is_safe_res_path(path)

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
