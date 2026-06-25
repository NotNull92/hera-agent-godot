extends RefCounted

const MAX_VALUE_LEN := 200

static func properties(node: Node) -> Dictionary:
	var result := {}
	for prop in node.get_property_list():
		var usage := int(prop.get("usage", 0))
		if not (usage & PROPERTY_USAGE_EDITOR):
			continue
		if usage & (PROPERTY_USAGE_CATEGORY | PROPERTY_USAGE_GROUP | PROPERTY_USAGE_SUBGROUP):
			continue
		var pname := String(prop.get("name", ""))
		if pname == "":
			continue
		var value := str(node.get(pname))
		if value.length() > MAX_VALUE_LEN:
			value = value.substr(0, MAX_VALUE_LEN) + "..."
		result[pname] = value
	return result

static func selected_properties(node: Node, names: Array) -> Dictionary:
	var result := {}
	for raw_name in names:
		var name := String(raw_name)
		var value := property_value(node, name)
		if not bool(value.get("ok", false)):
			return value
		result[name] = String(value.get("value", ""))
	return { "ok": true, "properties": result }

static func property_value(node: Node, name: String) -> Dictionary:
	if name.begins_with("metadata/"):
		var meta_name := name.substr("metadata/".length())
		if not node.has_meta(meta_name):
			return { "ok": false, "error": "node has no metadata: %s" % meta_name }
		var meta_value: Variant = node.get_meta(meta_name)
		return { "ok": true, "value": str(meta_value), "type": type_string(typeof(meta_value)) }
	for prop in node.get_property_list():
		if String(prop.get("name", "")) == name:
			var prop_value: Variant = node.get(name)
			return { "ok": true, "value": str(prop_value), "type": type_string(typeof(prop_value)) }
	return { "ok": false, "error": "node has no property: %s" % name }

static func coerce(raw: Variant, prop_info: Dictionary) -> Dictionary:
	var target_type := int(prop_info.get("type", TYPE_NIL))
	if typeof(raw) != TYPE_STRING:
		return { "ok": true, "value": raw }
	var text := String(raw)
	match target_type:
		TYPE_STRING:
			return { "ok": true, "value": text }
		TYPE_STRING_NAME:
			return { "ok": true, "value": StringName(text) }
		TYPE_BOOL:
			return _coerce_bool(text)
		TYPE_INT:
			if not text.is_valid_int():
				return { "ok": false, "error": "invalid int value for property: %s" % text }
			return { "ok": true, "value": int(text) }
		TYPE_FLOAT:
			if not text.is_valid_float():
				return { "ok": false, "error": "invalid float value for property: %s" % text }
			return { "ok": true, "value": float(text) }
		TYPE_NODE_PATH:
			return { "ok": true, "value": NodePath(text) }
		TYPE_NIL:
			if text == "null":
				return { "ok": true, "value": null }
			return { "ok": false, "error": "cannot infer type for null property: %s" % String(prop_info.get("name", "")) }
		TYPE_OBJECT:
			return { "ok": false, "error": "unsupported object/resource property: %s" % String(prop_info.get("name", "")) }
		_:
			return _coerce_variant(text, target_type)

static func call_args(request: Dictionary) -> Array:
	var raw_args: Variant = request.get("args", [])
	if typeof(raw_args) != TYPE_ARRAY:
		return []
	var result: Array = []
	for raw in raw_args:
		result.append(call_arg(raw))
	return result

static func call_arg(raw: Variant) -> Variant:
	if typeof(raw) != TYPE_STRING:
		return raw
	var text := String(raw)
	var parsed: Variant = str_to_var(text)
	if parsed == null and text != "null":
		return text
	return parsed

static func _coerce_bool(text: String) -> Dictionary:
	var lower := text.to_lower()
	if lower == "true" or lower == "1":
		return { "ok": true, "value": true }
	if lower == "false" or lower == "0":
		return { "ok": true, "value": false }
	return { "ok": false, "error": "invalid bool value for property: %s" % text }

static func _coerce_variant(text: String, target_type: int) -> Dictionary:
	var parsed: Variant = str_to_var(text)
	if parsed == null and text != "null":
		return { "ok": false, "error": "invalid %s value for property: %s" % [type_string(target_type), text] }
	if typeof(parsed) != target_type:
		return { "ok": false, "error": "property expects %s, got %s" % [type_string(target_type), type_string(typeof(parsed))] }
	return { "ok": true, "value": parsed }
