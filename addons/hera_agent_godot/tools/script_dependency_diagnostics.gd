extends RefCounted

static func analyze(script_path: String) -> Dictionary:
	var diagnostics := {
		"preloads": [],
		"missing_preloads": [],
		"global_class_type_hints": [],
	}
	if not script_path.ends_with(".gd"):
		return diagnostics
	var source := FileAccess.get_file_as_string(script_path)
	if source == "" and FileAccess.get_open_error() != OK:
		diagnostics["read_error"] = error_string(FileAccess.get_open_error())
		return diagnostics
	_collect_preload_diagnostics(source, diagnostics)
	_collect_global_class_type_hints(source, diagnostics)
	return diagnostics

static func _collect_preload_diagnostics(source: String, diagnostics: Dictionary) -> void:
	var preload_regex := RegEx.new()
	if preload_regex.compile("preload\\(\"(res://[^\"]+)\"\\)") != OK:
		return
	for result in preload_regex.search_all(source):
		var preload_path := String(result.get_string(1))
		_append_unique(diagnostics["preloads"], preload_path, 50)
		if not _is_safe_res_path(preload_path) or not FileAccess.file_exists(preload_path):
			_append_unique(diagnostics["missing_preloads"], preload_path, 50)

static func _collect_global_class_type_hints(source: String, diagnostics: Dictionary) -> void:
	var type_regex := RegEx.new()
	if type_regex.compile("(?::|->)\\s*([A-Z][A-Za-z0-9_]*)") != OK:
		return
	for result in type_regex.search_all(source):
		_append_unique(diagnostics["global_class_type_hints"], String(result.get_string(1)), 50)

static func _append_unique(values: Array, value: String, limit: int) -> void:
	if values.size() >= limit or values.has(value):
		return
	values.append(value)

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
