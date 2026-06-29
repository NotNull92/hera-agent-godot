extends RefCounted

const ToolResponse = preload("res://addons/hera_agent_godot/core/tool_response.gd")
const ScriptInspector = preload("res://addons/hera_agent_godot/tools/script_inspector.gd")
const ScriptTemplate = preload("res://addons/hera_agent_godot/tools/script_template.gd")

var _inspector: RefCounted
var _template: RefCounted


func _init() -> void:
	_inspector = ScriptInspector.new()
	_template = ScriptTemplate.new()


func get_name() -> String:
	return "script"


func execute(params: Dictionary) -> Dictionary:
	var action := String(params.get("action", ""))
	match action:
		"current":
			return _current()
		"inspect":
			return _inspect(params)
		"open":
			return _open(params)
		"create":
			return _create(params)
		_:
			return ToolResponse.failure("unknown script action: %s (want current|inspect|open|create)" % action)


func _current() -> Dictionary:
	var script := _current_script()
	if script == null:
		return ToolResponse.success({ "found": false, "path": "" })
	var path := script.resource_path
	if path != "" and path.ends_with(".gd") and FileAccess.file_exists(path):
		return _inspect_path(path)
	return ToolResponse.success({
		"found": true,
		"path": path,
		"base_type": script.get_instance_base_type(),
		"source_available": false,
	})


func _inspect(params: Dictionary) -> Dictionary:
	var path := String(params.get("path", ""))
	var guard := _guard_readable_script_path(path)
	if guard != "":
		return ToolResponse.failure(guard)
	return _inspect_path(path)


func _open(params: Dictionary) -> Dictionary:
	var path := String(params.get("path", ""))
	var guard := _guard_readable_script_path(path)
	if guard != "":
		return ToolResponse.failure(guard)
	var loaded := ResourceLoader.load(path)
	if not (loaded is Script):
		return ToolResponse.failure("not a script resource: %s" % path)
	if not EditorInterface.has_method("edit_script"):
		return ToolResponse.failure("editor cannot open scripts in this Godot version")
	var line := int(params.get("line", 1))
	var column := int(params.get("column", 1))
	EditorInterface.call("edit_script", loaded, max(0, line - 1), max(0, column - 1), true)
	return ToolResponse.success({
		"opened": path,
		"line": line,
		"column": column,
		"base_type": (loaded as Script).get_instance_base_type(),
	})


func _inspect_path(path: String) -> Dictionary:
	var file := FileAccess.open(path, FileAccess.READ)
	if file == null:
		return ToolResponse.failure("could not read script: %s" % path)
	var source := file.get_as_text()
	file.close()
	var metadata: Dictionary = _inspector.inspect(source)
	metadata["found"] = true
	metadata["path"] = path
	metadata["lines"] = source.split("\n", false).size()
	var loaded_res := ResourceLoader.load(path)
	if loaded_res is Script:
		var loaded: Script = loaded_res as Script
		metadata["base_type"] = loaded.get_instance_base_type()
	return ToolResponse.success(metadata)


func _create(params: Dictionary) -> Dictionary:
	var path := String(params.get("path", ""))
	var guard := _guard_script_path(path, bool(params.get("force", false)))
	if not guard.is_empty():
		return guard
	var template_result: Dictionary = _template.build(params)
	if not bool(template_result.get("ok", false)):
		return ToolResponse.failure(String(template_result.get("error", "invalid script template")))
	var dir_err := _ensure_parent_dir(path)
	if dir_err != "":
		return ToolResponse.failure(dir_err)
	var file := FileAccess.open(path, FileAccess.WRITE)
	if file == null:
		return ToolResponse.failure("could not create script: %s" % path)
	file.store_string(String(template_result.get("text", "")))
	file.close()
	_refresh_filesystem()
	return ToolResponse.success({
		"created": path,
		"extends": String(template_result.get("extends", "")),
		"class_name": String(template_result.get("class_name", "")),
		"tool": bool(template_result.get("tool", false)),
		"signals": template_result.get("signals", []),
		"exports": int(template_result.get("exports", 0)),
	})


func _current_script() -> Script:
	if not EditorInterface.has_method("get_script_editor"):
		return null
	var script_editor: Object = EditorInterface.call("get_script_editor")
	if script_editor == null or not script_editor.has_method("get_current_script"):
		return null
	var raw: Variant = script_editor.call("get_current_script")
	if raw is Script:
		return raw as Script
	return null


func _guard_readable_script_path(path: String) -> String:
	if path == "":
		return "script path is required"
	if not path.begins_with("res://"):
		return "script path must start with res://"
	if not path.ends_with(".gd"):
		return "script path must end with .gd"
	if not _is_safe_res_path(path):
		return "script path must stay inside res://"
	if not FileAccess.file_exists(path):
		return "script not found: %s" % path
	return ""


func _guard_script_path(path: String, force: bool) -> Dictionary:
	var read_guard := _guard_readable_script_parent(path)
	if read_guard != "":
		return ToolResponse.failure(read_guard)
	if FileAccess.file_exists(path) and not force:
		return ToolResponse.failure("script already exists: %s (pass --force to overwrite)" % path)
	return {}


func _guard_readable_script_parent(path: String) -> String:
	if path == "":
		return "script path is required"
	if not path.begins_with("res://"):
		return "script path must start with res://"
	if not path.ends_with(".gd"):
		return "script path must end with .gd"
	if not _is_safe_res_path(path):
		return "script path must stay inside res://"
	return ""


func _ensure_parent_dir(path: String) -> String:
	var parent := path.get_base_dir()
	var err := DirAccess.make_dir_recursive_absolute(ProjectSettings.globalize_path(parent))
	if err != OK:
		return "could not create parent directory: %s" % parent
	return ""


func _is_safe_res_path(path: String) -> bool:
	if path.find("\\") != -1:
		return false
	var rel := path.substr("res://".length())
	if rel == "" or rel.begins_with("/"):
		return false
	for part in rel.split("/", true):
		if ["", ".", ".."].has(part):
			return false
	return true


func _refresh_filesystem() -> void:
	var fs := EditorInterface.get_resource_filesystem()
	if fs != null:
		fs.scan()
