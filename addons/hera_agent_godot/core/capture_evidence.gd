@tool
extends RefCounted

const Operations = preload("res://addons/hera_agent_godot/core/operation_records.gd")
const DRAW_WAIT_SEC := 1.0

static func guard(params: Dictionary) -> String:
	if params.has("correlation_operation_id"):
		if not params.correlation_operation_id is String or Operations.id_parts(params.correlation_operation_id).is_empty():
			return "invalid_evidence: correlation_operation_id must be session:unix_ms:nonce"
	return ""

static func snapshot(source: String, session: String) -> Dictionary:
	var data := {"available": true, "source": source, "captured_at_unix_ms": int(Time.get_unix_time_from_system() * 1000), "process_frame": Engine.get_process_frames(), "physics_frame": Engine.get_physics_frames(), "draw_frame": Engine.get_frames_drawn(), "atomic_snapshot": false}
	data["runtime_session_id" if source == "runtime" else "editor_session_id"] = session
	return data

static func unavailable(reason: String) -> Dictionary:
	return {"ok": false, "error": "evidence_unavailable: " + reason, "data": {"evidence": {"available": false, "reason": reason}}}

static func capture_error(params: Dictionary, reason: String) -> Dictionary:
	if bool(params.get("evidence", false)):
		return unavailable(reason)
	return {"ok": false, "error": reason}

static func link(evidence: Dictionary, params: Dictionary) -> void:
	if params.has("correlation_operation_id"):
		evidence["operation"] = {"id": params.correlation_operation_id, "association": "caller_supplied"}

static func wait_for_draw(tree: SceneTree) -> bool:
	if tree == null:
		return false
	var seen: Array[bool] = [false]
	var callback := func() -> void: seen[0] = true
	RenderingServer.frame_post_draw.connect(callback, CONNECT_ONE_SHOT)
	var timer := tree.create_timer(DRAW_WAIT_SEC, true, false, true)
	while not seen[0] and timer.time_left > 0.0:
		await tree.process_frame
	if RenderingServer.frame_post_draw.is_connected(callback):
		RenderingServer.frame_post_draw.disconnect(callback)
	return seen[0]
