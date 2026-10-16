extends RefCounted

# `eval` — native C# loader or an explicit GDScript Expression.
#
# GDScript accepts one expression, no statements. The edited scene root is used
# as the base instance, so expressions can reach the scene, e.g.
# `get_node("Player").position`. Powerful (can call methods with side effects),
# hence it lives in the mutation phase.

const VariantJson = preload("res://addons/hera_agent_godot/core/variant_json.gd")
const ToolResponse = preload("res://addons/hera_agent_godot/core/tool_response.gd")
const CSHARP_LOADER := "res://addons/hera_agent_godot/csharp/HeraCSharpEval.cs"

class EditorInterfaceProxy:
	extends RefCounted

	func get_editor_settings() -> EditorSettings:
		return EditorInterface.get_editor_settings()

	func get_edited_scene_root() -> Node:
		return EditorInterface.get_edited_scene_root()

	func get_open_scenes() -> PackedStringArray:
		return EditorInterface.get_open_scenes()

	func is_playing_scene() -> bool:
		return EditorInterface.is_playing_scene()

	func get_playing_scene() -> String:
		return EditorInterface.get_playing_scene()

func get_name() -> String:
	return "eval"

func execute(params: Dictionary) -> Dictionary:
	var lang := String(params.get("lang", "csharp"))
	if lang == "csharp":
		return _csharp(params)
	if lang != "gdscript":
		return ToolResponse.failure("eval language must be csharp or gdscript")
	var text := String(params.get("expr", ""))
	if text == "":
		return ToolResponse.failure("eval requires an 'expr' string")

	var expression := Expression.new()
	var input_names := PackedStringArray()
	var input_values := []
	if text.contains("EditorInterface"):
		input_names.append("EditorInterface")
		input_values.append(EditorInterfaceProxy.new())

	if expression.parse(text, input_names) != OK:
		return ToolResponse.failure("parse error: %s" % expression.get_error_text())

	var base: Object = EditorInterface.get_edited_scene_root()
	var result: Variant = expression.execute(input_values, base, true)
	if expression.has_execute_failed():
		return ToolResponse.failure("execute error: %s" % expression.get_error_text())

	return ToolResponse.success(VariantJson.attach_result({
		"result": str(result),
		"type": type_string(typeof(result)),
		"undoable": false,
	}, result))

func _csharp(params: Dictionary) -> Dictionary:
	var version: Dictionary = Engine.get_version_info()
	if not ClassDB.class_exists("CSharpScript") or int(version.get("major", 0)) != 4 or int(version.get("minor", 0)) < 7:
		return ToolResponse.failure("csharp_unavailable: eval requires Godot 4.7 .NET; use --lang gdscript for a GDScript expression")
	var script: Script = ResourceLoader.load(CSHARP_LOADER) as Script
	if script == null or not script.can_instantiate():
		return ToolResponse.failure("csharp_build_required: build and reload the project with the Hera C# loader first")
	var loader: Variant = script.new()
	if loader == null or not loader.has_method("CompilationContext"):
		return ToolResponse.failure("csharp_build_required: rebuild and reload the Hera C# loader")
	if String(params.get("action", "")) == "csharp-context":
		var context: Dictionary = loader.call("CompilationContext")
		return ToolResponse.success(context)
	var encoded := String(params.get("assembly", ""))
	if encoded.is_empty() or encoded.length() > 524288:
		return ToolResponse.failure("csharp_compile_required: invoke eval through the CLI C# compiler host")
	var bytes: PackedByteArray = Marshalls.base64_to_raw(encoded)
	var response: Dictionary = loader.call("Evaluate", bytes, EditorInterface.get_edited_scene_root(), String(params.get("project_mvid", "")))
	if not bool(response.get("ok", false)):
		return ToolResponse.failure(String(response.get("error", "C# eval failed")))
	var result: Variant = response.get("value")
	return ToolResponse.success(VariantJson.attach_result({
		"result": str(result), "type": type_string(typeof(result)), "undoable": false, "lang": "csharp",
	}, result))
