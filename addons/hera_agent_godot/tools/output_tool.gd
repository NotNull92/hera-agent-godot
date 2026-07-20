extends RefCounted

# `output` — read the project log file (default user://logs/godot.log).
#
# Godot does not expose the editor Output panel / EditorLog to GDScript, so this
# reads the log file written when `debug/file_logging` is enabled. That setting
# is OFF by default; when it is, we report that and how to turn it on rather than
# pretending there is nothing to show.

const ToolResponse = preload("res://addons/hera_agent_godot/core/tool_response.gd")

func get_name() -> String:
	return "output"

func execute(params: Dictionary) -> Dictionary:
	# Read the effective value, not the base one: file logging defaults to true
	# on desktop via the `.pc` feature-tag override, and plain get_setting()
	# returns the untagged default (false). Keying availability off the base
	# value would report every desktop project as unreadable while it logs fine.
	var enabled := bool(ProjectSettings.get_setting_with_override("debug/file_logging/enable_file_logging"))
	var log_path := String(ProjectSettings.get_setting_with_override("debug/file_logging/log_path"))

	# Same blind spot as `diagnostics`: with file logging off, a stale log from an
	# earlier run still reads, so reporting `available` on file existence alone
	# would return an empty tail as if the project were quiet.
	if not enabled or not FileAccess.file_exists(log_path):
		var reason := "No log file yet." if enabled else "File logging is disabled."
		return ToolResponse.success({
			"available": false,
			"file_logging_enabled": enabled,
			"log_path": ProjectSettings.globalize_path(log_path),
			"hint": "%s This log only ever covers the running project — Godot installs no file logger in an editor session, so editor-console messages are never in it. Enable Project Settings > debug/file_logging/enable_file_logging for project runs, or relaunch the editor with --log-file <path> to capture editor output." % reason,
			"lines": [],
		})

	var max_lines := int(params.get("lines", 100))
	var type_filter := String(params.get("type", "all")).to_lower()
	var all_lines := FileAccess.get_file_as_string(log_path).split("\n", false)

	var filtered: Array = []
	for line in all_lines:
		if _matches(line, type_filter):
			filtered.append(line)

	var start: int = max(0, filtered.size() - max_lines)
	return ToolResponse.success({
		"available": true,
		"log_path": ProjectSettings.globalize_path(log_path),
		"type": type_filter,
		"total": filtered.size(),
		"lines": filtered.slice(start, filtered.size()),
	})

func _matches(line: String, type_filter: String) -> bool:
	match type_filter:
		"log":
			return line.find("ERROR") == -1 and line.find("WARNING") == -1
		"error":
			return line.find("ERROR") != -1
		"warning":
			return line.find("WARNING") != -1
		"all":
			return true
		_:
			return false
