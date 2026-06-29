extends RefCounted

const GameImageAnalyzer = preload("res://addons/hera_agent_godot/runtime/game_image_analyzer.gd")

static func click(viewport: Viewport, position: Vector2) -> void:
	_push_mouse_button(viewport, position, true)
	_push_mouse_button(viewport, position, false)

static func screenshot(viewport: Viewport, request: Dictionary, scene_path: String, pid: int) -> Dictionary:
	var image := viewport.get_texture().get_image()
	if image == null or image.is_empty():
		return { "ok": false, "error": "runtime screenshot produced an empty image" }
	var out_path := String(request.get("path", "user://hera_game_screenshots/latest.png"))
	var abs_path := ProjectSettings.globalize_path(out_path)
	DirAccess.make_dir_recursive_absolute(abs_path.get_base_dir())
	var err := image.save_png(out_path)
	if err != OK:
		return { "ok": false, "error": "save failed: %s" % error_string(err) }
	var data := {
		"path": abs_path,
		"width": image.get_width(),
		"height": image.get_height(),
		"pid": pid,
		"scene": scene_path,
	}
	if bool(request.get("analyze", false)):
		data["analysis"] = GameImageAnalyzer.analyze(image)
	return { "ok": true, "data": data }

static func _push_mouse_button(viewport: Viewport, position: Vector2, pressed: bool) -> void:
	var event := InputEventMouseButton.new()
	event.button_index = MOUSE_BUTTON_LEFT
	event.pressed = pressed
	event.position = position
	event.global_position = position
	event.factor = 1.0
	viewport.push_input(event, true)
