extends RefCounted

const ToolResponse = preload("res://addons/hera_agent_godot/core/tool_response.gd")

func get_name() -> String:
	return "status"

func execute(_params: Dictionary) -> Dictionary:
	var root := EditorInterface.get_edited_scene_root()
	var scene := ""
	if root != null:
		scene = root.scene_file_path
	return ToolResponse.success({
		"project_name": String(ProjectSettings.get_setting("application/config/name", "")),
		"project_path": ProjectSettings.globalize_path("res://"),
		"godot_version": String(Engine.get_version_info().get("string", "")),
		"scene": scene,
		"pid": OS.get_process_id(),
	})
