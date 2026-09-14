@tool
extends RefCounted

# Main-thread, in-process receipts. IDs fix the session and execution deadline.
# Capacity rejects admission; unexpired IDs are never evicted and re-executed.
const MAX_RECORDS := 128
const MAX_BYTES := 4194304
const MAX_RESULT_BYTES := 16384
const RETENTION_MS := 60000
const MAX_FUTURE_MS := 60000
var session := ""
var records: Dictionary = {}
var expired_count := 0
var expired_unknown_count := 0
var dropped_result_count := 0
var retired := false
var _clock_origin := int(Time.get_unix_time_from_system() * 1000)
var _ticks_origin := Time.get_ticks_msec()

static func id_parts(id: String) -> PackedStringArray:
	var parts := id.split(":")
	if id.length() > 160 or parts.size() != 3 or parts[0].is_empty() or parts[2].is_empty():
		return PackedStringArray()
	if not parts[1].is_valid_int() or str(int(parts[1])) != parts[1] or int(parts[1]) <= 0 or int(parts[1]) > 9007199254740991:
		return PackedStringArray()
	for character in parts[0] + parts[2]:
		if not (character in "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-"):
			return PackedStringArray()
	return parts

func accept(id: String, input: Dictionary) -> Dictionary:
	_prune()
	var parts := id_parts(id)
	if parts.is_empty():
		return _error("invalid_operation", "ID must be session:unix_ms:nonce")
	if retired or (session != "" and parts[0] != session):
		return _error("session_mismatch", "operation belongs to another session")
	var normalized := JSON.stringify({"editor_session_id": parts[0], "request": input}, "", true, true)
	if normalized.to_utf8_buffer().size() > 32768:
		return _error("invalid_operation", "normalized input exceeds byte limit")
	var digest := normalized.sha256_text()
	if records.has(id):
		if records[id].input_digest != digest:
			return _error("operation_id_conflict", "ID already binds another request")
		return {"receipt": lookup(id)}
	var now := _now()
	var deadline := int(parts[1])
	if deadline <= now:
		return _error("operation_expired", "execution deadline has passed; prior effect may be unknown")
	if deadline > now + MAX_FUTURE_MS:
		return _error("invalid_operation", "deadline must be within 60 seconds")
	if records.size() >= MAX_RECORDS or JSON.stringify(records).to_utf8_buffer().size() + 65536 > MAX_BYTES:
		return _error("operation_capacity", "receipt capacity reached; no execution")
	var receipt := {"id": id, "editor_session_id": parts[0], "input_digest": digest, "lifecycle": "accepted", "effect": "not_applied", "verification": "not_requested", "persistence": "not_requested", "error_code": "", "evidence": {}, "cancellable": true, "deadline": deadline, "retain_until": deadline + RETENTION_MS}
	var params: Dictionary = input.get("params", input)
	var expected: Dictionary = params.expected if params.get("expected") is Dictionary else {}
	var runtime: bool = input.get("tool") == "game"
	receipt["target"] = {"scope": "runtime" if runtime else "editor", "session_id": params.get("runtime_session_id", "") if runtime else parts[0], "pid": params.get("pid", params.get("target_pid", OS.get_process_id())), "node_instance_id": expected.get("node_instance_id", ""), "path": params.get("path", "."), "scene": expected.get("scene", ""), "prop": params.get("prop", "")}
	if params.get("verify") == true:
		receipt.verification = "unavailable"
	if runtime and String(params.get("action", "")) == "call":
		receipt.persistence = "unknown"
	records[id] = receipt
	return {"accepted": true}

static func supports_guarded_node_set(input: Dictionary) -> bool:
	var params: Variant = input.get("params", {})
	return input.get("tool") == "node" and params is Dictionary and params.get("action") in ["set", "set_guarded"] and params.get("expected") is Dictionary

static func supports_runtime_mutation(input: Dictionary) -> bool:
	var params: Variant = input.get("params", {})
	if input.get("tool") != "game" or not params is Dictionary or not params.get("action") in ["set", "call"]:
		return false
	if not params.get("runtime_session_id") is String or params.runtime_session_id.is_empty():
		return false
	return typeof(params.get("pid")) in [TYPE_INT, TYPE_FLOAT] and float(params.pid) == floorf(float(params.pid)) and float(params.pid) > 0

func prepare(item: Dictionary) -> void:
	var params: Variant = item.request.get("params")
	if not params is Dictionary or not params.get("id") is String:
		item["response"] = _error("invalid_operation", "params and string id required")
		_fail_item(item)
		return
	var id: String = params.id
	var action: Variant = params.get("action")
	if id_parts(id).is_empty():
		item["response"] = _error("invalid_operation", "malformed ID")
		_fail_item(item)
	elif action == "status":
		item["response"] = {"ok": true, "data": lookup(id)}
	elif action == "cancel":
		item["response"] = {"ok": true, "data": cancel(id)}
	elif action == "submit":
		_prepare_submit(item, id, params)
	else:
		item["response"] = _error("invalid_operation", "action must be submit, status, or cancel")
		_fail_item(item)

func _prepare_submit(item: Dictionary, id: String, params: Dictionary) -> void:
	var raw: Variant = params.get("request")
	if not raw is String or raw.to_utf8_buffer().size() > 16384 or params.get("digest") != raw.sha256_text():
		item["response"] = _error("invalid_operation", "request size or digest mismatch")
		_fail_item(item)
		return
	var json := JSON.new()
	if json.parse(raw) != OK:
		item["response"] = _error("invalid_operation", "malformed request JSON")
		_fail_item(item)
		return
	var input: Variant = json.data
	if not input is Dictionary or not input.get("params") is Dictionary:
		item["response"] = _error("invalid_operation", "request needs tool and params")
		_fail_item(item)
		return
	var inner: Dictionary = input.params
	if not inner.get("path", ".") is String:
		item["response"] = _error("invalid_operation", "path must be a string")
		_fail_item(item)
		return
	if supports_guarded_node_set(input):
		input.params.action = "set_guarded"
	elif not supports_runtime_mutation(input):
		item["response"] = _error("capability_unavailable", "operations support guarded node set or session-targeted game set/call")
		_fail_item(item)
		return
	var accepted := accept(id, input)
	if accepted.has("error"):
		item["response"] = {"ok": false, "error": accepted.error}
	elif accepted.has("receipt"):
		item["response"] = {"ok": true, "data": accepted.receipt}
	else:
		item["operation_id"] = id
		item.request = input
		if input.tool == "game":
			input.params["operation_id"] = id

func _fail_item(item: Dictionary) -> void:
	var response: Dictionary = item.response
	if response.has("error") and not response.has("ok"):
		item.response = {"ok": false, "error": response.error}

func reject(id: String, code: String) -> void:
	if not records.has(id) or records[id].lifecycle != "accepted":
		return
	var receipt: Dictionary = records[id]
	receipt.lifecycle = "rejected"
	receipt.effect = "not_applied"
	receipt.error_code = code
	receipt.cancellable = false

func begin(id: String) -> bool:
	if not records.has(id) or records[id].lifecycle != "accepted":
		return false
	var receipt: Dictionary = records[id]
	if retired or _now() >= int(receipt.deadline):
		reject(id, "session_mismatch" if retired else "operation_expired")
		return false
	receipt.lifecycle = "running"
	receipt.effect = "unknown"
	receipt.cancellable = false
	return true

func finish(id: String, response: Dictionary) -> void:
	if not records.has(id) or retired:
		return
	var receipt: Dictionary = records[id]
	if receipt.lifecycle != "running":
		return
	var error := String(response.get("error", ""))
	var code := String(response.get("error_code", error.get_slice(":", 0)))
	receipt.cancellable = false
	var response_data: Variant = response.get("data", {})
	if response_data is Dictionary and response_data.get("runtime_receipt") is Dictionary:
		var runtime: Dictionary = response_data.runtime_receipt
		for field: String in ["lifecycle", "effect", "verification", "persistence", "error_code", "evidence"]:
			receipt[field] = runtime[field]
		_bound_evidence(receipt)
		return
	if bool(response.get("ok", false)):
		receipt.lifecycle = "completed"
		receipt.effect = "applied"
		var data: Variant = response.get("data", {})
		if data is Dictionary:
			receipt.verification = data.get("verification", "not_requested")
	elif code in ["verification_failed", "verification_unavailable"]:
		receipt.lifecycle = "completed"
		receipt.effect = "applied"
		receipt.verification = "failed" if code == "verification_failed" else "unavailable"
		receipt.error_code = code
	elif response.get("attempted") == false:
		receipt.lifecycle = "rejected"
		receipt.effect = "not_applied"
		receipt.error_code = code if code != "" and code != error else "invalid_operation"
	else:
		receipt.lifecycle = "outcome_unknown"
		receipt.effect = "unknown"
		receipt.error_code = "outcome_unknown"
	if JSON.stringify(response).to_utf8_buffer().size() <= MAX_RESULT_BYTES:
		receipt.evidence = {"response": response.duplicate(true), "complete": true}
	else:
		dropped_result_count += 1
		receipt.evidence = {"complete": false, "reason": "result_byte_limit"}
	_bound_evidence(receipt)

func _bound_evidence(receipt: Dictionary) -> void:
	if JSON.stringify(records).to_utf8_buffer().size() > MAX_BYTES:
		dropped_result_count += 1
		receipt.evidence = {"complete": false, "reason": "record_byte_limit"}

func lookup(id: String) -> Dictionary:
	_prune()
	var receipt: Dictionary
	if records.has(id):
		receipt = records[id].duplicate(true)
	else:
		var parts := id_parts(id)
		var code := "operation_unknown"
		if parts.is_empty():
			code = "invalid_operation"
		elif session != "" and parts[0] != session:
			code = "session_mismatch"
		elif int(parts[1]) + RETENTION_MS <= _now():
			code = "operation_retention_expired"
		receipt = {"id": id, "editor_session_id": session, "input_digest": "", "target": {}, "lifecycle": "outcome_unknown", "effect": "unknown", "verification": "unavailable", "persistence": "unknown", "error_code": code, "evidence": {}, "cancellable": false}
	receipt["retention"] = {"durable": false, "max_records": MAX_RECORDS, "max_bytes": MAX_BYTES, "after_deadline_ms": RETENTION_MS, "expired_count": expired_count, "expired_unknown_count": expired_unknown_count, "dropped_result_count": dropped_result_count, "restart": "history_lost"}
	return receipt

func cancel(id: String) -> Dictionary:
	_prune()
	if records.has(id) and records[id].lifecycle == "accepted":
		records[id].lifecycle = "cancelled"
		records[id].cancellable = false
		records[id].error_code = "operation_cancelled"
	var receipt := lookup(id)
	receipt["cancelled"] = receipt.lifecycle == "cancelled"
	return receipt

func retire() -> void:
	retired = true
	for receipt: Dictionary in records.values():
		if receipt.lifecycle == "running":
			receipt.lifecycle = "outcome_unknown"
			receipt.error_code = "outcome_unknown"
		elif receipt.lifecycle == "accepted":
			receipt.lifecycle = "cancelled"
			receipt.error_code = "session_mismatch"
		receipt.cancellable = false

func _prune() -> void:
	var now := _now()
	for id: String in records.keys():
		if now >= int(records[id].retain_until):
			if records[id].lifecycle in ["running", "outcome_unknown"]:
				expired_unknown_count += 1
			records.erase(id)
			expired_count += 1

func _now() -> int:
	return _clock_origin + Time.get_ticks_msec() - _ticks_origin

func _error(code: String, detail: String) -> Dictionary:
	return {"error": "%s: %s" % [code, detail]}
