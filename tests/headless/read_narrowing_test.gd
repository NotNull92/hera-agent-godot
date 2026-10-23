# Hera test: editor
extends SceneTree

const SceneTool = preload("res://addons/hera_agent_godot/tools/scene_tool.gd")
const NodeTool = preload("res://addons/hera_agent_godot/tools/node_tool.gd")
const Codec = preload("res://addons/hera_agent_godot/tools/node_value_codec.gd")
const SignalTool = preload("res://addons/hera_agent_godot/tools/signal_tool.gd")
var failed: bool = false

func _initialize() -> void:
	call_deferred("_test")

func _check(value: bool, message: String) -> void:
	if not value:
		failed = true
		push_error(message)

func _test() -> void:
	var plain := Node2D.new()
	var changed: Dictionary = Codec.properties(plain, 200, true)
	_check(not changed.has("visible") and not changed.has("position"), "class defaults omitted")
	plain.visible = false
	_check(Codec.properties(plain, 200, true).get("visible") == "false", "changed native property included")
	plain.free()
	var custom: Node2D = load("res://tests/headless/read_narrowing_fixture.gd").new()
	changed = Codec.properties(custom, 200, true)
	_check(changed.get("amount") == "7", "revert takes priority over script default")
	var script_default: Variant = custom.get_script().get_property_default_value("count")
	_check(changed.has("count") == (custom.get("count") != script_default), "script comparison uses engine-reported default")
	custom.set("amount", 9)
	custom.set("count", 4)
	changed = Codec.properties(custom, 200, true)
	_check(not changed.has("amount") and changed.get("count") == "4", "script defaults and custom revert respected")
	custom.free()
	var exported_script := GDScript.new()
	exported_script.source_code = "extends Node\n@export var count: int = 3\n"
	_check(exported_script.reload() == OK, "exported default fixture compiles")
	var placeholder := Node.new()
	placeholder.set_script(exported_script)
	_check(exported_script.get_property_default_value("count") == 3, "placeholder export default is available")
	_check(not Codec.properties(placeholder, 200, true).has("count"), "unchanged script export omitted")
	placeholder.set("count", 4)
	_check(Codec.properties(placeholder, 200, true).get("count") == "4", "changed script export included")
	placeholder.free()
	var scene := Node2D.new()
	scene.name = "Main"
	var child := Node2D.new()
	child.name = "Child"
	scene.add_child(child)
	child.owner = scene
	var grandchild := Node2D.new()
	grandchild.name = "Grandchild"
	child.add_child(grandchild)
	grandchild.owner = scene
	var packed := PackedScene.new()
	packed.pack(scene)
	ResourceSaver.save(packed, "res://narrow.tscn")
	scene.free()
	EditorInterface.open_scene_from_path("res://narrow.tscn")
	await process_frame
	await process_frame
	scene = EditorInterface.get_edited_scene_root() as Node2D
	var tool := SceneTool.new()
	var result: Dictionary = tool.execute({"action": "tree", "path": "Child", "depth": 0, "type": "Node", "props": ["visible"]})
	_check(result.get("ok", false) and result.data.count == 1 and result.data.nodes[0].path == "Child" and result.data.nodes[0].props.visible == "true", "scoped tree with inherited class and props")
	_check(tool.execute({"action": "tree", "depth": 1}).data.count == 2, "depth relative to subtree")
	_check(tool.execute({"action": "tree", "type": "Control"}).data.count == 0, "unmatched class")
	_check(tool.execute({"action": "tree", "path": "../.."}).get("error") == "node not found: ../..", "subtree cannot escape scene")
	for params: Dictionary in [{"depth": -1}, {"depth": 0.5}, {"props": "name"}, {"props": [5]}]:
		params["action"] = "tree"
		_check(not tool.execute(params).get("ok", false), "invalid tree options rejected")
	_check(not tool.execute({"action": "tree", "props": ["missing_property"]}).get("ok", false), "missing selected property rejected")
	_check(not NodeTool.new().execute({"action": "get", "changed": true, "prop": "name"}).get("ok", false), "changed rejects selected props through RPC")
	var signals := SignalTool.new()
	scene.renamed.connect(Callable(scene.get_node("Child"), "get_name"))
	scene.tree_entered.connect(Callable(self, "_noop"))
	result = signals.execute({"action": "list", "node": ".", "connected": true})
	_check(result.get("ok", false), "connected signals read")
	var names: Array = []
	for entry: Dictionary in result.data.signals:
		names.append(entry.name)
		_check(not entry.connections.is_empty() or int(entry.get("external_connections", 0)) > 0, "empty signals excluded")
	_check(names.has("renamed") and names.has("tree_entered"), "local and external connections retained")
	quit(1 if failed else 0)

func _noop() -> void:
	pass
