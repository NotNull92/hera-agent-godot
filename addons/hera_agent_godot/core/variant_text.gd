extends RefCounted

# Appended to a coercion error so an agent can fix the value from the first
# failure. Complex properties are parsed with the engine's own str_to_var, so
# the accepted form is Godot variant text — the same text a .tscn stores.
static func hint_suffix(target_type: int) -> String:
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
