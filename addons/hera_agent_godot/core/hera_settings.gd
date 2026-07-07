extends RefCounted

const GAME_FEEL_UI_MODE_SETTING := "hera_agent_godot/ui_juicy_mode"
const GAME_FEEL_MODE_SETTING := "hera_agent_godot/game_feel_mode"


static func get_game_feel_ui_mode_enabled() -> bool:
	return _get_bool(GAME_FEEL_UI_MODE_SETTING, false)


static func set_game_feel_ui_mode_enabled(enabled: bool) -> void:
	_set_bool(GAME_FEEL_UI_MODE_SETTING, enabled)


static func get_game_feel_mode_enabled() -> bool:
	return _get_bool(GAME_FEEL_MODE_SETTING, false)


static func set_game_feel_mode_enabled(enabled: bool) -> void:
	_set_bool(GAME_FEEL_MODE_SETTING, enabled)


static func _get_bool(key: String, fallback: bool) -> bool:
	var settings: EditorSettings = EditorInterface.get_editor_settings()
	if not settings.has_setting(key):
		settings.set_setting(key, fallback)
		settings.mark_setting_changed(key)
	return bool(settings.get_setting(key))


static func _set_bool(key: String, enabled: bool) -> void:
	var settings: EditorSettings = EditorInterface.get_editor_settings()
	settings.set_setting(key, enabled)
	settings.mark_setting_changed(key)
