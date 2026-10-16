extends SceneTree

func _initialize() -> void:
	call_deferred("_probe")

func _probe() -> void:
	var theme: Theme = EditorInterface.get_editor_theme()
	assert(theme != null)
	assert(EditorInterface.get_editor_scale() > 0.0)
	var control := Control.new()
	control.theme = theme
	root.add_child(control)
	assert(control.has_theme_color("font_color", "Editor"))
	var color: Color = control.get_theme_color("font_color", "Editor")
	print("HERA_THEME_PROBE_OK:", color, ":", EditorInterface.get_editor_scale())
	control.free()
	quit(0)
