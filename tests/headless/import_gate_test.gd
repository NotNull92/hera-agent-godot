# Hera test: editor
extends SceneTree

const WorkQueue = preload("res://addons/hera_agent_godot/server/work_queue.gd")
const Registry = preload("res://addons/hera_agent_godot/core/tool_registry.gd")
const Project = preload("res://addons/hera_agent_godot/tools/project_tool.gd")
var queue := WorkQueue.new()
var registry := Registry.new()
var failed := false
var reentries := 0

func _initialize() -> void:
	_run.call_deferred()

func _run() -> void:
	var fs := EditorInterface.get_resource_filesystem()
	while fs.is_scanning() or fs.is_processing():
		await process_frame
	queue.configure_operations("imports")
	queue.set_filesystem(fs)
	registry.register(Project.new())
	var image := FileAccess.open("res://gate.svg", FileAccess.WRITE)
	image.store_string('<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16"><rect width="16" height="16" fill="red"/></svg>')
	image.close()
	var source := FileAccess.open("res://gate_script.gd", FileAccess.WRITE)
	source.store_string("@tool\nclass_name GateScannedScript\nextends Resource\n")
	source.close()
	if fs.has_signal("resources_reimporting"):
		fs.connect("resources_reimporting", _reenter)
	var result := await _execute({"tool": "project", "params": {"action": "scan"}})
	_check(result.ok and not fs.is_scanning() and ResourceLoader.exists("res://gate.svg"), "scan returns after requested resource becomes available")
	var script := ResourceLoader.load("res://gate_script.gd") as Script
	_check(script != null and script.can_instantiate(), "scan returns with requested GDScript available")
	var registered := false
	for entry: Dictionary in ProjectSettings.get_global_class_list():
		if entry.get("class") == "GateScannedScript":
			registered = true
	_check(registered, "scan returns after the requested global script class is registered")
	result = await _execute({"tool": "project", "params": {"action": "reimport", "paths": ["res://gate.svg"]}})
	_check(result.ok and ResourceLoader.load("res://gate.svg") is Texture2D, "reimport preserves loadable resource")
	_check(reentries >= 2, "real import callback attempted collisions during scan and reimport")
	_check(queue._owner.is_empty(), "import ownership released")
	fs.scan()
	var collision := {"request": {"tool": "project", "params": {"action": "scan"}}}
	_check(not queue.begin(collision), "engine-owned scan rejects mutation before dispatch")
	while fs.is_scanning() or fs.is_processing():
		await process_frame
	if fs.has_signal("resources_reimporting"):
		fs.disconnect("resources_reimporting", _reenter)
	queue.retire()
	print("import_gate_test: ", "FAIL" if failed else "PASS")
	quit(1 if failed else 0)

func _reenter(_paths: PackedStringArray) -> void:
	reentries += 1
	for action: String in ["scan", "reimport", "mkdir", "set_main_scene"]:
		var item := {"request": {"tool": "project", "params": {"action": action, "path": "res://must-not-exist"}}}
		_check(not queue.begin(item), "real import reentry cannot execute " + action)
		_check(String(item.get("response", {}).get("error", "")).begins_with("editor_busy:"), "reentry has stable busy code")
	_check(not DirAccess.dir_exists_absolute("res://must-not-exist"), "reentry caused no wrong mutation")
	var read := {"request": {"tool": "status"}}
	_check(queue.begin(read), "status admitted during real import")
	queue.complete(read, {"ok": true})

func _execute(request: Dictionary) -> Dictionary:
	var item := {"request": request}
	if not queue.begin(item):
		return item.response
	return queue.complete(item, await queue.execute_request(request, registry, item.get("gate_owner", "")))

func _check(value: bool, message: String) -> void:
	if not value:
		failed = true
		push_error(message)
