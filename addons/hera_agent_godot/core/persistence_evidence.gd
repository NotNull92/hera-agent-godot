@tool
extends RefCounted

static func observe(path: String) -> Dictionary:
	var result := {"available": false, "path": path, "observed_at_unix_ms": int(Time.get_unix_time_from_system() * 1000)}
	var file := FileAccess.open(path, FileAccess.READ)
	if file == null:
		result["reason"] = "file is not readable"
		return result
	var length := file.get_length()
	if length > 67108864:
		result["reason"] = "file exceeds 64 MiB evidence limit"
		return result
	var bytes := file.get_buffer(length)
	if bytes.size() != length or file.get_error() not in [OK, ERR_FILE_EOF]:
		result["reason"] = "incomplete file read"
		return result
	var hash := HashingContext.new()
	hash.start(HashingContext.HASH_SHA256)
	hash.update(bytes)
	result["available"] = true
	result["sha256"] = hash.finish().hex_encode()
	result["bytes"] = length
	return result

static func guard(params: Dictionary, disk: Dictionary) -> Dictionary:
	if not params.has("expected_sha256"):
		return {}
	var expected: Variant = params.expected_sha256
	if not expected is String or expected.length() != 64 or not expected.is_valid_hex_number():
		return {"ok": false, "error": "invalid_evidence: expected_sha256 must be a SHA-256 hash"}
	if not bool(disk.get("available", false)) or disk.get("sha256") != expected.to_lower():
		return {"ok": false, "error": "state_conflict: disk hash changed or is unavailable", "data": {"effect": "not_applied", "persistence": "not_requested", "save_call": "not_called", "evidence": {"available": false, "disk_before": disk}}}
	return {}

static func finish(response: Dictionary, observation: Dictionary) -> Dictionary:
	var disk := observe(String(observation.path))
	var saved: bool = observation.save_call == "succeeded"
	var data: Dictionary = response.get("data", {})
	data["effect"] = observation.effect
	data["save_call"] = observation.save_call
	var verification: Dictionary = observation.get("verification", {"available": false, "reason": "scene memory equivalence is not observed"})
	data["persistence"] = "unknown" if saved else "failed"
	if saved and bool(verification.get("available", false)):
		data["persistence"] = "saved" if bool(verification.get("matches", false)) else "failed"
	data["evidence"] = {"available": bool(disk.available), "disk_before": observation.before, "disk_after": disk, "memory_disk_equivalence": verification, "atomic_snapshot": false}
	response["data"] = data
	if saved and data.persistence == "failed":
		response["ok"] = false
		response["error"] = "save_failed: disk properties differ after the save call"
	if saved and not bool(disk.available):
		response["ok"] = false
		response["error"] = "evidence_unavailable: save returned success but file could not be observed"
	return response

static func verify_properties(path: String, expected: Dictionary) -> Dictionary:
	for value: Variant in expected.values():
		if typeof(value) in [TYPE_OBJECT, TYPE_ARRAY, TYPE_DICTIONARY, TYPE_CALLABLE, TYPE_SIGNAL]:
			return {"available": false, "reason": "selected property needs recursive resource verification"}
	var disk: Resource = ResourceLoader.load(path, "", ResourceLoader.CACHE_MODE_IGNORE)
	if disk == null:
		return {"available": false, "reason": "fresh resource load failed"}
	var matches := true
	for property: String in expected:
		if disk.get(property) != expected[property]:
			matches = false
	return {"available": true, "matches": matches, "scope": "selected_properties", "properties": expected.keys()}
