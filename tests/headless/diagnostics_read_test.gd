extends SceneTree

const DiagnosticsTool = preload("res://addons/hera_agent_godot/tools/diagnostics_tool.gd")
const OutputTool = preload("res://addons/hera_agent_godot/tools/output_tool.gd")

func _initialize() -> void:
	var failures: Array[String] = []
	var unreadable := OS.get_environment("HERA_QA_UNREADABLE_LOG")
	if unreadable.is_empty():
		unreadable = "res://missing-diagnostics-log"
	ProjectSettings.set_setting("debug/file_logging/enable_file_logging", true)
	ProjectSettings.set_setting("debug/file_logging/enable_file_logging.pc", true)
	ProjectSettings.set_setting("debug/file_logging/log_path", unreadable)
	for tool in [DiagnosticsTool.new(), OutputTool.new()]:
		var response: Dictionary = tool.execute({})
		var data: Dictionary = response.get("data", {})
		if not bool(response.get("ok", false)) or bool(data.get("available", true)):
			failures.append("unreadable log reported available: %s" % tool.get_name())
		if tool.get_name() == "diagnostics" and bool(data.get("clean", true)):
			failures.append("unreadable diagnostics reported clean")
	var clean_path := "res://diagnostics-empty.log"
	var clean_file := FileAccess.open(clean_path, FileAccess.WRITE)
	clean_file.close()
	ProjectSettings.set_setting("debug/file_logging/log_path", clean_path)
	var clean_response: Dictionary = DiagnosticsTool.new().execute({})
	var clean_data: Dictionary = clean_response.get("data", {})
	if not bool(clean_data.get("available", false)) or not bool(clean_data.get("clean", false)):
		failures.append("readable empty log must be available and clean")
	DirAccess.remove_absolute(ProjectSettings.globalize_path(clean_path))
	for failure in failures:
		push_error(failure)
	quit(0 if failures.is_empty() else 1)
