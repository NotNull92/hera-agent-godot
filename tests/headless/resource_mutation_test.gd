extends SceneTree

const Codec = preload("res://addons/hera_agent_godot/tools/resource_value_codec.gd")
const ThemeTool = preload("res://addons/hera_agent_godot/tools/theme_tool.gd")
var passed: bool = true

func _initialize() -> void:
	call_deferred("_test")

func _check(condition: bool, message: String) -> void:
	if not condition:
		passed = false
		push_error(message)

func _test() -> void:
	var resource := Resource.new()
	resource.resource_name = "original"
	var result: Dictionary = Codec.apply_props(resource, {"resource_name": "changed", "zz_invalid": "bad"})
	_check(not bool(result.get("ok", false)) and resource.resource_name == "original", "rejected resource edit changed first property")
	result = Codec.apply_props(resource, {"resource_name": "changed", "resource_local_to_scene": "invalid"})
	_check(not bool(result.get("ok", false)) and resource.resource_name == "original", "coercion failure changed first property")
	var gradient := Gradient.new()
	gradient.resource_name = "original"
	var offsets: PackedFloat32Array = gradient.offsets.duplicate()
	result = Codec.apply_props(gradient, {"resource_name": "changed", "offsets": {"bad": true}})
	_check(not bool(result.get("ok", false)) and gradient.resource_name == "original" and gradient.offsets == offsets, "wrong JSON type changed resource data")
	result = Codec.apply_props(gradient, {"resource_name": "changed", "interpolation_mode": 1.0})
	_check(bool(result.get("ok", false)) and gradient.interpolation_mode == 1, "integral JSON number rejected")
	result = Codec.apply_props(gradient, {"resource_name": "unexpected", "interpolation_mode": 0.5})
	_check(not bool(result.get("ok", false)) and gradient.resource_name == "changed" and gradient.interpolation_mode == 1, "fractional JSON integer changed resource data")
	result = Codec.apply_props(gradient, {"resource_name": "unexpected", "interpolation_mode": 1e30})
	_check(not bool(result.get("ok", false)) and gradient.resource_name == "changed", "out-of-range JSON integer was accepted")
	var theme := Theme.new()
	theme.set_color("font_color", "Label", Color.WHITE)
	_check(ResourceSaver.save(theme, "res://mutation_theme.tres") == OK, "fixture save failed")
	theme = ResourceLoader.load("res://mutation_theme.tres")
	var tool := ThemeTool.new()
	result = tool.execute({"action": "set", "path": "res://mutation_theme.tres", "type": "Label", "colors": {"font_color": "Color(1, 0, 0, 1)"}, "constants": {"outline_size": "bad"}})
	_check(not bool(result.get("ok", false)) and theme.get_color("font_color", "Label") == Color.WHITE, "rejected theme edit changed cached color")
	result = tool.execute({"action": "set", "path": "res://mutation_theme.tres", "type": "Label", "constants": {"outline_size": "3"}, "font_sizes": {"font_size": "bad"}})
	_check(not bool(result.get("ok", false)) and not theme.has_constant("outline_size", "Label"), "rejected theme edit added constant")
	result = tool.execute({"action": "set", "path": "res://mutation_theme.tres", "type": "Label", "colors": {"font_color": "Color(1, 0, 0, 1)"}, "constants": {"outline_size": "3"}, "font_sizes": {"font_size": "18"}})
	_check(bool(result.get("ok", false)), "valid theme edit failed")
	var saved: Theme = ResourceLoader.load("res://mutation_theme.tres", "", ResourceLoader.CACHE_MODE_IGNORE)
	_check(saved.get_color("font_color", "Label") == Color.RED and saved.get_constant("outline_size", "Label") == 3 and saved.get_font_size("font_size", "Label") == 18, "valid theme values not saved")
	DirAccess.remove_absolute(ProjectSettings.globalize_path("res://mutation_theme.tres"))
	print("resource mutation: ", "PASS" if passed else "FAIL")
	quit(0 if passed else 1)
