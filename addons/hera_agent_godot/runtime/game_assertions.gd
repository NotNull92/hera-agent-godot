extends RefCounted

const GameValueCodec = preload("res://addons/hera_agent_godot/runtime/game_value_codec.gd")

static func check(node: Node, request: Dictionary) -> Dictionary:
	var prop := String(request.get("prop", ""))
	var op := String(request.get("op", ""))
	var actual := GameValueCodec.property_value(node, prop)
	if not bool(actual.get("ok", false)):
		return { "ok": false, "error": String(actual.get("error", "property not found")) }
	var actual_value := String(actual.get("value", ""))
	var expected_value := String(request.get("value", ""))
	if not _matches(op, actual_value, expected_value):
		return {
			"ok": false,
			"error": "assert failed: %s %s %s (actual %s)" % [prop, op, expected_value, actual_value],
		}
	return {
		"ok": true,
		"prop": prop,
		"op": op,
		"actual": actual_value,
		"expected": expected_value,
	}

static func _matches(op: String, actual: String, expected: String) -> bool:
	match op:
		"eq":
			return actual == expected
		"ne":
			return actual != expected
		"contains":
			return actual.contains(expected)
		"exists":
			return true
		"gt", "lt":
			if not actual.is_valid_float() or not expected.is_valid_float():
				return false
			var actual_float := float(actual)
			var expected_float := float(expected)
			return actual_float > expected_float if op == "gt" else actual_float < expected_float
		_:
			return false
