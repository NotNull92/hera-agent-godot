extends RefCounted


func build(params: Dictionary) -> Dictionary:
	var script_class := String(params.get("class_name", ""))
	if script_class != "" and not _is_identifier(script_class):
		return { "ok": false, "error": "class_name must be a valid GDScript identifier" }
	var base_class := String(params.get("extends", "Node"))
	if not _is_identifier(base_class):
		return { "ok": false, "error": "extends must be a valid GDScript identifier" }
	var signal_names := _strings(params.get("signals", []))
	for signal_name in signal_names:
		if not _is_identifier(signal_name):
			return { "ok": false, "error": "signal name must be a valid GDScript identifier: %s" % signal_name }
	var export_lines := []
	for export_spec in _strings(params.get("exports", [])):
		var export_line := _export_line(export_spec)
		if export_line == "":
			return { "ok": false, "error": "export must use name:type or name:type=value with valid identifiers: %s" % export_spec }
		export_lines.append(export_line)
	return {
		"ok": true,
		"text": _text(params, script_class, base_class, signal_names, export_lines),
		"extends": base_class,
		"class_name": script_class,
		"tool": bool(params.get("tool", false)),
		"signals": signal_names,
		"exports": export_lines.size(),
	}


func _text(params: Dictionary, script_class: String, base_class: String, signal_names: Array[String], export_lines: Array) -> String:
	var lines: Array[String] = []
	if bool(params.get("tool", false)):
		lines.append("@tool")
	if script_class != "":
		lines.append("class_name %s" % script_class)
	lines.append("extends %s" % base_class)
	lines.append("")
	for signal_name in signal_names:
		lines.append("signal %s" % signal_name)
	if not signal_names.is_empty():
		lines.append("")
	for export_line in export_lines:
		lines.append(String(export_line))
	if not export_lines.is_empty():
		lines.append("")
	_append_stub(lines, params, "ready", "_ready() -> void")
	_append_stub(lines, params, "process", "_process(delta: float) -> void")
	_append_stub(lines, params, "physics_process", "_physics_process(delta: float) -> void")
	_append_stub(lines, params, "input", "_input(event: InputEvent) -> void")
	_append_stub(lines, params, "unhandled_input", "_unhandled_input(event: InputEvent) -> void")
	lines.append("")
	return "\n".join(lines)


func _append_stub(lines: Array[String], params: Dictionary, key: String, signature: String) -> void:
	if not bool(params.get(key, false)):
		return
	if lines.size() > 0 and lines[lines.size() - 1] != "":
		lines.append("")
	lines.append("func %s:" % signature)
	lines.append("\tpass")
	lines.append("")


func _strings(raw: Variant) -> Array[String]:
	var out: Array[String] = []
	if typeof(raw) != TYPE_ARRAY:
		return out
	for value in raw:
		var text := String(value)
		if text != "":
			out.append(text)
	return out


func _export_line(spec: String) -> String:
	var equal_index := spec.find("=")
	var declaration := spec
	var default_value := ""
	if equal_index != -1:
		declaration = spec.substr(0, equal_index)
		default_value = spec.substr(equal_index + 1).strip_edges()
	var colon_index := declaration.find(":")
	if colon_index <= 0:
		return ""
	var export_name := declaration.substr(0, colon_index).strip_edges()
	var export_type := declaration.substr(colon_index + 1).strip_edges()
	if not _is_identifier(export_name) or not _is_safe_type_text(export_type):
		return ""
	if default_value != "":
		if default_value.find("\n") != -1 or default_value.find("\r") != -1:
			return ""
		return "@export var %s: %s = %s" % [export_name, export_type, default_value]
	return "@export var %s: %s" % [export_name, export_type]


func _is_safe_type_text(value: String) -> bool:
	if value == "":
		return false
	for index in range(value.length()):
		var code := value.unicode_at(index)
		var ok := _is_identifier_continue(code) or code == 91 or code == 93 or code == 44 or code == 32
		if not ok:
			return false
	return true


func _is_identifier(value: String) -> bool:
	if value == "" or not _is_identifier_start(value.unicode_at(0)):
		return false
	for index in range(1, value.length()):
		if not _is_identifier_continue(value.unicode_at(index)):
			return false
	return true


func _is_identifier_start(code: int) -> bool:
	return code == 95 or (code >= 65 and code <= 90) or (code >= 97 and code <= 122)


func _is_identifier_continue(code: int) -> bool:
	return _is_identifier_start(code) or (code >= 48 and code <= 57)
