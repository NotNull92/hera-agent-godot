extends SceneTree

const WorkQueue = preload("res://addons/hera_agent_godot/server/work_queue.gd")
var failed := false

func _initialize() -> void:
	_run.call_deferred()

func _run() -> void:
	var queue := WorkQueue.new()
	if not queue.has_method("configure_operations"):
		_check(false, "queue must retain operation receipts before dispatch")
	else:
		queue.configure_operations("session")
		var now := int(Time.get_unix_time_from_system() * 1000)
		var id := "session:%d:counter" % (now + 30000)
		var input := {"tool": "game", "params": {"action": "call", "pid": 42, "runtime_session_id": "runtime", "path": "/root/Counter", "method": "increment"}}
		var request := _submit(id, input)
		queue.enqueue({"request": request})
		_check(queue.operations.lookup(id).lifecycle == "accepted", "receipt exists before execution")
		var item: Dictionary = queue.drain()[0]
		_check(queue.begin(item), "first request starts")
		var count := 1
		queue.complete(item, {"ok": true, "data": {"result": count}})
		# Test-only response loss: completed result is deliberately never delivered.
		queue.enqueue({"request": request})
		item = queue.drain()[0]
		if queue.begin(item):
			count += 1
		_check(count == 1 and item.response.data.effect == "applied", "lost response retry returns applied receipt without dispatch")
		input.params.method = "other"
		queue.enqueue({"request": _submit(id, input)})
		_check(queue.drain()[0].response.error.begins_with("operation_id_conflict:"), "same ID different input conflicts")
		var queued_id := "session:%d:cancel" % (now + 30000)
		queue.enqueue({"request": _submit(queued_id, input)})
		queue.enqueue({"request": {"tool": "operation", "params": {"action": "cancel", "id": queued_id}}})
		var pending: Array = queue.drain()
		_check(not queue.begin(pending[0]), "queued cancellation prevents execution")
		_check(pending[1].response.data.effect == "not_applied", "cancel preserves no effect")
		_check(queue.operations.cancel(id).effect == "applied", "cancel after complete preserves actual effect")
		var late_id := "session:%d:late" % (now + 30000)
		queue.enqueue({"request": _submit(late_id, input)})
		item = queue.drain()[0]
		queue.operations.records[late_id].deadline = now - 1
		_check(not queue.begin(item) and item.response.data.error_code == "operation_expired", "late expiry prevents dispatch")
		_check(queue.operations.lookup("old:1:lost").error_code == "session_mismatch", "old session cannot imply success")
		_check(queue.operations.lookup("session:1:lost").effect == "unknown", "expired missing receipt never implies no effect")
		var running_id := "session:%d:running" % (now + 30000)
		queue.enqueue({"request": _submit(running_id, input)})
		item = queue.drain()[0]
		_check(queue.begin(item), "race fixture starts")
		_check(not queue.operations.cancel(running_id).cancelled, "running mutation cannot truthfully cancel")
		queue.complete(item, {"ok": true, "data": {}})
		_check(queue.operations.lookup(running_id).effect == "applied", "completion wins after rejected running cancel")
		queue.operations.records[id].retain_until = 0
		_check(queue.operations.lookup(id).effect == "unknown" and queue.operations.expired_count == 1, "eviction reports unknown history")
		var lost_id := "session:%d:lost" % (now + 30000)
		queue.enqueue({"request": _submit(lost_id, input)})
		item = queue.drain()[0]
		queue.begin(item)
		queue.operations.retire()
		_check(queue.operations.lookup(lost_id).lifecycle == "outcome_unknown", "process retirement cannot infer a running effect")
		var bounded := WorkQueue.new()
		bounded.configure_operations("session")
		for i in bounded.operations.MAX_RECORDS:
			bounded.operations.accept("session:%d:n%d" % [now + 30000, i], input)
		_check(bounded.operations.accept("session:%d:overflow" % (now + 30000), input).error.begins_with("operation_capacity:"), "capacity refuses new execution without evicting IDs")
		var first_id := "session:%d:n0" % (now + 30000)
		bounded.operations.begin(first_id)
		bounded.operations.finish(first_id, {"ok": true, "data": "x".repeat(32768)})
		var retained := bounded.operations.lookup(first_id)
		_check(retained.effect == "applied" and not retained.evidence.complete and retained.retention.dropped_result_count == 1, "oversized result drops evidence, preserves effect and dedup record")
		bounded.operations._clock_origin += 100000
		retained = bounded.operations.lookup(first_id)
		_check(retained.error_code == "operation_retention_expired", "real retention expiry identifies missing history")
		_check(bounded.operations.accept(first_id, input).error.begins_with("operation_expired:"), "retention-expired ID cannot become fresh execution")
	print("operation_receipts_test: ", "FAIL" if failed else "PASS")
	quit(1 if failed else 0)

func _submit(id: String, input: Dictionary) -> Dictionary:
	var text := JSON.stringify(input)
	return {"tool": "operation", "params": {"action": "submit", "id": id, "request": text, "digest": text.sha256_text()}}

func _check(value: bool, message: String) -> void:
	if not value:
		failed = true
		push_error(message)
