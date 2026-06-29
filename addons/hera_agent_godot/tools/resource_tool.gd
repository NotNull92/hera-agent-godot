extends RefCounted

const ToolResponse = preload("res://addons/hera_agent_godot/core/tool_response.gd")
const ResourceValueCodec = preload("res://addons/hera_agent_godot/tools/resource_value_codec.gd")
const MeshLibraryExporter = preload("res://addons/hera_agent_godot/tools/mesh_library_exporter.gd")
const ResourceLister = preload("res://addons/hera_agent_godot/tools/resource_lister.gd")

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
		"list":
			return _list(params)
		"set":
			return _set_resource(params)
		"create":
			return _create(params)
		"resave":
			return _resave(params)
		"update_uids":
			return _update_uids()
		"export_mesh_library":
			return _export_mesh_library(params)
		_:
			return ToolResponse.failure("unknown resource action: %s (want get|uid|list|set|create|resave|update_uids|export_mesh_library)" % action)

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

func _list(params: Dictionary) -> Dictionary:
	var lister := ResourceLister.new()
	return lister.list_resources(params)

func _set_resource(params: Dictionary) -> Dictionary:
	var path := String(params.get("path", ""))
	var guard := _guard_loadable_path(path)
	if guard != "":
		return ToolResponse.failure(guard)
	var res := ResourceLoader.load(path)
	if res == null:
		return ToolResponse.failure("failed to load resource: %s" % path)
	var prop_result := ResourceValueCodec.apply_props(res, params.get("props", {}))
	if not bool(prop_result.get("ok", false)):
		return ToolResponse.failure(String(prop_result.get("error", "invalid resource properties")))
	var err := ResourceSaver.save(res, path)
	if err != OK:
		return ToolResponse.failure("save failed: %s" % error_string(err))
	return ToolResponse.success({
		"updated": path,
		"type": res.get_class(),
		"properties": prop_result.get("properties", {}),
		"uid": _resource_uid(path),
	})

func _create(params: Dictionary) -> Dictionary:
	var type := String(params.get("type", ""))
	if type == "":
		return ToolResponse.failure("resource type is required")
	if not ClassDB.class_exists(type):
		return ToolResponse.failure("class not found: %s" % type)
	if not ClassDB.can_instantiate(type) or not (type == "Resource" or ClassDB.is_parent_class(type, "Resource")):
		return ToolResponse.failure("not an instantiable Resource class: %s" % type)
	var path := String(params.get("path", ""))
	var guard := _guard_save_path(path, bool(params.get("force", false)))
	if guard != "":
		return ToolResponse.failure(guard)
	var res := ClassDB.instantiate(type) as Resource
	if res == null:
		return ToolResponse.failure("failed to instantiate Resource class: %s" % type)
	var prop_result := ResourceValueCodec.apply_props(res, params.get("props", {}))
	if not bool(prop_result.get("ok", false)):
		return ToolResponse.failure(String(prop_result.get("error", "invalid resource properties")))
	var dir_err := _ensure_parent_dir(path)
	if dir_err != "":
		return ToolResponse.failure(dir_err)
	var err := ResourceSaver.save(res, path)
	if err != OK:
		return ToolResponse.failure("save failed: %s" % error_string(err))
	_refresh_filesystem()
	return ToolResponse.success({
		"created": path,
		"type": res.get_class(),
		"properties": prop_result.get("properties", {}),
		"uid": _resource_uid(path),
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
	var response := MeshLibraryExporter.export(params)
	if bool(response.get("ok", false)):
		_refresh_filesystem()
	return response

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

func _guard_save_path(path: String, force: bool) -> String:
	if path == "":
		return "resource path is required"
	if not path.begins_with("res://"):
		return "resource path must start with res://"
	if not (path.ends_with(".tres") or path.ends_with(".res")):
		return "resource path must end with .tres or .res"
	if not _is_safe_res_path(path):
		return "resource path must stay inside res://"
	if FileAccess.file_exists(path) and not force:
		return "resource already exists: %s (pass --force to overwrite)" % path
	return ""

func _ensure_parent_dir(path: String) -> String:
	var parent := path.get_base_dir()
	var err := DirAccess.make_dir_recursive_absolute(ProjectSettings.globalize_path(parent))
	if err != OK:
		return "could not create parent directory: %s" % parent
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
