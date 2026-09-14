extends SceneTree

const Records = preload("res://addons/hera_agent_godot/core/operation_records.gd")
const Queue = preload("res://addons/hera_agent_godot/server/work_queue.gd")
var failed := false

func _initialize() -> void:
	_run.call_deferred()

func _run() -> void:
	var ledger := Records.new()
	ledger.session = "session"
	var deadline := int(Time.get_unix_time_from_system() * 1000) + 60000
	var input := {"tool": "game", "params": {"path": "x".repeat(20000)}}
	for i in Records.MAX_RECORDS:
		_check(ledger.accept("session:%d:n%d" % [deadline, i], input).has("accepted"), "bounded fixture admitted")
	for id: String in ledger.records.keys():
		ledger.begin(id)
		ledger.finish(id, {"ok": true, "data": "y".repeat(15000)})
		_check(JSON.stringify(ledger.records).to_utf8_buffer().size() <= Records.MAX_BYTES, "aggregate serialized record bytes stay bounded")
	_check(ledger.dropped_result_count > 0 and ledger.records.size() == Records.MAX_RECORDS, "byte pressure drops evidence, never live IDs")
	var queue := Queue.new()
	queue.configure_operations("session")
	var raw := "{"
	queue.enqueue({"request": {"tool": "operation", "params": {"action": "submit", "id": "session:%d:invalid" % deadline, "request": raw, "digest": raw.sha256_text()}}})
	_check(queue.drain()[0].response.error.begins_with("invalid_operation:"), "malformed JSON is quietly rejected")
	raw = '{"tool":"eval","params":{"expr":"1"}}'
	queue.enqueue({"request": {"tool": "operation", "params": {"action": "submit", "id": "session:%d:unsupported" % deadline, "request": raw, "digest": raw.sha256_text()}}})
	_check(queue.drain()[0].response.error.begins_with("capability_unavailable:"), "unsupported tool cannot acquire generic receipt guarantee")
	queue.enqueue({"request": {"tool": "operation", "params": {"action": "submit", "id": "session:%d:digest" % deadline, "request": raw, "digest": "wrong"}}})
	_check(queue.drain()[0].response.error.begins_with("invalid_operation:"), "wrong caller digest rejected before admission")
	_check(queue.operations.records.is_empty(), "invalid inputs never create runnable records")
	print("operation_limits_test: ", "FAIL" if failed else "PASS")
	quit(1 if failed else 0)

func _check(value: bool, message: String) -> void:
	if not value:
		failed = true
		push_error(message)
