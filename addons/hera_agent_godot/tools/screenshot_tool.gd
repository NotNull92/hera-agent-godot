extends RefCounted

# `screenshot` — render the edited scene off-screen and save a PNG.
#
# The editor's own viewport texture reads back as a placeholder in Godot 4.7, so
# instead we clone the edited scene into a temporary SubViewport, render it, and
# capture that. A correct capture needs a rendered frame, so the plugin routes
# this tool through execute_async() (which awaits one SceneTree frame);
# execute() is a synchronous best-effort fallback (e.g. when used inside `batch`).
#
# Caveats: the scene renders from the world origin unless it contains a camera
# (Camera2D / Camera3D); non-@tool scripts do not run in the editor.

const ToolResponse = preload("res://addons/hera_agent_godot/core/tool_response.gd")

const DEFAULT_PATH := "user://hera_screenshots/latest.png"
const MIN_CAPTURE_SIZE := 16
const MAX_CAPTURE_SIZE := 4096
const DUPLICATE_RENDER_FLAGS := Node.DUPLICATE_SIGNALS | Node.DUPLICATE_GROUPS | Node.DUPLICATE_USE_INSTANTIATION

var _host: Node # an editor-tree node to parent the offscreen viewport (the plugin)

func set_host(host: Node) -> void:
	_host = host

func get_name() -> String:
	return "screenshot"

# Synchronous best-effort (may be blank without a real frame; used inside batch).
func execute(params: Dictionary) -> Dictionary:
	var ctx := _setup(params)
	if ctx.has("error"):
		return ctx["error"]
	RenderingServer.force_draw(false)
	return _finalize(ctx, params)

# Awaits a rendered frame for a correct capture; used by the plugin's dispatch.
func execute_async(params: Dictionary) -> Dictionary:
	var ctx := _setup(params)
	if ctx.has("error"):
		return ctx["error"]
	# process_frame (unlike RenderingServer.frame_post_draw) also fires under
	# --headless, so this never hangs. By the next frame the UPDATE_ALWAYS
	# viewport has rendered once; in headless there is no render, so the capture
	# falls through to a graceful "empty image" error instead of blocking.
	var tree := _host.get_tree() if _host != null else null
	if tree != null:
		await tree.process_frame
	return _finalize(ctx, params)

func _setup(params: Dictionary) -> Dictionary:
	if _host == null:
		return { "error": ToolResponse.failure("screenshot host not set") }
	var root := EditorInterface.get_edited_scene_root()
	if root == null:
		return { "error": ToolResponse.failure("no scene is open to capture") }

	var width := int(params.get("width", _setting("display/window/size/viewport_width", 1280)))
	var height := int(params.get("height", _setting("display/window/size/viewport_height", 720)))
	if width < MIN_CAPTURE_SIZE or height < MIN_CAPTURE_SIZE:
		return { "error": ToolResponse.failure("size too small: %dx%d (min %d)" % [width, height, MIN_CAPTURE_SIZE]) }
	if width > MAX_CAPTURE_SIZE or height > MAX_CAPTURE_SIZE:
		return { "error": ToolResponse.failure("size too large: %dx%d (max %d)" % [width, height, MAX_CAPTURE_SIZE]) }

	var viewport := SubViewport.new()
	viewport.size = Vector2i(width, height)
	viewport.transparent_bg = bool(params.get("transparent", false))
	viewport.render_target_update_mode = SubViewport.UPDATE_ALWAYS
	viewport.add_child(root.duplicate(DUPLICATE_RENDER_FLAGS))
	_host.add_child(viewport)
	return { "viewport": viewport }

func _finalize(ctx: Dictionary, params: Dictionary) -> Dictionary:
	var viewport: SubViewport = ctx["viewport"]
	var image := viewport.get_texture().get_image()
	_host.remove_child(viewport)
	viewport.queue_free()

	if image == null or image.is_empty():
		return ToolResponse.failure("capture produced an empty image")
	if image.get_width() < MIN_CAPTURE_SIZE or image.get_height() < MIN_CAPTURE_SIZE:
		return ToolResponse.failure("captured image is too small: %dx%d" % [image.get_width(), image.get_height()])

	var out_path := String(params.get("path", DEFAULT_PATH))
	var abs_path := ProjectSettings.globalize_path(out_path)
	DirAccess.make_dir_recursive_absolute(abs_path.get_base_dir())
	var err := image.save_png(out_path)
	if err != OK:
		return ToolResponse.failure("save failed: %s" % error_string(err))

	return ToolResponse.success({
		"path": abs_path,
		"width": image.get_width(),
		"height": image.get_height(),
	})

func _setting(name: String, fallback: int) -> int:
	return int(ProjectSettings.get_setting(name, fallback))
