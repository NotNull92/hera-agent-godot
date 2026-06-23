extends RefCounted

# `screenshot` — capture the editor viewport to a PNG.
#
# Captures the last rendered frame of the 2D (default) or 3D editor viewport and
# saves it as PNG, returning the absolute path. GUI-only: under --headless there
# is no rendered frame, so it returns an error.

const ToolResponse = preload("res://addons/hera_agent_godot/core/tool_response.gd")

const DEFAULT_PATH := "user://hera_screenshots/latest.png"
const MIN_CAPTURE_SIZE := 16

func get_name() -> String:
	return "screenshot"

func execute(params: Dictionary) -> Dictionary:
	var view := String(params.get("view", "2d")).to_lower()
	var viewport: SubViewport
	if view == "3d":
		viewport = EditorInterface.get_editor_viewport_3d(0)
	elif view == "2d":
		viewport = EditorInterface.get_editor_viewport_2d()
	else:
		return ToolResponse.failure("unknown view: %s (want 2d|3d)" % view)
	if viewport == null:
		return ToolResponse.failure("no %s editor viewport available" % view)

	var texture := viewport.get_texture()
	var image: Image = texture.get_image() if texture != null else null
	if image == null or image.is_empty():
		return ToolResponse.failure("could not capture viewport image (no rendered frame; GUI editor required)")
	if image.get_width() < MIN_CAPTURE_SIZE or image.get_height() < MIN_CAPTURE_SIZE:
		return ToolResponse.failure("captured viewport image is too small: %dx%d" % [image.get_width(), image.get_height()])

	var out_path := String(params.get("path", DEFAULT_PATH))
	var abs_path := ProjectSettings.globalize_path(out_path)
	DirAccess.make_dir_recursive_absolute(abs_path.get_base_dir())
	var err := image.save_png(out_path)
	if err != OK:
		return ToolResponse.failure("save failed: %s" % error_string(err))

	return ToolResponse.success({
		"path": abs_path,
		"view": view,
		"width": image.get_width(),
		"height": image.get_height(),
	})
