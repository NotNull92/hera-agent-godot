extends RefCounted

const ValueCodec = preload("res://addons/hera_agent_godot/core/value_codec.gd")

static func coerce(raw: Variant, prop_info: Dictionary) -> Dictionary:
	return ValueCodec.coerce(raw, prop_info, {"reject_hint": true})

static func properties(node: Node, max_value_len: int) -> Dictionary:
	return ValueCodec.properties(node, max_value_len)

static func selected_properties(node: Node, names: Array, max_value_len: int) -> Dictionary:
	return ValueCodec.selected_properties(node, names, property_value.bind(max_value_len))

static func property_value(node: Node, name: String, max_value_len: int) -> Dictionary:
	return ValueCodec.property_value(node, name, max_value_len)
