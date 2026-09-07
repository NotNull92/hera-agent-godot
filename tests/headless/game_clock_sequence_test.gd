extends SceneTree

const Inspector = preload("res://addons/hera_agent_godot/runtime/game_inspector.gd")

class Counter extends Node:
	var process_count := 0
	var physics_count := 0
	var held_count := 0
	func _process(_delta: float) -> void:
		process_count += 1
	func _physics_process(_delta: float) -> void:
		physics_count += 1
		if Input.is_action_pressed("hera_sequence_test"):
			held_count += 1

var _failed := false

func _initialize() -> void:
	_run.call_deferred()

func _run() -> void:
	InputMap.add_action("hera_sequence_test")
	var inspector := Inspector.new()
	root.add_child(inspector)
	inspector.set_process(false)
	var counter := Counter.new()
	root.add_child(counter)
	paused = true
	for physics: bool in [true, false]:
		for priority: int in [-100, 100]:
			counter.process_priority = priority
			for attempt: int in range(3):
				var before := counter.physics_count if physics else counter.process_count
				var result: Dictionary = await inspector._clock_step({ "physics": physics })
				var after := counter.physics_count if physics else counter.process_count
				_check(bool(result.get("ok")) and paused and after - before == 1, "step callback exactly once: physics=%s priority=%d attempt=%d delta=%d" % [physics, priority, attempt, after-before])
	paused = false
	Engine.physics_ticks_per_second = 30
	var rejected: Dictionary = await inspector._input_sequence({ "events": [{ "frame": 0, "action": "hera_sequence_test", "pressed": true }] })
	_check(not bool(rejected.get("ok")), "slow physics rate rejected before injection")
	Engine.physics_ticks_per_second = 60
	Input.action_press("hera_sequence_test")
	rejected = await inspector._input_sequence({ "events": [{ "frame": 0, "action": "hera_sequence_test", "pressed": false }] })
	_check(not bool(rejected.get("ok")) and Input.is_action_pressed("hera_sequence_test"), "pre-held action is preserved")
	Input.action_release("hera_sequence_test")
	var held_before := counter.held_count
	var sequence: Dictionary = await inspector._input_sequence({ "events": [{ "frame": 0, "action": "hera_sequence_test", "pressed": true }, { "frame": 3, "action": "hera_sequence_test", "pressed": false }] })
	_check(bool(sequence.get("ok")) and counter.held_count - held_before == 3, "sequence holds action for exactly three physics callbacks")
	_check(not Input.is_action_pressed("hera_sequence_test"), "sequence releases action")
	sequence = await inspector._input_sequence({ "events": [{ "frame": 0, "action": "hera_sequence_test", "pressed": true }] })
	_check(bool(sequence.get("ok")) and not Input.is_action_pressed("hera_sequence_test"), "unmatched press is released on completion")
	sequence = await inspector._input_sequence({ "events": [{ "frame": 0, "action": "hera_sequence_test", "pressed": true }, { "frame": 1, "action": "missing_action", "pressed": false }] })
	_check(not bool(sequence.get("ok")) and not Input.is_action_pressed("hera_sequence_test"), "all actions validated before injection")
	get_tree_pause_later(inspector)
	sequence = await inspector._input_sequence({ "events": [{ "frame": 0, "action": "hera_sequence_test", "pressed": true }, { "frame": 120, "action": "hera_sequence_test", "pressed": false }] })
	_check(not bool(sequence.get("ok")) and not Input.is_action_pressed("hera_sequence_test"), "clock change cancels and releases held action")
	paused = false
	inspector.queue_free()
	counter.queue_free()
	await process_frame
	InputMap.erase_action("hera_sequence_test")
	print("game clock and input sequence checks: ", "FAIL" if _failed else "PASS")
	quit(1 if _failed else 0)

func get_tree_pause_later(inspector: Node) -> void:
	await create_timer(0.05).timeout
	var clock: Dictionary = inspector._clock({ "paused": true })
	_check(not bool(clock.get("ok")), "concurrent clock mutation rejected")
	var input: Dictionary = inspector._input_viewport({ "kind": "action", "name": "hera_sequence_test", "mode": "release" })
	_check(not bool(input.get("ok")), "concurrent input rejected")
	var sequence: Dictionary = await inspector._input_sequence({ "events": [{ "frame": 0, "action": "hera_sequence_test", "pressed": false }] })
	_check(not bool(sequence.get("ok")), "concurrent sequence rejected")
	paused = true

func _check(value: bool, message: String) -> void:
	if not value:
		_failed = true
		push_error(message)
