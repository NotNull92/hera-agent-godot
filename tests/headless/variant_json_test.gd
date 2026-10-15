extends SceneTree

const VariantJson = preload("res://addons/hera_agent_godot/core/variant_json.gd")
var _failed := false


func _initialize() -> void:
	for value in [null, true, false, 0, 3, -3, 9007199254740991, -9007199254740991, 3.0, "text\n한글", '{"json":true}']:
		_expect(value, value)
	_expect(9007199254740993, "9007199254740993")
	_expect(-9223372036854775808, "-9223372036854775808")
	_expect(9223372036854775807, "9223372036854775807")
	_expect(NAN, "nan")
	_expect(INF, "inf")
	_expect(-INF, "inf_neg")
	_expect(&"name", "name")
	_expect(NodePath("Main/Child"), "Main/Child")
	_expect({"hp": 3, "pos": Vector2(1, 2), 7: [null, NAN, 9007199254740993]}, {"hp": 3, "pos": "Vector2(1, 2)", "7": [null, "nan", "9007199254740993"]})
	for value in [Vector2(1.5, 2), Vector2i(1, 2), Vector3(1, 2, 3), Vector3i(1, 2, 3), Vector4(1, 2, 3, 4), Vector4i(1, 2, 3, 4), Rect2(1, 2, 3, 4), Rect2i(1, 2, 3, 4), Color(0.25, 0.5, 0.75), Transform2D.IDENTITY, Transform3D.IDENTITY, Basis.IDENTITY, Quaternion.IDENTITY, Plane(Vector3.UP, 2), AABB(Vector3.ZERO, Vector3.ONE), Projection.IDENTITY, RID(), Callable(self, "_initialize"), Signal(self, "process_frame")]:
		_expect(value, var_to_str(value))
	for value in [Vector2(1.5, 2), Color(0.25, 0.5, 0.75), Rect2(1, 2, 3, 4)]:
		var encoded: Dictionary = VariantJson.encode(value)
		_check(str_to_var(encoded.value) == value, "variant text round trip")
	_check(is_nan(str_to_var(VariantJson.encode(NAN).value)), "NaN variant text round trip")
	for value in [PackedByteArray([1, 2]), PackedInt32Array([1, 2]), PackedInt64Array([9007199254740993]), PackedFloat32Array([NAN, INF]), PackedFloat64Array([-INF, 1.5]), PackedStringArray(["a", "b"]), PackedVector2Array([Vector2(1, 2)]), PackedVector3Array([Vector3(1, 2, 3)]), PackedColorArray([Color.RED])]:
		var expected: Array = []
		for element in value:
			expected.append(VariantJson.encode(element).value)
		_expect(value, expected)
	if type_string(TYPE_MAX - 1) == "PackedVector4Array":
		_expect(str_to_var("PackedVector4Array(1, 2, 3, 4)"), ["Vector4(1, 2, 3, 4)"])
	var object: Object = Node.new()
	_expect(object, str(object))
	object.free()
	_expect(object, null)
	var cycle: Array = []
	cycle.append(cycle)
	_omitted(cycle, "depth_limit")
	cycle.clear()
	var recursive: Dictionary = {}
	recursive.self = recursive
	_omitted(recursive, "depth_limit")
	recursive.clear()
	var nested: Variant = 1
	for _i in range(32):
		nested = [nested]
	_check(VariantJson.encode(nested).ok, "depth 32 is accepted")
	_omitted([nested], "depth_limit")
	var items: Array = []
	items.resize(10000)
	items.fill(0)
	_check(VariantJson.encode(items).ok, "10000 items are accepted")
	items.append(0)
	_omitted(items, "item_limit")
	_expect("x".repeat(65534), "x".repeat(65534))
	_omitted("x".repeat(65535), "size_limit")
	_omitted("x".repeat(65 * 1024), "size_limit")
	_omitted("한".repeat(22000), "size_limit")
	_omitted("\n".repeat(32768), "size_limit")
	_omitted(["x".repeat(32766), "x".repeat(32766)], "size_limit")
	_omitted({3: "numeric", "3": "text"}, "key_collision")
	var payload: Dictionary = {"result": "<null>", "type": "Nil", "result_value_omitted": "stale"}
	VariantJson.attach_result(payload, null)
	_check(payload.has("result_value") and payload.result_value == null and not payload.has("result_value_omitted"), "null is present, not omitted")
	_check(payload.result == "<null>" and payload.type == "Nil", "legacy payload fields are unchanged")
	VariantJson.attach_result(payload, {3: 1, "3": 2})
	_check(not payload.has("result_value") and payload.result_value_omitted == "key_collision", "attach replaces opposite field")
	print("variant_json_test: ", "FAIL" if _failed else "PASS")
	quit(1 if _failed else 0)


func _expect(value: Variant, expected: Variant) -> void:
	var encoded: Dictionary = VariantJson.encode(value)
	_check(encoded.ok, "value is representable")
	if not encoded.ok:
		return
	_check(encoded.value == expected, "encoded value matches")
	var parser := JSON.new()
	_check(parser.parse(JSON.stringify(encoded.value)) == OK, "encoded result parses as JSON")


func _omitted(value: Variant, reason: String) -> void:
	var encoded: Dictionary = VariantJson.encode(value)
	_check(not encoded.ok and encoded.get("reason") == reason and not encoded.has("value"), "omission reason: " + reason)


func _check(ok: bool, message: String) -> void:
	if not ok:
		_failed = true
		push_error(message)
