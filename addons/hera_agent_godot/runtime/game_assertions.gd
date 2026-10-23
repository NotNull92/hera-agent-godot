extends RefCounted

const GameValueCodec = preload("res://addons/hera_agent_godot/runtime/game_value_codec.gd")

static func check(node: Node, request: Dictionary) -> Dictionary:
	var prop := String(request.get("prop", ""))
	var op := String(request.get("op", ""))
	var actual := GameValueCodec.property_value(node, prop)
	if not bool(actual.get("ok", false)):
		return { "ok": false, "error": String(actual.get("error", "property not found")) }
	var actual_value := String(actual.get("value", ""))
	var expected_raw: Variant = request.get("value", "")
	var expected_value: String = str(expected_raw)
	var matches: bool = _matches(op, actual_value, expected_value)
	if (op == "eq" or op == "ne") and (typeof(expected_raw) == TYPE_INT or typeof(expected_raw) == TYPE_FLOAT) and actual_value.is_valid_float():
		var numeric_match: bool = float(actual_value) == float(expected_raw)
		matches = numeric_match if op == "eq" else not numeric_match
	if not matches:
		return {
			"ok": false,
			"prop": prop, "op": op, "actual": actual_value, "expected": expected_value,
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

static func wait_error(request: Dictionary) -> String:
	if not request.has("within_ms"):
		return ""
	var value: Variant = request["within_ms"]
	if typeof(value) not in [TYPE_INT, TYPE_FLOAT] or not is_finite(float(value)) or float(value) != floor(float(value)) or float(value) < 1 or float(value) > 30000:
		return "within_ms must be between 1 and 30000"
	return ""
