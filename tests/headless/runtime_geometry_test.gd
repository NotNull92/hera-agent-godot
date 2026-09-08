extends SceneTree

const UI = preload("res://addons/hera_agent_godot/runtime/game_ui_inspector.gd")
const Auditor = preload("res://addons/hera_agent_godot/runtime/game_ui_auditor.gd")
const Actions = preload("res://addons/hera_agent_godot/runtime/game_viewport_actions.gd")
var _failed := false
var _clicks := 0

func _initialize() -> void:
	call_deferred("_run")

func _run() -> void:
	var viewport := SubViewport.new()
	viewport.size = Vector2i(400, 300)
	root.add_child(viewport)
	var layer := CanvasLayer.new()
	layer.offset = Vector2(100, 50)
	layer.scale = Vector2(2, 2)
	viewport.add_child(layer)
	var control := Control.new()
	control.name = "Target"
	control.position = Vector2(10, 20)
	control.size = Vector2(40, 30)
	control.focus_mode = Control.FOCUS_ALL
	control.gui_input.connect(_on_input)
	layer.add_child(control)
	await process_frame
	var target: Dictionary = UI.click_target(viewport, layer, {"path": "Target"})
	_check(target.get("position") == Vector2(160, 120), "CanvasLayer click uses viewport center")
	Actions.click(viewport, target.get("position", Vector2.ZERO))
	_check(_clicks == 1, "transformed target receives actual viewport input")
	var summary: Dictionary = UI._control_summary(control, ["rect"])
	_check(summary.rect == {"x": 120, "y": 90, "width": 80, "height": 60}, "tree uses transformed viewport bounds")
	layer.offset = Vector2(500, 0)
	var audit: Dictionary = Auditor.audit(viewport, layer, 100, {"rule": "interactive_outside_viewport"})
	_check(audit.errors == 1, "translated layer outside viewport is reported")
	_check(not bool(UI.click_target(root, layer, {"path": "Target"}).get("ok", false)), "cross-viewport click rejected")
	layer.free()
	var camera := Camera2D.new()
	camera.position = Vector2(500, 0)
	viewport.add_child(camera)
	camera.make_current()
	camera.force_update_scroll()
	control = Control.new()
	control.position = Vector2(500, 0)
	control.size = Vector2(40, 30)
	control.focus_mode = Control.FOCUS_ALL
	viewport.add_child(control)
	await process_frame
	target = UI.click_target(viewport, viewport, {"path": String(control.get_path())})
	_check(target.get("position") == Vector2(220, 165), "camera click uses viewport center")
	audit = Auditor.audit(viewport, control, 100, {"rule": "interactive_outside_viewport"})
	_check(audit.errors == 0, "camera brings canvas control into viewport")
	viewport.free()
	print("runtime_geometry_test: ", "FAIL" if _failed else "PASS")
	quit(1 if _failed else 0)

func _check(value: bool, message: String) -> void:
	if not value:
		_failed = true
		push_error(message)

func _on_input(event: InputEvent) -> void:
	if event is InputEventMouseButton and event.pressed:
		_clicks += 1
