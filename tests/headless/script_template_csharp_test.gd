extends SceneTree

var _failures: Array[String] = []


func _initialize() -> void:
	var path := "res://addons/hera_agent_godot/tools/script_template_csharp.gd"
	if not ResourceLoader.exists(path):
		push_error("C# script template is missing")
		quit(1)
		return
	var template: RefCounted = load(path).new()
	var result: Dictionary = template.build({"path": "res://Player.cs", "extends": "Node2D", "tool": true, "ready": true, "process": true, "physics_process": true, "input": true, "unhandled_input": true, "signals": ["Damaged"], "exports": ["Speed:float=3.5f", "Title:string=\"hello\"", "Target:Node"]})
	_expect(bool(result.get("ok", false)), "valid template failed: %s" % result)
	var source := String(result.get("text", ""))
	for fragment in ["using Godot;", "[Tool]", "public partial class Player : Node2D", "[Signal]", "public delegate void DamagedEventHandler();", "[Export] public float Speed { get; set; } = 3.5f;", "public override void _Ready()", "public override void _Process(double delta)", "public override void _PhysicsProcess(double delta)", "public override void _Input(InputEvent @event)", "public override void _UnhandledInput(InputEvent @event)"]:
		_expect(source.contains(fragment), "missing generated fragment: " + fragment)
	for invalid in [{"class_name": "Other"}, {"path": "res://class.cs"}, {"path": "res://Bad-name.cs"}, {"extends": "Node;"}, {"signals": ["Damage", "Damage"]}, {"signals": ["Player"]}, {"signals": ["Damage", "DamageEventHandler"]}, {"signals": ["class"]}, {"exports": ["Name:int", "Name:int"]}, {"signals": ["Damage"], "exports": ["Damage:int"]}, {"exports": ["class:int"]}, {"exports": ["Value:int;evil"]}, {"exports": ["Value:int=1\n2"]}, {"exports": ["Value:int="]}]:
		var params: Dictionary = {"path": "res://Player.cs"}
		params.merge(invalid, true)
		var rejected: Dictionary = template.build(params)
		_expect(not bool(rejected.get("ok", false)), "accepted invalid params: %s" % params)
	for type_name in ["int", "Godot.Node", "int[]", "Godot.Collections.Array<Node>", "Godot.Collections.Dictionary<string, Godot.Collections.Array<int>>"]:
		var typed_result: Dictionary = template.build({"path": "res://Player.cs", "exports": ["Values:" + type_name]})
		_expect(bool(typed_result.get("ok", false)), "rejected safe type: " + type_name)
	for type_name in ["int;", "Array<>", "Array<int", "Array<int>>", "Array<,int>", "Array<int,>", "9Node", "Node..Child"]:
		var typed_result: Dictionary = template.build({"path": "res://Player.cs", "exports": ["Values:" + type_name]})
		_expect(not bool(typed_result.get("ok", false)), "accepted unsafe type: " + type_name)
	for failure in _failures:
		push_error(failure)
	quit(0 if _failures.is_empty() else 1)


func _expect(condition: bool, message: String) -> void:
	if not condition:
		_failures.append(message)
