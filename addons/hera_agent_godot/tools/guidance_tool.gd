extends RefCounted

const ToolResponse = preload("res://addons/hera_agent_godot/core/tool_response.gd")

const UI_JUICY_MODE_SETTING := "hera_agent_godot/ui_juicy_mode"


func get_name() -> String:
	return "guidance"


func execute(params: Dictionary) -> Dictionary:
	var action := String(params.get("action", "ui"))
	match action:
		"ui":
			return _ui_guidance()
		_:
			return ToolResponse.failure("unknown guidance action: %s (want ui)" % action)


func _ui_guidance() -> Dictionary:
	var game_feel_enabled := _get_game_feel_ui_mode_enabled()
	var mode := "game_feel" if game_feel_enabled else "standard"
	return ToolResponse.success({
		"mode": mode,
		"game_feel_ui_mode": game_feel_enabled,
		"setting": UI_JUICY_MODE_SETTING,
		"instruction": _instruction(game_feel_enabled),
		"checklist": _checklist(game_feel_enabled),
	})


func _get_game_feel_ui_mode_enabled() -> bool:
	var settings: EditorSettings = EditorInterface.get_editor_settings()
	if not settings.has_setting(UI_JUICY_MODE_SETTING):
		return false
	return bool(settings.get_setting(UI_JUICY_MODE_SETTING))


func _instruction(enabled: bool) -> String:
	if enabled:
		return "Build UI with Game Feel: immediate feedback, crisp transitions, expressive state changes, satisfying input response, and runtime visual QA."
	return "Build UI with the standard Hera guidance: clear layout, readable state, predictable controls, and runtime visual QA."


func _checklist(enabled: bool) -> Array[String]:
	if enabled:
		return [
			"Give every primary input immediate pressed/hover/disabled feedback.",
			"Use brief motion, squash, pulse, flash, or count-up feedback where it clarifies state.",
			"Make success, failure, damage, reward, and progress changes visibly satisfying.",
			"Keep motion bounded and verify it does not cause clipping or layout shift.",
			"Capture runtime UI tree and screenshot analysis after interaction.",
		]
	return [
		"Keep layout readable and stable.",
		"Expose clear state labels and reachable controls.",
		"Verify Control rects through runtime UI tree.",
		"Capture screenshot analysis for visual changes.",
	]
