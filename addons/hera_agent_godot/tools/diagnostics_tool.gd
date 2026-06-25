extends RefCounted

const ToolResponse = preload("res://addons/hera_agent_godot/core/tool_response.gd")

func get_name() -> String:
	return "diagnostics"

func execute(params: Dictionary) -> Dictionary:
	var enabled := bool(ProjectSettings.get_setting("debug/file_logging/enable_file_logging", false))
	var log_path := String(ProjectSettings.get_setting("debug/file_logging/log_path", "user://logs/godot.log"))
	var max_lines: int = max(1, int(params.get("lines", 20)))
	var absolute_log_path := ProjectSettings.globalize_path(log_path)

	if not FileAccess.file_exists(log_path):
		return ToolResponse.success({
			"available": false,
			"file_logging_enabled": enabled,
			"log_path": absolute_log_path,
			"clean": true,
			"error_count": 0,
			"warning_count": 0,
			"errors": [],
			"warnings": [],
			"hint": "No log file. Enable Project Settings > debug/file_logging/enable_file_logging (restart the editor) to capture diagnostics.",
		})

	var all_lines := FileAccess.get_file_as_string(log_path).split("\n", false)
	var errors: Array = []
	var warnings: Array = []
	for line in all_lines:
		var upper := String(line).to_upper()
		if upper.find("ERROR") != -1:
			errors.append(line)
		elif upper.find("WARNING") != -1:
			warnings.append(line)

	return ToolResponse.success({
		"available": true,
		"file_logging_enabled": enabled,
		"log_path": absolute_log_path,
		"clean": errors.is_empty() and warnings.is_empty(),
		"total_lines": all_lines.size(),
		"error_count": errors.size(),
		"warning_count": warnings.size(),
		"errors": _tail(errors, max_lines),
		"warnings": _tail(warnings, max_lines),
	})

func _tail(lines: Array, max_lines: int) -> Array:
	var start: int = max(0, lines.size() - max_lines)
	return lines.slice(start, lines.size())
