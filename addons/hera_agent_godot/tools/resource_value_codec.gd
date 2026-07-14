extends RefCounted

static func apply_props(res: Resource, raw_props: Variant) -> Dictionary:
	if typeof(raw_props) != TYPE_DICTIONARY:
		return { "ok": false, "error": "props must be an object" }
	var set_props := {}
	var props: Dictionary = raw_props
	for raw_name in props.keys():
		var prop_name := String(raw_name)
		var prop_info := _property_info(res, prop_name)
		if prop_info.is_empty():
			return { "ok": false, "error": "resource has no property: %s" % prop_name }
		var coerced := _coerce(props[raw_name], prop_info)
		if not bool(coerced.get("ok", false)):
			return { "ok": false, "error": String(coerced.get("error", "invalid property value")) }
		var value: Variant = coerced.get("value")
		res.set(prop_name, value)
		set_props[prop_name] = str(res.get(prop_name))
	return { "ok": true, "properties": set_props }

static func _property_info(res: Resource, prop: String) -> Dictionary:
	for p in res.get_property_list():
		if String(p.get("name", "")) == prop:
			return p
	return {}

static func _coerce(raw: Variant, prop_info: Dictionary) -> Dictionary:
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
			var lower := text.to_lower()
			if lower == "true" or lower == "1":
				return { "ok": true, "value": true }
			if lower == "false" or lower == "0":
				return { "ok": true, "value": false }
			return { "ok": false, "error": "invalid bool value for property: %s" % text }
		TYPE_INT:
			if not text.is_valid_int():
				return { "ok": false, "error": "invalid int value for property: %s" % text }
			return { "ok": true, "value": int(text) }
		TYPE_FLOAT:
			if not text.is_valid_float():
				return { "ok": false, "error": "invalid float value for property: %s" % text }
			return { "ok": true, "value": float(text) }
		TYPE_OBJECT:
			return _coerce_resource_object(text, prop_info)
		TYPE_NIL:
			if text == "null":
				return { "ok": true, "value": null }
			return { "ok": false, "error": "cannot infer type for null property: %s" % String(prop_info.get("name", "")) }
		_:
			var parsed: Variant = str_to_var(text)
			if parsed == null and text != "null":
				return { "ok": false, "error": "invalid %s value for property: %s%s" % [type_string(target_type), text, _hint_suffix(target_type)] }
			if typeof(parsed) != target_type:
				return { "ok": false, "error": "property expects %s, got %s%s" % [type_string(target_type), type_string(typeof(parsed)), _hint_suffix(target_type)] }
			return { "ok": true, "value": parsed }

static func _coerce_resource_object(text: String, prop_info: Dictionary) -> Dictionary:
	if text == "null":
		return { "ok": true, "value": null }
	if not (text.begins_with("res://") or text.begins_with("user://")):
		return { "ok": false, "error": "object property expects a resource path or null: %s" % String(prop_info.get("name", "")) }
	if text.begins_with("res://") and not _is_safe_res_path(text):
		return { "ok": false, "error": "resource path must stay inside res://" }
	if not ResourceLoader.exists(text):
		return { "ok": false, "error": "resource not found: %s" % text }
	var loaded := ResourceLoader.load(text)
	if loaded == null or not (loaded is Resource):
		return { "ok": false, "error": "not a resource: %s" % text }
	var expected := String(prop_info.get("class_name", ""))
	if expected != "" and not loaded.is_class(expected):
		return { "ok": false, "error": "resource type %s is not compatible with property %s (%s)" % [loaded.get_class(), String(prop_info.get("name", "")), expected] }
	return { "ok": true, "value": loaded }

static func _is_safe_res_path(path: String) -> bool:
	if path.find("\\") != -1:
		return false
	var rel := path.substr("res://".length())
	if rel == "" or rel.begins_with("/"):
		return false
	for part in rel.split("/", true):
		if part == "" or part == "." or part == "..":
			return false
	return true

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
