extends RefCounted

const ToolResponse = preload("res://addons/hera_agent_godot/core/tool_response.gd")

func get_name() -> String:
	return "diagnostics"

func execute(params: Dictionary) -> Dictionary:
	# Read the effective value, not the base one: file logging defaults to true
	# on desktop via the `.pc` feature-tag override, and plain get_setting()
	# returns the untagged default (false). Keying availability off the base
	# value would report every desktop project as unreadable while it logs fine.
	var enabled := bool(ProjectSettings.get_setting_with_override("debug/file_logging/enable_file_logging"))
	var log_path := String(ProjectSettings.get_setting_with_override("debug/file_logging/log_path"))
	var max_lines: int = max(1, int(params.get("lines", 20)))
	var absolute_log_path := ProjectSettings.globalize_path(log_path)

	# Diagnostics are observable only while the engine is actually writing the
	# log. Checking for the file alone is not enough: a stale log left by an
	# earlier run still parses, so with file logging off this would report an
	# up-to-date, clean project while nothing at all is being captured — the
	# console can be full of errors and this tool would never see one.
	#
	# `clean` says false rather than true in that state on purpose. It cannot be
	# asserted, and a false "all clear" is the failure worth preventing; pair it
	# with `available` and `hint` to tell "cannot see" apart from "saw problems".
	if not enabled or not FileAccess.file_exists(log_path):
		var reason := "No log file yet." if enabled else "File logging is disabled."
		return ToolResponse.success({
			"available": false,
			"file_logging_enabled": enabled,
			"log_path": absolute_log_path,
			"clean": false,
			"error_count": 0,
			"warning_count": 0,
			"errors": [],
			"warnings": [],
			"hint": "%s This log only ever covers the running project — Godot installs no file logger in an editor session, so editor-console messages are never in it. Enable Project Settings > debug/file_logging/enable_file_logging for project runs, or relaunch the editor with --log-file <path> to capture editor output." % reason,
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
