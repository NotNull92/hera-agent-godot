extends SceneTree

const ResourceTool = preload("res://addons/hera_agent_godot/tools/resource_tool.gd")
const ViewportActions = preload("res://addons/hera_agent_godot/runtime/game_viewport_actions.gd")
const CaptureEvidence = preload("res://addons/hera_agent_godot/core/capture_evidence.gd")
const Persistence = preload("res://addons/hera_agent_godot/core/persistence_evidence.gd")
const Inspector = preload("res://addons/hera_agent_godot/runtime/game_inspector.gd")

func _initialize() -> void:
	call_deferred("_run")

func _run() -> void:
	var path := "user://save_failure.tres"
	var resource := Resource.new()
	resource.resource_name = "before"
	assert(ResourceSaver.save(resource, path) == OK)
	var cached: Resource = ResourceLoader.load(path)
	var disk := Persistence.observe(path)
	assert(bool(disk.available))
	var conflict: Dictionary = ResourceTool.new().execute({"action": "set", "path": path, "props": {"resource_name": "wrong"}, "evidence": true, "expected_sha256": "0".repeat(64)})
	assert(not bool(conflict.ok) and cached.resource_name == "before", "external change guard must reject before mutation")
	assert(DirAccess.remove_absolute(path) == OK)
	assert(DirAccess.make_dir_absolute(path) == OK)
	var result: Dictionary = ResourceTool.new().execute({"action": "set", "path": path, "props": {"resource_name": "after"}, "evidence": true})
	var data: Dictionary = result.get("data", {})
	if bool(result.get("ok", true)) or data.get("effect") != "applied" or data.get("persistence") != "failed" or cached.resource_name != "after":
		push_error("save failure must preserve applied memory effect and failed persistence: %s" % result)
		quit(1)
		return
	var capture: Dictionary = ViewportActions.screenshot(root, {"evidence": true}, "", OS.get_process_id())
	var evidence: Dictionary = capture.get("data", {}).get("evidence", {})
	if bool(capture.get("ok", true)) or evidence.get("available") != false:
		push_error("headless capture must report unavailable evidence")
		quit(1)
		return
	var plain: Dictionary = ViewportActions.screenshot(root, {}, "", OS.get_process_id())
	if bool(plain.get("ok", true)) or String(plain.get("error", "")).begins_with("evidence_unavailable"):
		push_error("default headless capture must not use evidence_unavailable: %s" % plain)
		quit(1)
		return
	var start := Time.get_ticks_msec()
	Engine.time_scale = 0.0
	var drew: bool = await CaptureEvidence.wait_for_draw(self)
	var elapsed := Time.get_ticks_msec() - start
	Engine.time_scale = 1.0
	if drew or elapsed < 200 or elapsed > 2500:
		push_error("wait_for_draw must honor the 1000 ms deadline at time scale 0 (drew=%s elapsed=%d)" % [drew, elapsed])
		quit(1)
		return
	var inspector := Inspector.new()
	root.add_child(inspector)
	var stale: Dictionary = inspector._handle({"action": "screenshot", "evidence": true, "runtime_session_id": "old"})
	assert(not bool(stale.ok) and String(stale.error).begins_with("session_mismatch"), "old runtime must fail before capture")
	inspector.queue_free()
	print("evidence_test: PASS")
	quit(0)
