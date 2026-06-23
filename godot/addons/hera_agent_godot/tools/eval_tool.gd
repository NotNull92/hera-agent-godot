extends RefCounted

# `eval` — evaluate a single GDScript expression via the Expression class.
#
# Not full GDScript: one expression, no statements. The edited scene root is used
# as the base instance, so expressions can reach the scene, e.g.
# `get_node("Player").position`. Powerful (can call methods with side effects),
# hence it lives in the mutation phase.

const ToolResponse = preload("res://addons/hera_agent_godot/core/tool_response.gd")

func get_name() -> String:
	return "eval"

func execute(params: Dictionary) -> Dictionary:
	var text := String(params.get("expr", ""))
	if text == "":
		return ToolResponse.failure("eval requires an 'expr' string")

	var expression := Expression.new()
	if expression.parse(text) != OK:
		return ToolResponse.failure("parse error: %s" % expression.get_error_text())

	var base: Object = EditorInterface.get_edited_scene_root()
	var result: Variant = expression.execute([], base, true)
	if expression.has_execute_failed():
		return ToolResponse.failure("execute error: %s" % expression.get_error_text())

	return ToolResponse.success({
		"result": str(result),
		"type": type_string(typeof(result)),
		"undoable": false,
	})
