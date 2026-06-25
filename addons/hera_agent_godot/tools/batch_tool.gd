extends RefCounted

# `batch` — run several tool requests in one round-trip.
#
# Sequential, not transactional: commands run in order on the editor main thread.
# With stop_on_error (default true) the run halts at the first failure. Each
# mutation sub-command still registers its own undo step (so a batch is several
# undo steps, not one). Nesting `batch` inside `batch` is rejected.

const ToolResponse = preload("res://addons/hera_agent_godot/core/tool_response.gd")

var _registry

func set_registry(registry) -> void:
	_registry = registry

func get_name() -> String:
	return "batch"

func execute(params: Dictionary) -> Dictionary:
	return ToolResponse.failure("batch requires async dispatch")

func execute_async(params: Dictionary) -> Dictionary:
	var commands: Variant = params.get("commands", [])
	if typeof(commands) != TYPE_ARRAY:
		return ToolResponse.failure("batch requires a 'commands' array")
	var stop_on_error := bool(params.get("stop_on_error", true))
	var results: Array = []
	var stopped := false

	for entry in commands:
		var result := await _run_one(entry)
		results.append(result)
		if stop_on_error and not bool(result.get("ok", false)):
			stopped = true
			break

	return ToolResponse.success({ "count": results.size(), "stopped": stopped, "results": results })

func _run_one(entry: Variant) -> Dictionary:
	if typeof(entry) != TYPE_DICTIONARY:
		return { "ok": false, "error": "each command must be an object" }
	var tool_name := String(entry.get("tool", ""))
	if tool_name == "":
		return { "ok": false, "error": "command missing 'tool'" }
	if tool_name == "batch":
		return { "tool": "batch", "ok": false, "error": "batch cannot nest batch" }
	var tool = _registry.resolve(tool_name) if _registry != null else null
	if tool == null:
		return { "tool": tool_name, "ok": false, "error": "unknown tool: %s" % tool_name }
	var sub_params: Variant = entry.get("params", {})
	if typeof(sub_params) != TYPE_DICTIONARY:
		return { "tool": tool_name, "ok": false, "error": "params must be an object" }
	var result: Dictionary
	if tool.has_method("execute_async"):
		result = await tool.execute_async(sub_params)
	else:
		result = tool.execute(sub_params)
	result["tool"] = tool_name
	return result
