extends SceneTree

const ScriptTool = preload("res://addons/hera_agent_godot/tools/script_tool.gd")


func _initialize() -> void:
	var tool := ScriptTool.new()
	var failures: Array[String] = []
	for path in ["res://../probe.gd", "probe.gd", "res://missing_validation_probe.gd", "res://Probe.cs"]:
		var rejected: Dictionary = tool.execute({"action": "validate-context", "path": path})
		if bool(rejected.get("ok", false)):
			failures.append("accepted invalid validation path: %s" % path)
	var response: Dictionary = tool.execute({"action": "validate-context", "path": "res://tests/headless/script_validation_test.gd"})
	var data: Dictionary = response.get("data", {})
	if not bool(response.get("ok", false)) or String(data.get("executable", "")) != OS.get_executable_path() or String(data.get("project_path", "")) != ProjectSettings.globalize_path("res://"):
		failures.append("validation context did not identify current engine/project")
	for failure in failures:
		push_error(failure)
	quit(0 if failures.is_empty() else 1)
