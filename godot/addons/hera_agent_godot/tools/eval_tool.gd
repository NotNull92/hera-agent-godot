extends RefCounted

const ToolResponse = preload("res://addons/hera_agent_godot/core/tool_response.gd")

func get_name() -> String:
	return "eval"

func execute(_request: Dictionary) -> Dictionary:
	return ToolResponse.failure("eval: not implemented (skeleton)")
