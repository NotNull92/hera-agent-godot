extends SceneTree

const HeraPlugin = preload("res://addons/hera_agent_godot/hera_agent_plugin.gd")


func _initialize() -> void:
	var plugin := HeraPlugin.new()
	if not plugin.has_method("_owns_game_autoload"):
		push_error("Hera plugin must recognize its persisted autoload")
		plugin.free()
		quit(1)
		return
	var expected := "res://addons/hera_agent_godot/runtime/game_inspector.gd"
	var failures: Array[String] = []
	if not plugin._owns_game_autoload("*" + expected):
		failures.append("owned res:// autoload not recognized")
	if not plugin._owns_game_autoload("*uid://c4ug7a211oav8"):
		failures.append("owned uid:// autoload not recognized")
	if plugin._owns_game_autoload("*res://user/game_inspector.gd"):
		failures.append("user-owned autoload claimed")
	plugin.free()
	for failure in failures:
		push_error(failure)
	quit(0 if failures.is_empty() else 1)
