extends RefCounted

# `screenshot` — render the edited scene off-screen and save a PNG.
#
# The editor's own viewport texture reads back as a placeholder in Godot 4.7, so
# instead we clone the edited scene into a temporary SubViewport, render it, and
# capture that. A correct capture needs a rendered frame, so the plugin routes
# this tool through execute_async() with a bounded rendered-frame wait.
# execute() is a synchronous best-effort fallback for direct callers.
#
# Caveats: the scene renders from the world origin unless it contains a camera
# (Camera2D / Camera3D); non-@tool scripts do not run in the editor.

const ToolResponse = preload("res://addons/hera_agent_godot/core/tool_response.gd")
const CaptureEvidence = preload("res://addons/hera_agent_godot/core/capture_evidence.gd")
const ViewportActions = preload("res://addons/hera_agent_godot/runtime/game_viewport_actions.gd")

const DEFAULT_PATH := "user://hera_screenshots/latest.png"
const MIN_CAPTURE_SIZE := 16
const MAX_CAPTURE_SIZE := 4096
const DUPLICATE_RENDER_FLAGS := Node.DUPLICATE_SIGNALS | Node.DUPLICATE_GROUPS | Node.DUPLICATE_USE_INSTANTIATION

var _host: Node # an editor-tree node to parent the offscreen viewport (the plugin)
var editor_session_id := ""

func set_host(host: Node) -> void:
	_host = host

func get_name() -> String:
	return "screenshot"

# Synchronous best-effort; normal CLI and batch dispatch use execute_async.
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
	if not await CaptureEvidence.wait_for_draw(_host.get_tree()):
		if is_instance_valid(ctx.get("viewport")):
			(ctx["viewport"] as SubViewport).queue_free()
		return CaptureEvidence.capture_error(params, "no rendered frame within 1000 ms")
	return _finalize(ctx, params)

func _setup(params: Dictionary) -> Dictionary:
	var guard := CaptureEvidence.guard(params)
	if guard != "":
		return {"error": ToolResponse.failure(guard)}
	if DisplayServer.get_name() == "headless":
		return {"error": CaptureEvidence.capture_error(params, "headless renderer")}
	if not is_instance_valid(_host) or not _host.is_inside_tree():
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
	return { "viewport": viewport, "scene": root.scene_file_path }

func _finalize(ctx: Dictionary, params: Dictionary) -> Dictionary:
	if not is_instance_valid(_host) or not is_instance_valid(ctx.get("viewport")):
		if is_instance_valid(ctx.get("viewport")):
			(ctx["viewport"] as SubViewport).queue_free()
		return CaptureEvidence.capture_error(params, "editor capture host ended")
	var viewport: SubViewport = ctx["viewport"]
	var image := viewport.get_texture().get_image()
	var evidence := CaptureEvidence.snapshot("editor_preview", editor_session_id)
	var geometry := ViewportActions.geometry(viewport)
	_host.remove_child(viewport)
	viewport.queue_free()

	if image == null or image.is_empty():
		return CaptureEvidence.capture_error(params, "capture produced an empty image")
	if image.get_width() < MIN_CAPTURE_SIZE or image.get_height() < MIN_CAPTURE_SIZE:
		return ToolResponse.failure("captured image is too small: %dx%d" % [image.get_width(), image.get_height()])

	var out_path := String(params.get("path", DEFAULT_PATH))
	var abs_path := ProjectSettings.globalize_path(out_path)
	DirAccess.make_dir_recursive_absolute(abs_path.get_base_dir())
	var err := image.save_png(out_path)
	if err != OK:
		return ToolResponse.failure("save failed: %s" % error_string(err))

	var data := {
		"path": abs_path,
		"width": image.get_width(),
		"height": image.get_height(),
	}
	if bool(params.get("evidence", false)):
		evidence["editor_pid"] = OS.get_process_id()
		evidence["scene"] = ctx.scene
		evidence["width"] = image.get_width()
		evidence["height"] = image.get_height()
		evidence["geometry"] = geometry
		CaptureEvidence.link(evidence, params)
		data["evidence"] = evidence
	return ToolResponse.success(data)

func _setting(name: String, fallback: int) -> int:
	return int(ProjectSettings.get_setting(name, fallback))
