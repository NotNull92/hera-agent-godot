extends RefCounted

static func coerce(raw: Variant, prop_info: Dictionary) -> Dictionary:
	var target_type := int(prop_info.get("type", TYPE_NIL))
	if typeof(raw) != TYPE_STRING:
		return { "ok": true, "value": raw }
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
			return { "ok": true, "value": NodePath(s) }
		TYPE_NIL:
			if s == "null":
				return { "ok": true, "value": null }
			return { "ok": false, "error": "cannot infer type for null property: %s" % String(prop_info.get("name", "")) }
		TYPE_OBJECT:
			return { "ok": false, "error": "unsupported object/resource property: %s" % String(prop_info.get("name", "")) }
		_:
			var parsed: Variant = str_to_var(s)
			if parsed == null and s != "null":
				return { "ok": false, "error": "invalid %s value for property: %s" % [type_string(target_type), s] }
			if typeof(parsed) != target_type:
				return { "ok": false, "error": "property expects %s, got %s" % [type_string(target_type), type_string(typeof(parsed))] }
			return { "ok": true, "value": parsed }

static func properties(node: Node, max_value_len: int) -> Dictionary:
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
			text = text.substr(0, max_value_len) + "…"
		result[pname] = text
	return result

static func selected_properties(node: Node, names: Array, max_value_len: int) -> Dictionary:
	var result: Dictionary = {}
	for raw_name in names:
		var name := String(raw_name)
		var value := property_value(node, name, max_value_len)
		if not bool(value.get("ok", false)):
			return value
		result[name] = String(value.get("value", ""))
	return { "ok": true, "properties": result }

static func property_value(node: Node, name: String, max_value_len: int) -> Dictionary:
	if name == "":
		return { "ok": false, "error": "property name is empty" }
	for prop in node.get_property_list():
		if String(prop.get("name", "")) == name:
			var text := str(node.get(name))
			if text.length() > max_value_len:
				text = text.substr(0, max_value_len) + "…"
			return { "ok": true, "value": text }
	return { "ok": false, "error": "node has no property: %s" % name }
