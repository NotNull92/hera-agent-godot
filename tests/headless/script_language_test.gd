extends SceneTree

const ScriptTool = preload("res://addons/hera_agent_godot/tools/script_tool.gd")
const GameInspector = preload("res://addons/hera_agent_godot/runtime/game_inspector.gd")

class QAFixture extends Node:
	func qa_ready() -> bool:
		return true
	func QaReady() -> bool:
		return true
	func Qat() -> bool:
		return true


func _initialize() -> void:
	var failures: Array[String] = []
	var tool := ScriptTool.new()
	if tool._guard_readable_script_parent("res://Player.cs") != "":
		failures.append("C# path must be accepted")
	for path in ["res://../Player.cs", "res://Player.txt", "Player.cs"]:
		if tool._guard_readable_script_parent(path) == "":
			failures.append("unsafe/unsupported path accepted: %s" % path)
	for params in [{"path": "res://Mismatch.cs", "lang": "gdscript"}, {"path": "res://mismatch.gd", "lang": "csharp"}, {"path": "res://Unknown.cs", "lang": "python"}]:
		var result: Dictionary = tool._create(params)
		if bool(result.get("ok", false)) or FileAccess.file_exists(String(params["path"])):
			failures.append("batch language mismatch was not rejected before writing")
	var fixture := QAFixture.new()
	var inspector := GameInspector.new()
	var names: Array[String] = []
	for method in inspector._qa_methods(fixture):
		names.append(String(method.get("name", "")))
	if not names.has("qa_ready") or not names.has("QaReady") or names.has("Qat"):
		failures.append("QA naming conventions not respected: %s" % str(names))
	fixture.free()
	inspector.free()
	for failure in failures:
		push_error(failure)
	quit(0 if failures.is_empty() else 1)
