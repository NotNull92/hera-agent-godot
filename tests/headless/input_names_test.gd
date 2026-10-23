extends SceneTree

const Actions = preload("res://addons/hera_agent_godot/runtime/game_viewport_actions.gd")

func _initialize() -> void:
	for name: String in Actions.JOY_BUTTONS:
		var index: int = Actions._joy_button_index(name)
		assert(Actions.joy_button_name(index) == name)
		assert(Actions._joy_button_index("JOY_BUTTON_" + name) == index)
		assert(Actions._joy_button_index(name.to_lower().replace("_", "-")) == index)
		assert(Actions._joy_button_index(str(index)) == index)
	for name: String in Actions.JOY_AXES:
		var index: int = Actions._joy_axis_index(name)
		assert(Actions.joy_axis_name(index) == name)
		assert(Actions._joy_axis_index("JOY_AXIS_" + name) == index)
		assert(Actions._joy_axis_index(str(index)) == index)
	for name: String in Actions.MOUSE_BUTTONS:
		assert(Actions._mouse_button_name(Actions._mouse_button_index(name.to_upper())) == name)
	for name: String in Actions.KEYS:
		var index: int = Actions._keycode({"key": name.to_lower()})
		var canonical := String(Actions.KEYS.find_key(index))
		assert(Actions._key_name(index) == "KEY_" + canonical)
		assert(Actions._keycode({"key": "KEY_" + name}) == index)
	for code: int in range(KEY_A, KEY_Z + 1):
		assert(Actions._keycode({"key": Actions._key_name(code)}) == code)
	for code: int in range(KEY_0, KEY_9 + 1):
		assert(Actions._keycode({"key": Actions._key_name(code)}) == code)
	for code: int in range(KEY_F1, KEY_F12 + 1):
		assert(Actions._keycode({"key": Actions._key_name(code)}) == code)
	assert(Actions._joy_button_index("unknown") == -1)
	assert(Actions._joy_axis_index("unknown") == -1)
	assert(Actions._mouse_button_index("unknown") == 0)
	assert(Actions._keycode({"key": "unknown"}) == 0)
	quit()
