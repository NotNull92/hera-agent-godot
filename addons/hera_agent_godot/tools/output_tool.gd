extends RefCounted

# `output` — read the project log file (default user://logs/godot.log).
#
# The default reads the configured file log; the opt-in editor source uses
# the plugin's registered Logger collector and never falls back to that file.

const ToolResponse = preload("res://addons/hera_agent_godot/core/tool_response.gd")
const EditorLog = preload("res://addons/hera_agent_godot/core/editor_log.gd")
var editor_log: RefCounted

func get_name() -> String:
	return "output"

func execute(params: Dictionary) -> Dictionary:
	if params.get("source", "file") == "editor":
		return editor_log.read(params, false) if editor_log != null else ToolResponse.success(EditorLog.unavailable())
	if params.get("source", "file") != "file" or params.has("since"):
		return ToolResponse.failure("source must be file or editor; since requires editor source")
	# Read the effective value, not the base one: file logging defaults to true
	# on desktop via the `.pc` feature-tag override, and plain get_setting()
	# returns the untagged default (false). Keying availability off the base
	# value would report every desktop project as unreadable while it logs fine.
	var enabled := bool(ProjectSettings.get_setting_with_override("debug/file_logging/enable_file_logging"))
	var log_path := String(ProjectSettings.get_setting_with_override("debug/file_logging/log_path"))

	# Same blind spot as `diagnostics`: with file logging off, a stale log from an
	# earlier run still reads, so reporting `available` on file existence alone
	# would return an empty tail as if the project were quiet.
	var log_file: FileAccess = FileAccess.open(log_path, FileAccess.READ) if enabled else null
	var log_bytes := PackedByteArray()
	var readable := false
	if log_file != null:
		var length := log_file.get_length()
		log_bytes = log_file.get_buffer(length)
		readable = log_bytes.size() == length and log_file.get_error() in [OK, ERR_FILE_EOF]
		log_file.close()
	if not readable:
		var reason := "Log file is missing or unreadable." if enabled else "File logging is disabled."
		return ToolResponse.success({
			"available": false,
			"source": "file",
			"reason": "evidence_unavailable",
			"file_logging_enabled": enabled,
			"log_path": ProjectSettings.globalize_path(log_path),
			"hint": "%s This log only ever covers the running project — Godot installs no file logger in an editor session, so editor-console messages are never in it. Enable Project Settings > debug/file_logging/enable_file_logging for project runs, or relaunch the editor with --log-file <path> to capture editor output." % reason,
			"lines": [],
		})

	var max_lines := int(params.get("lines", 100))
	var type_filter := String(params.get("type", "all")).to_lower()
	var all_lines := log_bytes.get_string_from_utf8().split("\n", false)

	var filtered: Array = []
	for line in all_lines:
		if _matches(line, type_filter):
			filtered.append(line)

	var start: int = max(0, filtered.size() - max_lines)
	return ToolResponse.success({
		"available": true,
		"source": "file",
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
