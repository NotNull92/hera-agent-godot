extends Node

const REQUEST_DIR := "user://hera_game_requests"
const RESPONSE_DIR := "user://hera_game_responses"
const LEGACY_REQUEST_PATH := "user://hera_game_request.json"
const LEGACY_RESPONSE_PATH := "user://hera_game_response.json"
const MAX_NODES := 1000
const MAX_VALUE_LEN := 200

func _process(_delta: float) -> void:
	if not OS.has_feature("editor"):
		return
	if FileAccess.file_exists(LEGACY_REQUEST_PATH):
		_handle_file(LEGACY_REQUEST_PATH, LEGACY_RESPONSE_PATH)
	var dir := DirAccess.open(REQUEST_DIR)
	if dir == null:
		return
	dir.list_dir_begin()
	var file_name := dir.get_next()
	while file_name != "":
		if not dir.current_is_dir() and file_name.ends_with(".json"):
			var request_path := "%s/%s" % [REQUEST_DIR, file_name]
			var response_path := "%s/%s" % [RESPONSE_DIR, file_name]
			_handle_file(request_path, response_path)
		file_name = dir.get_next()
	dir.list_dir_end()

func _handle_file(path: String, response_path: String) -> void:
	var file := FileAccess.open(path, FileAccess.READ)
	if file == null:
		return
	var text := file.get_as_text()
	file.close()
	DirAccess.remove_absolute(ProjectSettings.globalize_path(path))
	var decoded: Variant = JSON.parse_string(text)
	if typeof(decoded) != TYPE_DICTIONARY:
		return
	var request := decoded as Dictionary
	if String(request.get("id", "")) == "":
		_write(response_path, { "ok": false, "error": "invalid game request" })
		return
	_write(response_path, _handle(request))

func _handle(request: Dictionary) -> Dictionary:
	var action := String(request.get("action", ""))
	match action:
		"tree":
			return _tree(request)
		"get":
			return _get_node(request)
		"set":
			return _set_node(request)
		"call":
			return _call_node(request)
		_:
			return _response(request, false, { "error": "unknown game action: %s (want tree|get|set|call)" % action })

func _tree(request: Dictionary) -> Dictionary:
	var root := get_tree().root
	var nodes: Array = []
	_collect(root, nodes)
	var truncated := nodes.size() > MAX_NODES
	if truncated:
		nodes = nodes.slice(0, MAX_NODES)
	return _response(request, true, {
		"count": nodes.size(),
		"truncated": truncated,
		"nodes": nodes,
	})

func _get_node(request: Dictionary) -> Dictionary:
	var path := String(request.get("path", ""))
	if path == "":
		return _response(request, false, { "error": "path is required" })
	var node := _resolve(path)
	if node == null:
		return _response(request, false, { "error": "node not found: %s" % path })
	return _response(request, true, {
		"path": String(node.get_path()),
		"type": node.get_class(),
		"name": String(node.name),
		"properties": _properties(node),
	})

func _set_node(request: Dictionary) -> Dictionary:
	var path := String(request.get("path", ""))
	var node := _node_from_request(request)
	if node == null:
		return _response(request, false, { "error": "node not found: %s" % path })
	var prop := String(request.get("prop", ""))
	var prop_info := _property_info(node, prop)
	if prop == "" or prop_info.is_empty():
		return _response(request, false, { "error": "node has no property: %s" % prop })
	var coerced := _coerce(request.get("value"), prop_info)
	if not bool(coerced.get("ok", false)):
		return _response(request, false, { "error": String(coerced.get("error", "invalid property value")) })
	node.set(prop, coerced.get("value"))
	return _response(request, true, {
		"path": String(node.get_path()),
		"prop": prop,
		"value": str(node.get(prop)),
	})

func _call_node(request: Dictionary) -> Dictionary:
	var path := String(request.get("path", ""))
	var node := _node_from_request(request)
	if node == null:
		return _response(request, false, { "error": "node not found: %s" % path })
	var method := String(request.get("method", ""))
	if method == "" or not node.has_method(method):
		return _response(request, false, { "error": "node has no method: %s" % method })
	var call_args := _call_args(request)
	var result: Variant = node.callv(method, call_args)
	return _response(request, true, {
		"path": String(node.get_path()),
		"method": method,
		"result": str(result),
		"type": type_string(typeof(result)),
	})

func _node_from_request(request: Dictionary) -> Node:
	var path := String(request.get("path", ""))
	if path == "":
		return null
	return _resolve(path)

func _collect(node: Node, out: Array) -> void:
	if out.size() > MAX_NODES:
		return
	out.append({
		"path": String(node.get_path()),
		"type": node.get_class(),
		"name": String(node.name),
	})
	for child in node.get_children():
		_collect(child, out)
		if out.size() > MAX_NODES:
			return

func _resolve(path: String) -> Node:
	if path.begins_with("/"):
		return get_node_or_null(NodePath(path))
	var current := get_tree().current_scene
	return current.get_node_or_null(path) if current != null else null

func _property_info(node: Node, prop: String) -> Dictionary:
	for p in node.get_property_list():
		if String(p.get("name", "")) == prop:
			return p
	return {}

func _coerce(raw: Variant, prop_info: Dictionary) -> Dictionary:
	var target_type := int(prop_info.get("type", TYPE_NIL))
	if typeof(raw) != TYPE_STRING:
		return { "ok": true, "value": raw }
	var text := String(raw)
	match target_type:
		TYPE_STRING:
			return { "ok": true, "value": text }
		TYPE_STRING_NAME:
			return { "ok": true, "value": StringName(text) }
		TYPE_BOOL:
			var lower := text.to_lower()
			if lower == "true" or lower == "1":
				return { "ok": true, "value": true }
			if lower == "false" or lower == "0":
				return { "ok": true, "value": false }
			return { "ok": false, "error": "invalid bool value for property: %s" % text }
		TYPE_INT:
			if not text.is_valid_int():
				return { "ok": false, "error": "invalid int value for property: %s" % text }
			return { "ok": true, "value": int(text) }
		TYPE_FLOAT:
			if not text.is_valid_float():
				return { "ok": false, "error": "invalid float value for property: %s" % text }
			return { "ok": true, "value": float(text) }
		TYPE_NODE_PATH:
			return { "ok": true, "value": NodePath(text) }
		TYPE_NIL:
			if text == "null":
				return { "ok": true, "value": null }
			return { "ok": false, "error": "cannot infer type for null property: %s" % String(prop_info.get("name", "")) }
		TYPE_OBJECT:
			return { "ok": false, "error": "unsupported object/resource property: %s" % String(prop_info.get("name", "")) }
		_:
			var parsed: Variant = str_to_var(text)
			if parsed == null and text != "null":
				return { "ok": false, "error": "invalid %s value for property: %s" % [type_string(target_type), text] }
			if typeof(parsed) != target_type:
				return { "ok": false, "error": "property expects %s, got %s" % [type_string(target_type), type_string(typeof(parsed))] }
			return { "ok": true, "value": parsed }

func _call_args(request: Dictionary) -> Array:
	var raw_args: Variant = request.get("args", [])
	if typeof(raw_args) != TYPE_ARRAY:
		return []
	var result: Array = []
	for raw in raw_args:
		result.append(_call_arg(raw))
	return result

func _call_arg(raw: Variant) -> Variant:
	if typeof(raw) != TYPE_STRING:
		return raw
	var text := String(raw)
	var parsed: Variant = str_to_var(text)
	if parsed == null and text != "null":
		return text
	return parsed

func _properties(node: Node) -> Dictionary:
	var result := {}
	for prop in node.get_property_list():
		var usage := int(prop.get("usage", 0))
		if not (usage & PROPERTY_USAGE_EDITOR):
			continue
		if usage & (PROPERTY_USAGE_CATEGORY | PROPERTY_USAGE_GROUP | PROPERTY_USAGE_SUBGROUP):
			continue
		var pname := String(prop.get("name", ""))
		if pname == "":
			continue
		var value := str(node.get(pname))
		if value.length() > MAX_VALUE_LEN:
			value = value.substr(0, MAX_VALUE_LEN) + "..."
		result[pname] = value
	return result

func _response(request: Dictionary, ok: bool, payload: Dictionary) -> Dictionary:
	payload["id"] = String(request.get("id", ""))
	payload["ok"] = ok
	return payload

func _write(response_path: String, response: Dictionary) -> void:
	DirAccess.make_dir_recursive_absolute(ProjectSettings.globalize_path(RESPONSE_DIR))
	var file := FileAccess.open(response_path, FileAccess.WRITE)
	if file == null:
		return
	file.store_string(JSON.stringify(response))
	file.close()
