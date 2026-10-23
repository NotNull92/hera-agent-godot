# Hera test: editor
extends SceneTree

const SignalTool = preload("res://addons/hera_agent_godot/tools/signal_tool.gd")
const EditorTool = preload("res://addons/hera_agent_godot/tools/editor_tool.gd")
const Instancer = preload("res://addons/hera_agent_godot/tools/node_scene_instancer.gd")
var passed: bool = true

func _initialize() -> void:
	call_deferred("_test")

func _check(result: Dictionary, expected: String) -> void:
	if result.get("ok", false) or result.get("error", "") != expected:
		passed = false
		push_error("expected %s, got %s" % [expected, result])

func _test() -> void:
	var scene := Node.new()
	scene.name = "EscapeScene"
	var packed := PackedScene.new()
	packed.pack(scene)
	ResourceSaver.save(packed, "res://escape_scene.tscn")
	scene.free()
	EditorInterface.open_scene_from_path("res://escape_scene.tscn")
	await process_frame
	await process_frame
	scene = EditorInterface.get_edited_scene_root()
	var signals := SignalTool.new()
	for connect_signal in [true, false]:
		_check(signals._wire(scene, {"from": ".", "signal": "renamed", "to": "../..", "method": "queue_free"}, connect_signal), "to node not found: ../..")
		_check(signals._wire(scene, {"from": "../..", "signal": "renamed", "to": ".", "method": "queue_free"}, connect_signal), "from node not found: ../..")
	_check(EditorTool.new().execute({"action": "select", "path": "../.."}), "node not found: ../..")
	_check(Instancer.execute(scene, {"scene": "res://escape_scene.tscn", "parent": "../.."}, null), "parent not found: ../..")
	quit(0 if passed else 1)
