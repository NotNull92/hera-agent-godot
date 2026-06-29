extends RefCounted

const ToolResponse = preload("res://addons/hera_agent_godot/core/tool_response.gd")

const DEFAULT_LIMIT := 500
const MAX_LIMIT := 5000

func get_name() -> String:
	return "project"

func execute(params: Dictionary) -> Dictionary:
	var action := String(params.get("action", ""))
	match action:
		"info":
			return _info()
		"list_files":
			return _list_files(params)
		"scan":
			return _scan()
		"reimport":
			return _reimport(params)
		"mkdir":
			return _mkdir(params)
		"set_main_scene":
			return _set_main_scene(params)
		_:
			return ToolResponse.failure("unknown project action: %s (want info|list_files|scan|reimport|mkdir|set_main_scene)" % action)

func _info() -> Dictionary:
	var files := _scan_files()
	var counts := {
		"all": files.size(),
		"scene": 0,
		"script": 0,
		"resource": 0,
		"asset": 0,
		"shader": 0,
		"other": 0,
	}
	for path in files:
		var kind := _file_type(String(path))
		counts[kind] = int(counts.get(kind, 0)) + 1
	return ToolResponse.success({
		"name": String(ProjectSettings.get_setting("application/config/name", "")),
		"path": ProjectSettings.globalize_path("res://"),
		"godot": Engine.get_version_info(),
		"files": counts,
		"current_scene": _current_scene(),
	})

func _list_files(params: Dictionary) -> Dictionary:
	var want_type := String(params.get("type", "all"))
	if not ["all", "scene", "script", "resource", "asset", "shader"].has(want_type):
		return ToolResponse.failure("type must be one of all|scene|script|resource|asset|shader")
	var pattern := String(params.get("pattern", ""))
	var limit := clampi(int(params.get("limit", DEFAULT_LIMIT)), 1, MAX_LIMIT)
	var files := []
	var total := 0
	for path in _scan_files():
		var file_path := String(path)
		var kind := _file_type(file_path)
		if want_type != "all" and kind != want_type:
			continue
		if pattern != "" and not _matches_pattern(file_path, pattern):
			continue
		total += 1
		if files.size() < limit:
			files.append({ "path": file_path, "type": kind })
	return ToolResponse.success({
		"type": want_type,
		"pattern": pattern,
		"count": files.size(),
		"total": total,
		"truncated": total > files.size(),
		"files": files,
	})

func _mkdir(params: Dictionary) -> Dictionary:
	var path := String(params.get("path", ""))
	if path == "":
		return ToolResponse.failure("directory path is required")
	if not path.begins_with("res://"):
		return ToolResponse.failure("directory path must start with res://")
	if not _is_safe_res_path(path):
		return ToolResponse.failure("directory path must stay inside res://")
	var err := DirAccess.make_dir_recursive_absolute(ProjectSettings.globalize_path(path))
	if err != OK:
		return ToolResponse.failure("could not create directory: %s" % path)
	_refresh_filesystem()
	return ToolResponse.success({ "created": path })

func _scan() -> Dictionary:
	var fs := EditorInterface.get_resource_filesystem()
	if fs == null:
		return ToolResponse.failure("editor resource filesystem is not available")
	fs.scan()
	return ToolResponse.success({ "scanned": true, "path": "res://" })

func _reimport(params: Dictionary) -> Dictionary:
	var raw_paths: Variant = params.get("paths", [])
	if typeof(raw_paths) != TYPE_ARRAY:
		return ToolResponse.failure("paths must be an array")
	var paths := PackedStringArray()
	for raw_path in raw_paths:
		var path := String(raw_path)
		var guard := _guard_project_file(path)
		if guard != "":
			return ToolResponse.failure(guard)
		paths.append(path)
	var fs := EditorInterface.get_resource_filesystem()
	if fs == null:
		return ToolResponse.failure("editor resource filesystem is not available")
	if not fs.has_method("reimport_files"):
		return ToolResponse.failure("resource filesystem cannot reimport files in this Godot version")
	fs.call("reimport_files", paths)
	var out := []
	for path in paths:
		out.append(String(path))
	return ToolResponse.success({ "reimported": paths.size(), "paths": out })

func _set_main_scene(params: Dictionary) -> Dictionary:
	var path := String(params.get("path", ""))
	if path == "":
		return ToolResponse.failure("scene path is required")
	if not path.begins_with("res://"):
		return ToolResponse.failure("scene path must start with res://")
	if not _is_safe_res_path(path):
		return ToolResponse.failure("scene path must stay inside res://")
	if path.get_extension().to_lower() != "tscn":
		return ToolResponse.failure("scene path must end with .tscn")
	if not ResourceLoader.exists(path, "PackedScene"):
		return ToolResponse.failure("scene not found or not a PackedScene: %s" % path)
	ProjectSettings.set_setting("application/run/main_scene", path)
	var err := ProjectSettings.save()
	if err != OK:
		return ToolResponse.failure("could not save ProjectSettings: %s" % error_string(err))
	return ToolResponse.success({ "main_scene": path })

func _guard_project_file(path: String) -> String:
	if path == "":
		return "file path is required"
	if not path.begins_with("res://"):
		return "file path must start with res://"
	if not _is_safe_res_path(path):
		return "file path must stay inside res://"
	if not FileAccess.file_exists(path):
		return "file not found: %s" % path
	return ""

func _scan_files() -> Array:
	var out := []
	_scan_dir("res://", out)
	out.sort()
	return out

func _scan_dir(dir_path: String, out: Array) -> void:
	var dir := DirAccess.open(dir_path)
	if dir == null:
		return
	dir.list_dir_begin()
	var name := dir.get_next()
	while name != "":
		if name != "." and name != "..":
			var child_path := dir_path.path_join(name)
			if dir.current_is_dir():
				if not _skip_dir(name):
					_scan_dir(child_path, out)
			elif not name.ends_with(".uid") and not name.ends_with(".import"):
				out.append(child_path)
		name = dir.get_next()

func _skip_dir(name: String) -> bool:
	return name.begins_with(".") or ["addons_cache", ".godot", ".import", ".git"].has(name)

func _file_type(path: String) -> String:
	var ext := path.get_extension().to_lower()
	if ext == "tscn" or ext == "scn":
		return "scene"
	if ext == "gd" or ext == "cs":
		return "script"
	if ext == "tres" or ext == "res":
		return "resource"
	if ext == "gdshader" or ext == "shader":
		return "shader"
	if ["png", "jpg", "jpeg", "webp", "svg", "ogg", "wav", "mp3", "ttf", "otf", "fbx", "glb", "gltf", "obj"].has(ext):
		return "asset"
	return "other"

func _matches_pattern(path: String, pattern: String) -> bool:
	if pattern.find("*") != -1 or pattern.find("?") != -1:
		return path.matchn(pattern)
	return path.to_lower().find(pattern.to_lower()) != -1

func _current_scene() -> String:
	var root := EditorInterface.get_edited_scene_root()
	return "" if root == null else root.scene_file_path

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
