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
			var oname := String(prop_info.get("name", ""))
			return { "ok": false, "error": "object/resource property: %s — set it with `node set-resource <path> --prop %s --resource res://...`" % [oname, oname] }
		_:
			var parsed: Variant = str_to_var(s)
			if parsed == null and s != "null":
				return { "ok": false, "error": "invalid %s value for property: %s%s" % [type_string(target_type), s, _hint_suffix(target_type)] }
			if typeof(parsed) != target_type:
				return { "ok": false, "error": "property expects %s, got %s%s" % [type_string(target_type), type_string(typeof(parsed)), _hint_suffix(target_type)] }
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

# Appended to a coercion error so an agent can fix the value from the first
# failure. Complex properties are parsed with the engine's own str_to_var, so
# the accepted form is Godot variant text — the same text a .tscn stores.
static func _hint_suffix(target_type: int) -> String:
	var hint := _syntax_hint(target_type)
	if hint != "":
		return " — use Godot variant text, e.g. %s" % hint
	return " — use Godot variant text (the form var_to_str() produces)"

static func _syntax_hint(target_type: int) -> String:
	match target_type:
		TYPE_VECTOR2:
			return "Vector2(x, y) like Vector2(120, 200)"
		TYPE_VECTOR2I:
			return "Vector2i(x, y)"
		TYPE_VECTOR3:
			return "Vector3(x, y, z)"
		TYPE_VECTOR3I:
			return "Vector3i(x, y, z)"
		TYPE_VECTOR4:
			return "Vector4(x, y, z, w)"
		TYPE_VECTOR4I:
			return "Vector4i(x, y, z, w)"
		TYPE_RECT2:
			return "Rect2(x, y, w, h)"
		TYPE_RECT2I:
			return "Rect2i(x, y, w, h)"
		TYPE_COLOR:
			return "Color(r, g, b, a) like Color(0.3, 0.8, 1, 1)"
		TYPE_ARRAY:
			return "[a, b, c]"
		TYPE_DICTIONARY:
			return "{\"key\": value}"
		TYPE_PACKED_BYTE_ARRAY:
			return "PackedByteArray(1, 2, 3)"
		TYPE_PACKED_INT32_ARRAY:
			return "PackedInt32Array(1, 2, 3)"
		TYPE_PACKED_INT64_ARRAY:
			return "PackedInt64Array(1, 2, 3)"
		TYPE_PACKED_FLOAT32_ARRAY:
			return "PackedFloat32Array(1, 2)"
		TYPE_PACKED_FLOAT64_ARRAY:
			return "PackedFloat64Array(1, 2)"
		TYPE_PACKED_STRING_ARRAY:
			return "PackedStringArray(\"a\", \"b\")"
		TYPE_PACKED_VECTOR2_ARRAY:
			return "PackedVector2Array(x1, y1, x2, y2, …) — a flat number list"
		TYPE_PACKED_VECTOR3_ARRAY:
			return "PackedVector3Array(x1, y1, z1, …) — a flat number list"
		TYPE_PACKED_COLOR_ARRAY:
			return "PackedColorArray(Color(1, 1, 1, 1))"
	return ""
