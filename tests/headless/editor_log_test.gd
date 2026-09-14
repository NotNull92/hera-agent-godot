extends SceneTree

const OutputTool = preload("res://addons/hera_agent_godot/tools/output_tool.gd")
const DiagnosticsTool = preload("res://addons/hera_agent_godot/tools/diagnostics_tool.gd")
const EditorLog = preload("res://addons/hera_agent_godot/core/editor_log.gd")

func _initialize() -> void:
	var failures: Array[String] = []
	for tool in [OutputTool.new(), DiagnosticsTool.new()]:
		var data: Dictionary = tool.execute({"source": "editor"}).get("data", {})
		if data.get("reason", "") != "evidence_unavailable" or data.get("available", true) or data.get("clean", true):
			failures.append("unregistered editor source must explicitly report unavailable")
		if tool.execute({"since": "old:1"}).get("ok", true):
			failures.append("file source must reject editor cursors")
	if ClassDB.class_exists("Logger"):
		var output: Array = []
		var code := OS.execute(OS.get_executable_path(), ["--headless", "--path", ProjectSettings.globalize_path("res://"), "--script", "res://tests/headless/editor_log_fixture.gd"], output, true)
		var evidence: Variant = JSON.parse_string(FileAccess.get_file_as_string("res://editor-log-evidence.json"))
		if code != 0 or typeof(evidence) != TYPE_DICTIONARY:
			failures.append("editor logger fixture failed: %s" % str(output))
		elif not evidence.get("failures", []).is_empty():
			failures.append("editor logger fixture: %s" % str(evidence))
		DirAccess.remove_absolute(ProjectSettings.globalize_path("res://editor-log-evidence.json"))
	else:
		var collector := EditorLog.new()
		collector.start("unsupported")
		if collector.capability != "unsupported" or collector.read({}, false).get("data", {}).get("available", true):
			failures.append("absent API must not compile or register the adapter")
	for failure in failures:
		push_error(failure)
	quit(0 if failures.is_empty() else 1)
