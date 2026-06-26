extends Node

const GameValueCodec = preload("res://addons/hera_agent_godot/runtime/game_value_codec.gd")
const GameImageAnalyzer = preload("res://addons/hera_agent_godot/runtime/game_image_analyzer.gd")
const GameAssertions = preload("res://addons/hera_agent_godot/runtime/game_assertions.gd")

const INSTANCE_DIR := "user://hera_game_instances"
const REQUEST_ROOT := "user://hera_game_requests"
const RESPONSE_ROOT := "user://hera_game_responses"
const MAX_NODES := 1000
const HEARTBEAT_INTERVAL_SEC := 0.5

var _pid := 0
var _heartbeat_accum := 0.0

func _ready() -> void:
	_pid = OS.get_process_id()
	_ensure_dirs()
	_write_heartbeat()

func _exit_tree() -> void:
	if _pid != 0:
		DirAccess.remove_absolute(ProjectSettings.globalize_path(_instance_path()))

func _process(delta: float) -> void:
	if not OS.has_feature("editor"):
		return
	_heartbeat_accum += delta
	if _heartbeat_accum >= HEARTBEAT_INTERVAL_SEC:
		_heartbeat_accum = 0.0
		_write_heartbeat()
	var dir := DirAccess.open(_request_dir())
	if dir == null:
		return
	dir.list_dir_begin()
	var file_name := dir.get_next()
	while file_name != "":
		if not dir.current_is_dir() and file_name.ends_with(".json"):
			var request_path := "%s/%s" % [_request_dir(), file_name]
			var response_path := "%s/%s" % [_response_dir(), file_name]
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
	if int(request.get("target_pid", _pid)) != _pid:
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
		"assert":
			return _assert_node(request)
		"call":
			return _call_node(request)
		"click":
			return _click_viewport(request)
		"screenshot":
			return _screenshot(request)
		_:
			return _response(request, false, { "error": "unknown game action: %s (want tree|get|set|assert|call|click|screenshot)" % action })

func _tree(request: Dictionary) -> Dictionary:
	var root := get_tree().root
	var nodes: Array = []
	_collect(root, nodes)
	var truncated := nodes.size() > MAX_NODES
	if truncated:
		nodes = nodes.slice(0, MAX_NODES)
	return _response(request, true, {
		"count": nodes.size(),
		"pid": _pid,
		"scene": _current_scene_path(),
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
	var properties := _properties_from_request(node, request)
	if not bool(properties.get("ok", false)):
		return _response(request, false, { "error": String(properties.get("error", "property not found")) })
	return _response(request, true, {
		"path": String(node.get_path()),
		"type": node.get_class(),
		"name": String(node.name),
		"properties": properties.get("properties", {}),
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
	var coerced := GameValueCodec.coerce(request.get("value"), prop_info)
	if not bool(coerced.get("ok", false)):
		return _response(request, false, { "error": String(coerced.get("error", "invalid property value")) })
	node.set(prop, coerced.get("value"))
	return _response(request, true, {
		"path": String(node.get_path()),
		"prop": prop,
		"value": str(node.get(prop)),
	})

func _assert_node(request: Dictionary) -> Dictionary:
	var path := String(request.get("path", ""))
	var node := _node_from_request(request)
	if node == null:
		return _response(request, false, { "error": "node not found: %s" % path })
	var assertion := GameAssertions.check(node, request)
	if not bool(assertion.get("ok", false)):
		return _response(request, false, { "error": String(assertion.get("error", "assert failed")) })
	return _response(request, true, {
		"path": String(node.get_path()),
		"prop": assertion.get("prop", ""),
		"op": assertion.get("op", ""),
		"actual": assertion.get("actual", ""),
		"expected": assertion.get("expected", ""),
	})

func _call_node(request: Dictionary) -> Dictionary:
	var path := String(request.get("path", ""))
	var node := _node_from_request(request)
	if node == null:
		return _response(request, false, { "error": "node not found: %s" % path })
	var method := String(request.get("method", ""))
	if method == "" or not node.has_method(method):
		return _response(request, false, { "error": "node has no method: %s" % method })
	var call_args := GameValueCodec.call_args(request)
	var result: Variant = node.callv(method, call_args)
	return _response(request, true, {
		"path": String(node.get_path()),
		"method": method,
		"result": str(result),
		"type": type_string(typeof(result)),
	})

func _click_viewport(request: Dictionary) -> Dictionary:
	if not request.has("x") or not request.has("y"):
		return _response(request, false, { "error": "click requires x and y" })
	var position := Vector2(float(request.get("x", 0)), float(request.get("y", 0)))
	var viewport_size := get_viewport().get_visible_rect().size
	if position.x < 0.0 or position.y < 0.0 or position.x >= viewport_size.x or position.y >= viewport_size.y:
		return _response(request, false, { "error": "click position outside viewport: %s" % position })
	_push_mouse_button(position, true)
	_push_mouse_button(position, false)
	return _response(request, true, {
		"x": int(position.x),
		"y": int(position.y),
		"viewport_width": int(viewport_size.x),
		"viewport_height": int(viewport_size.y),
	})

func _push_mouse_button(position: Vector2, pressed: bool) -> void:
	var event := InputEventMouseButton.new()
	event.button_index = MOUSE_BUTTON_LEFT
	event.pressed = pressed
	event.position = position
	event.global_position = position
	event.factor = 1.0
	get_viewport().push_input(event, true)

func _screenshot(request: Dictionary) -> Dictionary:
	var image := get_viewport().get_texture().get_image()
	if image == null or image.is_empty():
		return _response(request, false, { "error": "runtime screenshot produced an empty image" })
	var out_path := String(request.get("path", "user://hera_game_screenshots/latest.png"))
	var abs_path := ProjectSettings.globalize_path(out_path)
	DirAccess.make_dir_recursive_absolute(abs_path.get_base_dir())
	var err := image.save_png(out_path)
	if err != OK:
		return _response(request, false, { "error": "save failed: %s" % error_string(err) })
	var data := {
		"path": abs_path,
		"width": image.get_width(),
		"height": image.get_height(),
		"pid": _pid,
		"scene": _current_scene_path(),
	}
	if bool(request.get("analyze", false)):
		data["analysis"] = GameImageAnalyzer.analyze(image)
	return _response(request, true, data)

func _properties_from_request(node: Node, request: Dictionary) -> Dictionary:
	if request.has("prop"):
		var selected := GameValueCodec.selected_properties(node, [String(request.get("prop", ""))])
		if bool(selected.get("ok", false)):
			return selected
		return { "ok": false, "error": String(selected.get("error", "property not found")) }
	if request.has("props"):
		var selected := GameValueCodec.selected_properties(node, request.get("props", []))
		if bool(selected.get("ok", false)):
			return selected
		return { "ok": false, "error": String(selected.get("error", "property not found")) }
	return { "ok": true, "properties": GameValueCodec.properties(node) }

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

func _response(request: Dictionary, ok: bool, payload: Dictionary) -> Dictionary:
	payload["id"] = String(request.get("id", ""))
	payload["ok"] = ok
	return payload

func _write(response_path: String, response: Dictionary) -> void:
	DirAccess.make_dir_recursive_absolute(ProjectSettings.globalize_path(_response_dir()))
	var file := FileAccess.open(response_path, FileAccess.WRITE)
	if file == null:
		return
	file.store_string(JSON.stringify(response))
	file.close()

func _ensure_dirs() -> void:
	DirAccess.make_dir_recursive_absolute(ProjectSettings.globalize_path(INSTANCE_DIR))
	DirAccess.make_dir_recursive_absolute(ProjectSettings.globalize_path(_request_dir()))
	DirAccess.make_dir_recursive_absolute(ProjectSettings.globalize_path(_response_dir()))

func _write_heartbeat() -> void:
	_ensure_dirs()
	var file := FileAccess.open(_instance_path(), FileAccess.WRITE)
	if file == null:
		return
	file.store_string(JSON.stringify({
		"pid": _pid,
		"scene": _current_scene_path(),
		"ts": Time.get_unix_time_from_system(),
	}))
	file.close()

func _current_scene_path() -> String:
	var current := get_tree().current_scene
	return "" if current == null else current.scene_file_path

func _request_dir() -> String:
	return "%s/%d" % [REQUEST_ROOT, _pid]

func _response_dir() -> String:
	return "%s/%d" % [RESPONSE_ROOT, _pid]

func _instance_path() -> String:
	return "%s/%d.json" % [INSTANCE_DIR, _pid]
