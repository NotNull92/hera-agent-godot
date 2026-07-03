extends RefCounted

const ToolResponse = preload("res://addons/hera_agent_godot/core/tool_response.gd")

const UI_JUICY_MODE_SETTING := "hera_agent_godot/ui_juicy_mode"


func get_name() -> String:
	return "status"


func execute(_params: Dictionary) -> Dictionary:
	var root := EditorInterface.get_edited_scene_root()
	var scene := ""
	if root != null:
		scene = root.scene_file_path
	var game_feel_enabled := _get_game_feel_ui_mode_enabled()
	return ToolResponse.success({
		"project_name": String(ProjectSettings.get_setting("application/config/name", "")),
		"project_path": ProjectSettings.globalize_path("res://"),
		"godot_version": String(Engine.get_version_info().get("string", "")),
		"scene": scene,
		"pid": OS.get_process_id(),
		"game_feel_ui_mode": game_feel_enabled,
	})


func _get_game_feel_ui_mode_enabled() -> bool:
	var settings: EditorSettings = EditorInterface.get_editor_settings()
	if not settings.has_setting(UI_JUICY_MODE_SETTING):
		return false
	return bool(settings.get_setting(UI_JUICY_MODE_SETTING))
