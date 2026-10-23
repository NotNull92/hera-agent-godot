extends RefCounted

const VariantText = preload("res://addons/hera_agent_godot/core/variant_text.gd")

static func coerce(raw: Variant, prop_info: Dictionary, options: Dictionary = {}) -> Dictionary:
	var target_type := int(prop_info.get("type", TYPE_NIL))
	if typeof(raw) != TYPE_STRING:
		if not bool(options.get("numeric", false)):
			return { "ok": true, "value": raw }
		if typeof(raw) == target_type or (raw == null and target_type == TYPE_OBJECT):
			return { "ok": true, "value": raw }
		if typeof(raw) == TYPE_INT and target_type == TYPE_FLOAT:
			return { "ok": true, "value": float(raw) }
		if typeof(raw) == TYPE_FLOAT and target_type == TYPE_INT and is_finite(raw) and raw >= -9223372036854775808.0 and raw < 9223372036854775808.0 and raw == float(int(raw)):
			return { "ok": true, "value": int(raw) }
		return { "ok": false, "error": "property expects %s, got %s%s" % [type_string(target_type), type_string(typeof(raw)), VariantText.hint_suffix(target_type)] }
	var s := String(raw)
	match target_type:
		TYPE_STRING:
			return { "ok": true, "value": s }
		TYPE_STRING_NAME:
			return { "ok": true, "value": StringName(s) }
		TYPE_BOOL:
			var lower := s.to_lower()
			if lower == "true" or lower == "1":
				return { "ok": true, "value": true }
			if lower == "false" or lower == "0":
				return { "ok": true, "value": false }
			return { "ok": false, "error": "invalid bool value for property: %s" % s }
		TYPE_INT:
			if not s.is_valid_int():
				return { "ok": false, "error": "invalid int value for property: %s" % s }
			return { "ok": true, "value": int(s) }
		TYPE_FLOAT:
			if not s.is_valid_float():
				return { "ok": false, "error": "invalid float value for property: %s" % s }
			return { "ok": true, "value": float(s) }
		TYPE_NODE_PATH:
			if bool(options.get("parse_node_path", false)):
				return _coerce_variant(s, target_type)
			return { "ok": true, "value": NodePath(s) }
		TYPE_NIL:
			if s == "null":
				return { "ok": true, "value": null }
			return { "ok": false, "error": "cannot infer type for null property: %s" % String(prop_info.get("name", "")) }
		TYPE_OBJECT:
			if options.has("load_resource"):
				var loader: Callable = options["load_resource"]
				return loader.call(s, prop_info)
			var oname := String(prop_info.get("name", ""))
			if bool(options.get("reject_hint", false)):
				return { "ok": false, "error": "object/resource property: %s — set it with `node set-resource <path> --prop %s --resource res://...`" % [oname, oname] }
			return { "ok": false, "error": "unsupported object/resource property: %s" % oname }
		_:
			return _coerce_variant(s, target_type)

static func properties(node: Node, max_value_len: int, suffix: String = "…") -> Dictionary:
	var result: Dictionary = {}
	for prop in node.get_property_list():
		var usage := int(prop.get("usage", 0))
		if not (usage & PROPERTY_USAGE_EDITOR):
			continue
		if usage & (PROPERTY_USAGE_CATEGORY | PROPERTY_USAGE_GROUP | PROPERTY_USAGE_SUBGROUP):
			continue
		var pname := String(prop.get("name", ""))
		if pname == "":
			continue
		var text := str(node.get(pname))
		if text.length() > max_value_len:
			text = text.substr(0, max_value_len) + suffix
		result[pname] = text
	return result

static func property_value(node: Node, name: String, max_value_len: int = -1) -> Dictionary:
	if name == "" and max_value_len >= 0:
		return { "ok": false, "error": "property name is empty" }
	for prop in node.get_property_list():
		if String(prop.get("name", "")) == name:
			var raw: Variant = node.get(name)
			if max_value_len < 0:
				return { "ok": true, "value": str(raw), "type": type_string(typeof(raw)) }
			var text := str(raw)
			if text.length() > max_value_len:
				text = text.substr(0, max_value_len) + "…"
			return { "ok": true, "value": text }
	return { "ok": false, "error": "node has no property: %s" % name }

static func _coerce_variant(text: String, target_type: int) -> Dictionary:
	var parsed: Variant = str_to_var(text)
	if parsed == null and text != "null":
		return { "ok": false, "error": "invalid %s value for property: %s%s" % [type_string(target_type), text, VariantText.hint_suffix(target_type)] }
	if typeof(parsed) != target_type:
		return { "ok": false, "error": "property expects %s, got %s%s" % [type_string(target_type), type_string(typeof(parsed)), VariantText.hint_suffix(target_type)] }
	return { "ok": true, "value": parsed }

static func selected_properties(node: Node, names: Array, read: Callable) -> Dictionary:
	var result: Dictionary = {}
	for raw_name in names:
		var name := String(raw_name)
		var value: Dictionary = read.call(node, name)
		if not bool(value.get("ok", false)):
			return value
		result[name] = String(value.get("value", ""))
	return { "ok": true, "properties": result }

