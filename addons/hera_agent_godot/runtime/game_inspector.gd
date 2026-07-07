extends Node

const GameValueCodec = preload("res://addons/hera_agent_godot/runtime/game_value_codec.gd")
const GameAssertions = preload("res://addons/hera_agent_godot/runtime/game_assertions.gd")
const GameTreeInspector = preload("res://addons/hera_agent_godot/runtime/game_tree_inspector.gd")
const GameUIInspector = preload("res://addons/hera_agent_godot/runtime/game_ui_inspector.gd")
const GameViewportActions = preload("res://addons/hera_agent_godot/runtime/game_viewport_actions.gd")

const INSTANCE_DIR := "user://hera_game_instances"
const REQUEST_ROOT := "user://hera_game_requests"
const RESPONSE_ROOT := "user://hera_game_responses"
const MAX_NODES := 1000
const HEARTBEAT_INTERVAL_SEC := 0.5
const MAX_INPUT_LOG := 200
const LONG_CLICK_MS := 500

var _pid := 0
var _heartbeat_accum := 0.0
var _input_log: Array[Dictionary] = []
var _mouse_presses := {}
var _active_keys := {}
var _active_mouse_buttons := {}

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

func _input(event: InputEvent) -> void:
	if event.device == GameViewportActions.HERA_INPUT_DEVICE_ID:
		return
	_record_input_event(event, "external")

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
		"ui_tree":
			return _ui_tree(request)
		"get":
			return _get_node(request)
		"set":
			return _set_node(request)
		"assert":
			return _assert_node(request)
		"call":
			return _call_node(request)
		"qa_discover":
			return _qa_discover(request)
		"click":
			return _click_viewport(request)
		"input":
			return _input_viewport(request)
		"input_log":
			return _input_log_response(request)
		"screenshot":
			return _screenshot(request)
		_:
			return _response(request, false, { "error": "unknown game action: %s (want tree|ui_tree|get|set|assert|call|qa_discover|click|input|input_log|screenshot)" % action })

func _tree(request: Dictionary) -> Dictionary:
	var result := GameTreeInspector.tree(get_tree().root, MAX_NODES)
	return _response(request, true, {
		"count": result.get("count", 0),
		"pid": _pid,
		"scene": _current_scene_path(),
		"truncated": result.get("truncated", false),
		"nodes": result.get("nodes", []),
	})

func _ui_tree(request: Dictionary) -> Dictionary:
	var result := GameUIInspector.tree(get_tree().root, get_tree().current_scene, MAX_NODES, request)
	if not bool(result.get("ok", false)):
		return _response(request, false, { "error": String(result.get("error", "ui tree failed")) })
	return _response(request, true, {
		"count": result.get("count", 0),
		"pid": _pid,
		"scene": _current_scene_path(),
		"truncated": result.get("truncated", false),
		"controls": result.get("controls", []),
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

func _qa_discover(request: Dictionary) -> Dictionary:
	var path := String(request.get("path", ""))
	var node: Node = get_tree().current_scene if path == "" else _resolve(path)
	if node == null:
		var target := "current scene" if path == "" else path
		return _response(request, false, { "error": "node not found: %s" % target })
	var methods := _qa_methods(node)
	return _response(request, true, {
		"path": String(node.get_path()),
		"type": node.get_class(),
		"count": methods.size(),
		"methods": methods,
	})

func _click_viewport(request: Dictionary) -> Dictionary:
	var target := GameUIInspector.click_target(get_tree().root, get_tree().current_scene, request)
	if not bool(target.get("ok", false)):
		return _response(request, false, { "error": String(target.get("error", "invalid click target")) })
	var position: Vector2 = target.get("position", Vector2.ZERO)
	var viewport_size := get_viewport().get_visible_rect().size
	if position.x < 0.0 or position.y < 0.0 or position.x >= viewport_size.x or position.y >= viewport_size.y:
		return _response(request, false, { "error": "click position outside viewport: %s" % position })
	GameViewportActions.click(get_viewport(), position)
	var data := {
		"x": int(position.x),
		"y": int(position.y),
		"viewport_width": int(viewport_size.x),
		"viewport_height": int(viewport_size.y),
	}
	if target.has("path"):
		data["path"] = target.get("path")
	if target.has("text"):
		data["text"] = target.get("text")
	return _response(request, true, data)

func _input_viewport(request: Dictionary) -> Dictionary:
	var result := GameViewportActions.input(get_viewport(), request)
	if not bool(result.get("ok", false)):
		return _response(request, false, { "error": String(result.get("error", "runtime input failed")) })
	var events: Array = result.get("events", [])
	for event in events:
		_record_input_event(event as InputEvent, "hera")
	if result.has("text"):
		_record_text_input(String(result.get("text", "")), "hera")
	var data: Dictionary = result.get("data", {})
	data["pid"] = _pid
	data["log_count"] = _input_log.size()
	return _response(request, true, data)

func _input_log_response(request: Dictionary) -> Dictionary:
	var limit := int(request.get("limit", 20))
	if limit < 0:
		return _response(request, false, { "error": "limit must be non-negative" })
	var events := _recent_input_log(limit)
	var cleared := bool(request.get("clear", false))
	if cleared:
		_input_log.clear()
	return _response(request, true, {
		"count": events.size(),
		"total": _input_log.size(),
		"cleared": cleared,
		"events": events,
	})

func _screenshot(request: Dictionary) -> Dictionary:
	var result := GameViewportActions.screenshot(get_viewport(), request, _current_scene_path(), _pid)
	if not bool(result.get("ok", false)):
		return _response(request, false, { "error": String(result.get("error", "runtime screenshot failed")) })
	var data: Dictionary = result.get("data", {})
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

func _qa_methods(node: Node) -> Array[Dictionary]:
	var methods: Array[Dictionary] = []
	for method_info in node.get_method_list():
		var method := method_info as Dictionary
		var method_name := String(method.get("name", ""))
		if not method_name.begins_with("qa_") or not node.has_method(method_name):
			continue
		var entry := {
			"name": method_name,
		}
		var args := _method_arg_names(method)
		if not args.is_empty():
			entry["args"] = args
		var default_count := _method_default_count(method)
		if default_count > 0:
			entry["defaults"] = default_count
		var return_type := _method_return_type(method)
		if return_type != "":
			entry["return"] = return_type
		methods.append(entry)
	return methods

func _record_input_event(event: InputEvent, source: String) -> void:
	if event == null:
		return
	if event is InputEventMouseButton:
		_record_mouse_button(event as InputEventMouseButton, source)
	elif event is InputEventMouseMotion:
		_record_mouse_motion(event as InputEventMouseMotion, source)
	elif event is InputEventKey:
		_record_key(event as InputEventKey, source)
	elif event is InputEventAction:
		_record_action(event as InputEventAction, source)

func _record_mouse_button(event: InputEventMouseButton, source: String) -> void:
	var now := Time.get_ticks_msec()
	var button_name := _mouse_button_name(event.button_index)
	var duration_ms := 0
	if event.pressed:
		_mouse_presses[event.button_index] = {
			"ts": now,
			"x": int(event.position.x),
			"y": int(event.position.y),
		}
		_active_mouse_buttons[event.button_index] = true
	else:
		var press: Dictionary = _mouse_presses.get(event.button_index, {})
		if not press.is_empty():
			duration_ms = now - int(press.get("ts", now))
		_mouse_presses.erase(event.button_index)
		_active_mouse_buttons.erase(event.button_index)
	var entry := _base_input_entry("mouse_button", source)
	entry["button"] = button_name
	entry["button_index"] = int(event.button_index)
	entry["pressed"] = event.pressed
	entry["x"] = int(event.position.x)
	entry["y"] = int(event.position.y)
	entry["double_click"] = event.double_click
	entry["duration_ms"] = duration_ms
	entry["click_kind"] = _click_kind(event.pressed, duration_ms)
	entry["modifiers"] = _modifiers(event)
	entry["active_keys"] = _active_key_names()
	entry["active_mouse_buttons"] = _active_mouse_button_names()
	_append_input_log(entry)

func _record_mouse_motion(event: InputEventMouseMotion, source: String) -> void:
	var entry := _base_input_entry("mouse_motion", source)
	entry["x"] = int(event.position.x)
	entry["y"] = int(event.position.y)
	entry["dx"] = int(event.relative.x)
	entry["dy"] = int(event.relative.y)
	entry["modifiers"] = _modifiers(event)
	entry["active_keys"] = _active_key_names()
	entry["active_mouse_buttons"] = _active_mouse_button_names()
	_append_input_log(entry)

func _record_key(event: InputEventKey, source: String) -> void:
	var key_name := _key_name(event.keycode if event.keycode != 0 else event.physical_keycode)
	if event.pressed:
		_active_keys[key_name] = true
	else:
		_active_keys.erase(key_name)
	var entry := _base_input_entry("key", source)
	entry["key"] = key_name
	entry["keycode"] = int(event.keycode)
	entry["physical_keycode"] = int(event.physical_keycode)
	entry["unicode"] = int(event.unicode)
	entry["pressed"] = event.pressed
	entry["echo"] = event.echo
	entry["modifiers"] = _modifiers(event)
	entry["active_keys"] = _active_key_names()
	entry["active_mouse_buttons"] = _active_mouse_button_names()
	_append_input_log(entry)

func _record_action(event: InputEventAction, source: String) -> void:
	var entry := _base_input_entry("action", source)
	entry["name"] = String(event.action)
	entry["pressed"] = event.pressed
	entry["strength"] = event.strength
	entry["active_keys"] = _active_key_names()
	entry["active_mouse_buttons"] = _active_mouse_button_names()
	_append_input_log(entry)

func _record_text_input(text: String, source: String) -> void:
	var entry := _base_input_entry("text", source)
	entry["text"] = text
	entry["length"] = text.length()
	entry["active_keys"] = _active_key_names()
	entry["active_mouse_buttons"] = _active_mouse_button_names()
	_append_input_log(entry)

func _base_input_entry(kind: String, source: String) -> Dictionary:
	return {
		"kind": kind,
		"source": source,
		"ts_ms": Time.get_ticks_msec(),
	}

func _append_input_log(entry: Dictionary) -> void:
	_input_log.append(entry)
	while _input_log.size() > MAX_INPUT_LOG:
		_input_log.pop_front()

func _recent_input_log(limit: int) -> Array[Dictionary]:
	var events: Array[Dictionary] = []
	var start := 0 if limit == 0 or limit >= _input_log.size() else _input_log.size() - limit
	for index in range(start, _input_log.size()):
		events.append(_input_log[index])
	return events

func _click_kind(pressed: bool, duration_ms: int) -> String:
	if pressed:
		return "press"
	if duration_ms >= LONG_CLICK_MS:
		return "long"
	return "short"

func _modifiers(event: InputEventWithModifiers) -> Array[String]:
	var values: Array[String] = []
	if event.shift_pressed:
		values.append("shift")
	if event.ctrl_pressed:
		values.append("ctrl")
	if event.alt_pressed:
		values.append("alt")
	if event.meta_pressed:
		values.append("meta")
	return values

func _active_key_names() -> Array[String]:
	var values: Array[String] = []
	for key in _active_keys.keys():
		values.append(String(key))
	values.sort()
	return values

func _active_mouse_button_names() -> Array[String]:
	var values: Array[String] = []
	for button in _active_mouse_buttons.keys():
		values.append(_mouse_button_name(int(button)))
	values.sort()
	return values

func _mouse_button_name(index: int) -> String:
	match index:
		MOUSE_BUTTON_LEFT:
			return "left"
		MOUSE_BUTTON_RIGHT:
			return "right"
		MOUSE_BUTTON_MIDDLE:
			return "middle"
		MOUSE_BUTTON_WHEEL_UP:
			return "wheel_up"
		MOUSE_BUTTON_WHEEL_DOWN:
			return "wheel_down"
		_:
			return str(index)

func _key_name(keycode: int) -> String:
	if keycode >= KEY_A and keycode <= KEY_Z:
		return "KEY_%s" % String.chr(65 + keycode - KEY_A)
	if keycode >= KEY_0 and keycode <= KEY_9:
		return "KEY_%s" % String.chr(48 + keycode - KEY_0)
	match keycode:
		KEY_SPACE:
			return "KEY_SPACE"
		KEY_ENTER:
			return "KEY_ENTER"
		KEY_ESCAPE:
			return "KEY_ESCAPE"
		KEY_TAB:
			return "KEY_TAB"
		KEY_BACKSPACE:
			return "KEY_BACKSPACE"
		KEY_LEFT:
			return "KEY_LEFT"
		KEY_RIGHT:
			return "KEY_RIGHT"
		KEY_UP:
			return "KEY_UP"
		KEY_DOWN:
			return "KEY_DOWN"
		KEY_SHIFT:
			return "KEY_SHIFT"
		KEY_CTRL:
			return "KEY_CTRL"
		KEY_ALT:
			return "KEY_ALT"
		KEY_META:
			return "KEY_META"
		_:
			if keycode >= KEY_F1 and keycode <= KEY_F12:
				return "KEY_F%d" % (keycode - KEY_F1 + 1)
	return str(keycode)

func _method_arg_names(method: Dictionary) -> Array[String]:
	var names: Array[String] = []
	var raw_args: Array = method.get("args", [])
	for raw_arg in raw_args:
		var arg := raw_arg as Dictionary
		var arg_name := String(arg.get("name", ""))
		if arg_name == "":
			arg_name = "arg%d" % names.size()
		names.append(arg_name)
	return names

func _method_default_count(method: Dictionary) -> int:
	var default_args: Array = method.get("default_args", [])
	return default_args.size()

func _method_return_type(method: Dictionary) -> String:
	var return_info: Dictionary = method.get("return", {})
	var type_id := int(return_info.get("type", TYPE_NIL))
	if type_id == TYPE_NIL:
		return ""
	if type_id == TYPE_OBJECT:
		var object_class_name := String(return_info.get("class_name", ""))
		if object_class_name != "":
			return object_class_name
	return type_string(type_id)

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
