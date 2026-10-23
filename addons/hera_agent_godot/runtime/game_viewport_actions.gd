extends RefCounted

const JOY_BUTTONS: Dictionary = {
	"A": JOY_BUTTON_A,
	"B": JOY_BUTTON_B,
	"X": JOY_BUTTON_X,
	"Y": JOY_BUTTON_Y,
	"BACK": JOY_BUTTON_BACK,
	"GUIDE": JOY_BUTTON_GUIDE,
	"START": JOY_BUTTON_START,
	"LEFT_STICK": JOY_BUTTON_LEFT_STICK,
	"RIGHT_STICK": JOY_BUTTON_RIGHT_STICK,
	"LEFT_SHOULDER": JOY_BUTTON_LEFT_SHOULDER,
	"RIGHT_SHOULDER": JOY_BUTTON_RIGHT_SHOULDER,
	"DPAD_UP": JOY_BUTTON_DPAD_UP,
	"DPAD_DOWN": JOY_BUTTON_DPAD_DOWN,
	"DPAD_LEFT": JOY_BUTTON_DPAD_LEFT,
	"DPAD_RIGHT": JOY_BUTTON_DPAD_RIGHT,
}

const JOY_AXES: Dictionary = {
	"LEFT_X": JOY_AXIS_LEFT_X,
	"LEFT_Y": JOY_AXIS_LEFT_Y,
	"RIGHT_X": JOY_AXIS_RIGHT_X,
	"RIGHT_Y": JOY_AXIS_RIGHT_Y,
	"TRIGGER_LEFT": JOY_AXIS_TRIGGER_LEFT,
	"TRIGGER_RIGHT": JOY_AXIS_TRIGGER_RIGHT,
}

const MOUSE_BUTTONS: Dictionary = {
	"left": MOUSE_BUTTON_LEFT,
	"right": MOUSE_BUTTON_RIGHT,
	"middle": MOUSE_BUTTON_MIDDLE,
	"wheel_up": MOUSE_BUTTON_WHEEL_UP,
	"wheel_down": MOUSE_BUTTON_WHEEL_DOWN,
}

const KEYS: Dictionary = {
	"SPACE": KEY_SPACE,
	"ENTER": KEY_ENTER,
	"RETURN": KEY_ENTER,
	"ESCAPE": KEY_ESCAPE,
	"ESC": KEY_ESCAPE,
	"TAB": KEY_TAB,
	"BACKSPACE": KEY_BACKSPACE,
	"LEFT": KEY_LEFT,
	"RIGHT": KEY_RIGHT,
	"UP": KEY_UP,
	"DOWN": KEY_DOWN,
	"SHIFT": KEY_SHIFT,
	"CTRL": KEY_CTRL,
	"CONTROL": KEY_CTRL,
	"ALT": KEY_ALT,
	"META": KEY_META,
	"CMD": KEY_META,
	"COMMAND": KEY_META,
}


const GameImageAnalyzer = preload("res://addons/hera_agent_godot/runtime/game_image_analyzer.gd")
const CaptureEvidence = preload("res://addons/hera_agent_godot/core/capture_evidence.gd")

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
		"joypad":
			return _joypad_input(request)
		"axis":
			return _axis_input(request)
		_:
			return { "ok": false, "error": "unknown input kind: %s (want mouse|key|action|text|joypad|axis)" % kind }

static func sizes_match(width: int, height: int, project_width: int, project_height: int) -> bool:
	return project_width > 0 and project_height > 0 and width == project_width and height == project_height

static func geometry(viewport: Viewport) -> Dictionary:
	var visible := Vector2.ZERO
	if viewport != null:
		visible = viewport.get_visible_rect().size
	var window := DisplayServer.window_get_size()
	var project_width := int(ProjectSettings.get_setting("display/window/size/viewport_width", 0))
	var project_height := int(ProjectSettings.get_setting("display/window/size/viewport_height", 0))
	var visible_width := int(visible.x)
	var visible_height := int(visible.y)
	return {
		"window_width": int(window.x),
		"window_height": int(window.y),
		"visible_width": visible_width,
		"visible_height": visible_height,
		"project_width": project_width,
		"project_height": project_height,
	}

static func screenshot(viewport: Viewport, request: Dictionary, scene_path: String, pid: int) -> Dictionary:
	var guard := CaptureEvidence.guard(request)
	if guard != "":
		return {"ok": false, "error": guard}
	if DisplayServer.get_name() == "headless":
		return CaptureEvidence.capture_error(request, "headless renderer")
	var image := viewport.get_texture().get_image()
	var evidence := CaptureEvidence.snapshot("runtime", "")
	if image == null or image.is_empty():
		return { "ok": false, "error": "runtime screenshot produced an empty image" }
	var out_path := String(request.get("path", "user://hera_game_screenshots/latest.png"))
	var abs_path := ProjectSettings.globalize_path(out_path)
	DirAccess.make_dir_recursive_absolute(abs_path.get_base_dir())
	var err := image.save_png(out_path)
	if err != OK:
		return { "ok": false, "error": "save failed: %s" % error_string(err) }
	var data := geometry(viewport)
	var width := image.get_width()
	var height := image.get_height()
	data["path"] = abs_path
	data["width"] = width
	data["height"] = height
	data["pid"] = pid
	data["scene"] = scene_path
	if bool(request.get("evidence", false)):
		evidence["width"] = width
		evidence["height"] = height
		evidence["geometry"] = geometry(viewport)
		CaptureEvidence.link(evidence, request)
		data["evidence"] = evidence
	data["size_matches_project"] = sizes_match(width, height, int(data.get("project_width", 0)), int(data.get("project_height", 0)))
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

static func _joypad_input(request: Dictionary) -> Dictionary:
	var mode := String(request.get("mode", ""))
	if mode != "press" and mode != "release":
		return { "ok": false, "error": "joypad input requires mode press or release" }
	var button := _joy_button_index(request.get("button", ""))
	if button < 0:
		return { "ok": false, "error": "unknown joypad button: %s" % String(request.get("button", "")) }
	var event := InputEventJoypadButton.new()
	event.device = int(request.get("device", HERA_INPUT_DEVICE_ID))
	event.button_index = button
	event.pressed = mode == "press"
	Input.parse_input_event(event)
	return {
		"ok": true,
		"events": [event],
		"data": {
			"kind": "joypad",
			"mode": mode,
			"button": joy_button_name(button),
			"button_index": button,
			"device": event.device,
		},
	}

static func _axis_input(request: Dictionary) -> Dictionary:
	var axis := _joy_axis_index(request.get("axis", ""))
	if axis < 0:
		return { "ok": false, "error": "unknown joypad axis: %s" % String(request.get("axis", "")) }
	if not request.has("value"):
		return { "ok": false, "error": "axis input requires value" }
	var value := float(request.get("value", 0.0))
	if value < -1.0 or value > 1.0:
		return { "ok": false, "error": "axis value must be between -1 and 1" }
	var event := InputEventJoypadMotion.new()
	event.device = int(request.get("device", HERA_INPUT_DEVICE_ID))
	event.axis = axis
	event.axis_value = value
	Input.parse_input_event(event)
	return {
		"ok": true,
		"events": [event],
		"data": {
			"kind": "axis",
			"axis": joy_axis_name(axis),
			"axis_index": axis,
			"value": value,
			"device": event.device,
		},
	}

static func _joy_button_index(raw: Variant) -> int:
	if typeof(raw) == TYPE_INT or typeof(raw) == TYPE_FLOAT:
		var index := int(raw)
		return index if index >= 0 else -1
	var name := String(raw).strip_edges().to_upper().replace("-", "_")
	if name.is_valid_int():
		return int(name)
	if name.begins_with("JOY_BUTTON_"):
		name = name.substr(11)
	return int(JOY_BUTTONS.get(name, -1))

static func joy_button_name(index: int) -> String:
	var name: Variant = JOY_BUTTONS.find_key(index)
	return str(index) if name == null else String(name)

static func _joy_axis_index(raw: Variant) -> int:
	if typeof(raw) == TYPE_INT or typeof(raw) == TYPE_FLOAT:
		var index := int(raw)
		return index if index >= 0 else -1
	var name := String(raw).strip_edges().to_upper().replace("-", "_")
	if name.is_valid_int():
		return int(name)
	if name.begins_with("JOY_AXIS_"):
		name = name.substr(9)
	return int(JOY_AXES.get(name, -1))

static func joy_axis_name(index: int) -> String:
	var name: Variant = JOY_AXES.find_key(index)
	return str(index) if name == null else String(name)

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
	return int(MOUSE_BUTTONS.get(name.to_lower(), 0))

static func _mouse_button_name(index: int) -> String:
	var name: Variant = MOUSE_BUTTONS.find_key(index)
	return str(index) if name == null else String(name)

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
	if KEYS.has(raw_key):
		return int(KEYS[raw_key])
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
	var name: Variant = KEYS.find_key(keycode)
	if name != null:
		return "KEY_" + String(name)
	if keycode >= KEY_F1 and keycode <= KEY_F12:
		return "KEY_F%d" % (keycode - KEY_F1 + 1)
	return str(keycode)
