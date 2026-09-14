extends RefCounted

const ToolResponse = preload("res://addons/hera_agent_godot/core/tool_response.gd")
const HeraSettings = preload("res://addons/hera_agent_godot/core/hera_settings.gd")

var editor_session_id: String = Crypto.new().generate_random_bytes(16).hex_encode()
var editor_log: RefCounted


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
		"godot_commit": String(Engine.get_version_info().get("hash", "")),
		"editor_session_id": editor_session_id,
		"capabilities": _capabilities(),
		"scene": scene,
		"pid": OS.get_process_id(),
		"game_feel_ui_mode": game_feel_ui_enabled,
		"game_feel_mode": game_feel_enabled,
		"csharp_supported": ClassDB.class_exists("CSharpScript"),
	})


func _capabilities() -> Dictionary:
	return {
		"editor_mutation_gate": "supported",
		"linked_evidence": "supported",
		"import_busy_api": _support(ClassDB.class_has_method("EditorFileSystem", "is_importing")),
		"operation_receipts": "supported",
		"node_set_guard": "supported",
		"editor_log_cursor": editor_log.capability if editor_log != null else "unverified",
		"editor_logger_api": _support(ClassDB.class_exists("Logger") and OS.has_method("add_logger") and OS.has_method("remove_logger")),
		"debugger_messages_api": _support(ClassDB.class_has_method("EditorPlugin", "add_debugger_plugin") and ClassDB.class_has_method("EditorDebuggerSession", "send_message") and EngineDebugger.has_method("register_message_capture") and EngineDebugger.has_method("send_message")),
		"dap": "unverified",
		"script_metadata_api": _support(ClassDB.class_has_method("Script", "get_script_method_list")),
		"script_symbol_lookup": "unverified",
		"global_class_lookup_api": _support(ProjectSettings.has_method("get_global_class_list")),
		"import_state_api": _support(ClassDB.class_has_method("EditorFileSystem", "is_scanning") and ClassDB.class_has_method("EditorFileSystem", "get_scanning_progress")),
		"render_frame_event_api": _support(RenderingServer.has_signal("frame_post_draw")),
		"csharp": _support(ClassDB.class_exists("CSharpScript")),
		"dotnet_sdk": "unverified",
	}


func _support(available: bool) -> String:
	return "supported" if available else "unsupported"
