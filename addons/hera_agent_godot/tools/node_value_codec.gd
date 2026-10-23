extends RefCounted

const ValueCodec = preload("res://addons/hera_agent_godot/core/value_codec.gd")

static func coerce(raw: Variant, prop_info: Dictionary) -> Dictionary:
	return ValueCodec.coerce(raw, prop_info, {"reject_hint": true})

static func properties(node: Node, max_value_len: int, changed: bool = false) -> Dictionary:
	var result: Dictionary = ValueCodec.properties(node, max_value_len)
	if not changed:
		return result
	var script: Script = node.get_script() as Script
	var script_properties: Dictionary = {}
	if script != null:
		for prop: Dictionary in script.get_script_property_list():
			script_properties[String(prop.get("name", ""))] = true
	var class_default: Object = null
	for name: String in result.keys():
		var default_value: Variant
		if node.property_can_revert(name):
			default_value = node.property_get_revert(name)
		elif script_properties.has(name):
			# shortcut: 4.2 tool-script defaults may be uncached; use the engine result until it exposes availability.
			default_value = script.get_property_default_value(name)
		elif ClassDB.has_method("class_get_property_default_value"):
			default_value = ClassDB.call("class_get_property_default_value", node.get_class(), name)
		else:
			# Godot 4.2 has no ClassDB property-default query.
			if class_default == null and ClassDB.can_instantiate(node.get_class()):
				class_default = ClassDB.instantiate(node.get_class())
			if class_default == null:
				continue
			default_value = class_default.get(name)
		var current_value: Variant = node.get(name)
		var current_type: int = typeof(current_value)
		var default_type: int = typeof(default_value)
		var comparable: bool = current_type == default_type or (current_type in [TYPE_INT, TYPE_FLOAT] and default_type in [TYPE_INT, TYPE_FLOAT])
		if comparable and current_value == default_value:
			result.erase(name)
	if class_default != null:
		class_default.free()
	return result

static func selected_properties(node: Node, names: Array, max_value_len: int) -> Dictionary:
	return ValueCodec.selected_properties(node, names, ValueCodec.property_value.bind(max_value_len))

static func property_value(node: Node, name: String, max_value_len: int) -> Dictionary:
	return ValueCodec.property_value(node, name, max_value_len)

static func prepare_properties(node: Node, raw: Variant) -> Dictionary:
	if not raw is Dictionary:
		return {"ok": false, "error": "props must be an object"}
	var values: Dictionary = {}
	for name: String in raw:
		var info: Dictionary = {}
		for property: Dictionary in node.get_property_list():
			if String(property.get("name", "")) == name:
				info = property
				break
		if info.is_empty():
			return {"ok": false, "error": "node has no property: %s" % name}
		var coerced: Dictionary = coerce(raw[name], info)
		if not coerced.get("ok", false):
			return coerced
		values[name] = coerced.get("value")
	return {"ok": true, "values": values}

static func read_values(node: Node, values: Dictionary) -> Dictionary:
	var result: Dictionary = {}
	for name: String in values:
		result[name] = str(node.get(name))
	return result
