extends RefCounted

const ToolResponse = preload("res://addons/hera_agent_godot/core/tool_response.gd")
const HeraSettings = preload("res://addons/hera_agent_godot/core/hera_settings.gd")


func get_name() -> String:
	return "status"


func execute(_params: Dictionary) -> Dictionary:
	var root := EditorInterface.get_edited_scene_root()
	var scene := ""
	if root != null:
		scene = root.scene_file_path
	var game_feel_ui_enabled := HeraSettings.get_game_feel_ui_mode_enabled()
	var game_feel_enabled := HeraSettings.get_game_feel_mode_enabled()
	return ToolResponse.success({
		"project_name": String(ProjectSettings.get_setting("application/config/name", "")),
		"project_path": ProjectSettings.globalize_path("res://"),
		"godot_version": String(Engine.get_version_info().get("string", "")),
		"scene": scene,
		"pid": OS.get_process_id(),
		"game_feel_ui_mode": game_feel_ui_enabled,
		"game_feel_mode": game_feel_enabled,
	})
