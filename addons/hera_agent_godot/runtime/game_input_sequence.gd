extends Node

signal completed(result: Dictionary)

var _events: Array = []
var _held: Dictionary = {}
var _inject: Callable
var _frame := -1
var _index := 0
var _running := false
var _ticks := 60
var _executed: Array[Dictionary] = []

func run(request: Dictionary, inject: Callable) -> Dictionary:
	process_mode = Node.PROCESS_MODE_ALWAYS
	var error := _validate(request)
	if error != "":
		return { "ok": false, "error": error }
	_events = request["events"]
	_inject = inject
	_ticks = Engine.physics_ticks_per_second
	_running = true
	get_tree().physics_frame.connect(_on_frame)
	get_tree().create_timer(2.5, true, false, true).timeout.connect(_on_timeout)
	return await completed

func _validate(request: Dictionary) -> String:
	if get_tree().paused or Engine.time_scale != 1.0 or Engine.physics_ticks_per_second < 60:
		return "input sequence requires an unpaused tree, time_scale 1, and at least 60 physics ticks per second"
	var events: Variant = request.get("events")
	if not events is Array or events.size() == 0 or events.size() > 128:
		return "input sequence requires 1..128 events"
	var previous := 0
	for value: Variant in events:
		if not value is Dictionary:
			return "input sequence events must be objects"
		var event: Dictionary = value
		var frame: Variant = event.get("frame")
		var action: Variant = event.get("action")
		if (not frame is int and not frame is float) or not is_finite(float(frame)) or float(frame) != floor(float(frame)) or float(frame) < previous or float(frame) > 120:
			return "input sequence frames must be ordered integers in 0..120"
		if not action is String or String(action).strip_edges() == "" or not event.get("pressed") is bool or event.size() != 3:
			return "input sequence events require only frame, action, and boolean pressed"
		if not InputMap.has_action(action):
			return "unknown InputMap action: %s" % action
		if Input.is_action_pressed(action):
			return "input sequence action is already held: %s" % action
		previous = int(frame)
	return ""

func _process(_delta: float) -> void:
	if _running and (get_tree().paused or Engine.time_scale != 1.0 or Engine.physics_ticks_per_second != _ticks):
		_finish("input sequence cancelled because the game clock changed")

func _on_frame() -> void:
	if not _running:
		return
	if get_tree().paused or Engine.time_scale != 1.0 or Engine.physics_ticks_per_second != _ticks:
		_finish("input sequence cancelled because the game clock changed")
		return
	_frame += 1
	while _index < _events.size() and int(_events[_index]["frame"]) == _frame:
		var event: Dictionary = _events[_index]
		var action := String(event["action"])
		var pressed := bool(event["pressed"])
		_inject.call({ "kind": "action", "name": action, "mode": "press" if pressed else "release" })
		if pressed:
			_held[action] = true
		else:
			_held.erase(action)
		_executed.append({ "frame": _frame, "action": action, "pressed": pressed, "physics_frame": Engine.get_physics_frames() })
		_index += 1
	# Parsed events are buffered by Godot until explicitly flushed.
	Input.flush_buffered_events()
	if _index == _events.size():
		await get_tree().create_timer(0.0, true, true, true).timeout
		_finish("")

func _on_timeout() -> void:
	_finish("input sequence exceeded its 2.5 second runtime deadline")

func _finish(error: String) -> void:
	if not _running:
		return
	_running = false
	get_tree().physics_frame.disconnect(_on_frame)
	_release()
	var result := { "ok": error == "", "kind": "sequence", "frames": _frame + 1, "events": _executed }
	if error != "":
		result["error"] = error
	completed.emit(result)

func _release() -> void:
	for action: String in _held:
		_inject.call({ "kind": "action", "name": action, "mode": "release" })
	_held.clear()
	Input.flush_buffered_events()

func _exit_tree() -> void:
	_release()
