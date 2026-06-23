extends RefCounted

const ToolResponse = preload("res://addons/hera_agent_godot/core/tool_response.gd")

func get_name() -> String:
	return "output"

func execute(_request: Dictionary) -> Dictionary:
	return ToolResponse.failure("output: not implemented (skeleton)")
