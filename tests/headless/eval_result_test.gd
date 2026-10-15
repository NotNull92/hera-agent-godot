# Hera test: editor
extends SceneTree

const EvalTool = preload("res://addons/hera_agent_godot/tools/eval_tool.gd")
var _failed := false


func _initialize() -> void:
	call_deferred("_run")


func _run() -> void:
	var tool := EvalTool.new()
	for test in [
		{"expr": "1+1", "value": 2},
		{"expr": "null", "value": null},
		{"expr": '{"a": 1}', "value": {"a": 1}},
		{"expr": "Vector2(1, 2)", "value": "Vector2(1, 2)"},
		{"expr": "NAN", "value": "nan"},
		{"expr": "INF", "value": "inf"},
		{"expr": "-INF", "value": "inf_neg"},
		{"expr": "9007199254740993", "value": "9007199254740993"},
	]:
		var response: Dictionary = tool.execute({"expr": test.expr})
		_check(response.get("ok", false), "eval succeeds: " + test.expr)
		var data: Dictionary = response.get("data", {})
		_check(data.has("result_value") and data.get("result_value") == test.value and not data.has("result_value_omitted"), "eval attaches structured value: " + test.expr)
		_check(data.get("result") is String and data.get("undoable") == false and data.get("type") is String, "stable fields retain types")
		var parser := JSON.new()
		_check(parser.parse(JSON.stringify(response)) == OK, "eval response is valid JSON")
	var sum: Dictionary = tool.execute({"expr": "1+1"}).get("data", {})
	_check(sum.result == "2" and sum.type == "int", "original eval scalar contract is unchanged")
	var omitted: Dictionary = tool.execute({"expr": '{3: 1, "3": 2}'}).get("data", {})
	_check(omitted.get("result_value_omitted") == "key_collision" and not omitted.has("result_value"), "eval exposes omission")
	_check(not tool.execute({"expr": "1+"}).get("ok", true), "expression errors still fail")
	print("eval_result_test: ", "FAIL" if _failed else "PASS")
	quit(1 if _failed else 0)


func _check(ok: bool, message: String) -> void:
	if not ok:
		_failed = true
		push_error(message)
