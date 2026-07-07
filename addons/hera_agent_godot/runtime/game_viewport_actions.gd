extends RefCounted

const GameImageAnalyzer = preload("res://addons/hera_agent_godot/runtime/game_image_analyzer.gd")

const HERA_INPUT_DEVICE_ID := 240920

static func click(viewport: Viewport, position: Vector2) -> void:
	_push_mouse_button(viewport, position, true)
	_push_mouse_button(viewport, position, false)

static func input(viewport: Viewport, request: Dictionary) -> Dictionary:
	var kind := String(request.get("kind", ""))
	match kind:
		"mouse":
			return _mouse_input(viewport, request)
		"key":
			return _key_input(viewport, request)
		"action":
			return _action_input(request)
		"text":
			return _text_input(viewport, request)
		_:
			return { "ok": false, "error": "unknown input kind: %s (want mouse|key|action|text)" % kind }

static func screenshot(viewport: Viewport, request: Dictionary, scene_path: String, pid: int) -> Dictionary:
	var image := viewport.get_texture().get_image()
	if image == null or image.is_empty():
		return { "ok": false, "error": "runtime screenshot produced an empty image" }
	var out_path := String(request.get("path", "user://hera_game_screenshots/latest.png"))
	var abs_path := ProjectSettings.globalize_path(out_path)
	DirAccess.make_dir_recursive_absolute(abs_path.get_base_dir())
	var err := image.save_png(out_path)
	if err != OK:
		return { "ok": false, "error": "save failed: %s" % error_string(err) }
	var data := {
		"path": abs_path,
		"width": image.get_width(),
		"height": image.get_height(),
		"pid": pid,
		"scene": scene_path,
	}
	if bool(request.get("analyze", false)):
		data["analysis"] = GameImageAnalyzer.analyze(image)
	return { "ok": true, "data": data }

static func _push_mouse_button(viewport: Viewport, position: Vector2, pressed: bool) -> void:
	var event := InputEventMouseButton.new()
	event.button_index = MOUSE_BUTTON_LEFT
	event.pressed = pressed
	event.position = position
	event.global_position = position
	event.factor = 1.0
	viewport.push_input(event, true)

static func _mouse_input(viewport: Viewport, request: Dictionary) -> Dictionary:
	var mode := String(request.get("mode", ""))
	if mode == "":
		return { "ok": false, "error": "mouse input requires mode" }
	var position := Vector2(float(request.get("x", -1.0)), float(request.get("y", -1.0)))
	var viewport_size := viewport.get_visible_rect().size
	if position.x < 0.0 or position.y < 0.0 or position.x >= viewport_size.x or position.y >= viewport_size.y:
		return { "ok": false, "error": "input position outside viewport: %s" % position }
	var button := _mouse_button_index(String(request.get("button", "left")))
	if mode != "move" and button == 0:
		return { "ok": false, "error": "unknown mouse button: %s" % String(request.get("button", "")) }
	var events: Array[InputEvent] = []
	match mode:
		"press":
			events.append(_mouse_button_event(position, button, true, request))
		"release":
			events.append(_mouse_button_event(position, button, false, request))
		"click":
			events.append(_mouse_button_event(position, button, true, request))
			events.append(_mouse_button_event(position, button, false, request))
		"move":
			events.append(_mouse_motion_event(position, request))
		_:
			return { "ok": false, "error": "unknown mouse input mode: %s" % mode }
	for event in events:
		viewport.push_input(event, true)
	return {
		"ok": true,
		"events": events,
		"data": {
			"kind": "mouse",
			"mode": mode,
			"x": int(position.x),
			"y": int(position.y),
			"button": _mouse_button_name(button),
		},
	}

static func _key_input(viewport: Viewport, request: Dictionary) -> Dictionary:
	var mode := String(request.get("mode", ""))
	if mode != "press" and mode != "release":
		return { "ok": false, "error": "key input requires mode press or release" }
	var keycode := _keycode(request)
	if keycode <= 0:
		return { "ok": false, "error": "unknown key: %s" % String(request.get("key", request.get("keycode", ""))) }
	var event := InputEventKey.new()
	event.device = HERA_INPUT_DEVICE_ID
	event.pressed = mode == "press"
	event.keycode = keycode
	event.physical_keycode = keycode if bool(request.get("physical", false)) else 0
	event.unicode = int(request.get("unicode", 0))
	_apply_modifiers(event, request)
	if String(request.get("route", "input")) == "viewport":
		viewport.push_input(event, true)
	else:
		Input.parse_input_event(event)
	return {
		"ok": true,
		"events": [event],
		"data": {
			"kind": "key",
			"mode": mode,
			"key": _key_name(keycode),
			"keycode": keycode,
			"physical": bool(request.get("physical", false)),
			"route": String(request.get("route", "input")),
		},
	}

static func _action_input(request: Dictionary) -> Dictionary:
	var name := String(request.get("name", ""))
	if name == "":
		return { "ok": false, "error": "action input requires name" }
	var mode := String(request.get("mode", ""))
	if mode != "press" and mode != "release":
		return { "ok": false, "error": "action input requires mode press or release" }
	var event := InputEventAction.new()
	event.device = HERA_INPUT_DEVICE_ID
	event.action = StringName(name)
	event.pressed = mode == "press"
	event.strength = float(request.get("strength", 1.0 if event.pressed else 0.0))
	Input.parse_input_event(event)
	return {
		"ok": true,
		"events": [event],
		"data": {
			"kind": "action",
			"mode": mode,
			"name": name,
			"strength": event.strength,
		},
	}

static func _text_input(viewport: Viewport, request: Dictionary) -> Dictionary:
	var text := String(request.get("text", ""))
	if text == "":
		return { "ok": false, "error": "text input requires text" }
	viewport.push_text_input(text)
	return {
		"ok": true,
		"text": text,
		"data": {
			"kind": "text",
			"text": text,
		},
	}

static func _mouse_button_event(position: Vector2, button: int, pressed: bool, request: Dictionary) -> InputEventMouseButton:
	var event := InputEventMouseButton.new()
	event.device = HERA_INPUT_DEVICE_ID
	event.button_index = button
	event.pressed = pressed
	event.position = position
	event.global_position = position
	event.factor = 1.0
	event.double_click = bool(request.get("double", false))
	_apply_modifiers(event, request)
	return event

static func _mouse_motion_event(position: Vector2, request: Dictionary) -> InputEventMouseMotion:
	var event := InputEventMouseMotion.new()
	event.device = HERA_INPUT_DEVICE_ID
	event.position = position
	event.global_position = position
	event.relative = Vector2(float(request.get("dx", 0.0)), float(request.get("dy", 0.0)))
	_apply_modifiers(event, request)
	return event

static func _apply_modifiers(event: InputEventWithModifiers, request: Dictionary) -> void:
	var modifiers: Array = request.get("modifiers", [])
	for raw_modifier in modifiers:
		match String(raw_modifier).to_lower():
			"shift":
				event.shift_pressed = true
			"ctrl":
				event.ctrl_pressed = true
			"alt":
				event.alt_pressed = true
			"meta":
				event.meta_pressed = true

static func _mouse_button_index(name: String) -> int:
	match name.to_lower():
		"left":
			return MOUSE_BUTTON_LEFT
		"right":
			return MOUSE_BUTTON_RIGHT
		"middle":
			return MOUSE_BUTTON_MIDDLE
		"wheel_up":
			return MOUSE_BUTTON_WHEEL_UP
		"wheel_down":
			return MOUSE_BUTTON_WHEEL_DOWN
		_:
			return 0

static func _mouse_button_name(index: int) -> String:
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

static func _keycode(request: Dictionary) -> int:
	if request.has("keycode"):
		return int(request.get("keycode", 0))
	var raw_key := String(request.get("key", "")).to_upper()
	if raw_key.begins_with("KEY_"):
		raw_key = raw_key.substr(4)
	if raw_key.length() == 1:
		var code := raw_key.unicode_at(0)
		if code >= 65 and code <= 90:
			return KEY_A + (code - 65)
		if code >= 48 and code <= 57:
			return KEY_0 + (code - 48)
	match raw_key:
		"SPACE":
			return KEY_SPACE
		"ENTER", "RETURN":
			return KEY_ENTER
		"ESC", "ESCAPE":
			return KEY_ESCAPE
		"TAB":
			return KEY_TAB
		"BACKSPACE":
			return KEY_BACKSPACE
		"LEFT":
			return KEY_LEFT
		"RIGHT":
			return KEY_RIGHT
		"UP":
			return KEY_UP
		"DOWN":
			return KEY_DOWN
		"SHIFT":
			return KEY_SHIFT
		"CTRL", "CONTROL":
			return KEY_CTRL
		"ALT":
			return KEY_ALT
		"META", "CMD", "COMMAND":
			return KEY_META
		_:
			if raw_key.begins_with("F") and raw_key.length() <= 3:
				var number := int(raw_key.substr(1))
				if number >= 1 and number <= 12:
					return KEY_F1 + number - 1
	return 0

static func _key_name(keycode: int) -> String:
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
