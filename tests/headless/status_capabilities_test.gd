# Hera test: editor
extends SceneTree

const StatusTool = preload("res://addons/hera_agent_godot/tools/status_tool.gd")
const Heartbeat = preload("res://addons/hera_agent_godot/server/heartbeat.gd")

func _initialize() -> void:
	call_deferred("_run")

func _run() -> void:
	var failures: Array[String] = []
	var tool := StatusTool.new()
	if tool._support(false) != "unsupported" or tool._support(true) != "supported":
		failures.append("API presence must distinguish supported and unsupported")
	var data: Dictionary = tool.execute({}).get("data", {})
	var session := String(data.get("editor_session_id", ""))
	if session.length() != 32 or not session.is_valid_hex_number():
		failures.append("status must expose a random 128-bit editor session")
	if data.get("godot_commit", "") != Engine.get_version_info().get("hash", ""):
		failures.append("status must report the running engine commit")
	var capabilities: Dictionary = data.get("capabilities", {})
	if capabilities.get("editor_mutation_gate") != "supported" or capabilities.get("import_busy_api") != tool._support(ClassDB.class_has_method("EditorFileSystem", "is_importing")):
		failures.append("gate support must not imply external import-state API availability")
	for key in ["editor_logger_api", "editor_log_cursor", "debugger_messages_api", "dap", "script_metadata_api", "script_symbol_lookup", "global_class_lookup_api", "import_state_api", "render_frame_event_api", "csharp", "dotnet_sdk"]:
		if not ["supported", "unsupported", "unverified"].has(capabilities.get(key, "")):
			failures.append("missing capability state: " + key)
	if capabilities.get("csharp", "") != ("supported" if ClassDB.class_exists("CSharpScript") else "unsupported"):
		failures.append("C# capability must follow the actual engine class")
	if capabilities.get("dap", "") != "unverified" or capabilities.get("dotnet_sdk", "") != "unverified":
		failures.append("status must not infer DAP or SDK readiness from engine APIs")
	if not session.is_empty():
		if tool.execute({}).get("data", {}).get("editor_session_id", "") != session:
			failures.append("status session changed during a tool lifetime")
		if StatusTool.new().execute({}).get("data", {}).get("editor_session_id", "") == session:
			failures.append("a new addon lifetime reused a session")
		var heartbeat := Heartbeat.new()
		heartbeat.call("start", 8770, session)
		var path: String = heartbeat._path
		var advertised: Dictionary = JSON.parse_string(FileAccess.get_file_as_string(path))
		if advertised.get("editor_session_id", "") != session:
			failures.append("heartbeat and status must advertise the same session")
		heartbeat.stop()
	for failure in failures:
		push_error(failure)
	quit(0 if failures.is_empty() else 1)
