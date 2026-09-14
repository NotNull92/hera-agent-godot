extends SceneTree

const EditorLog = preload("res://addons/hera_agent_godot/core/editor_log.gd")
const OutputTool = preload("res://addons/hera_agent_godot/tools/output_tool.gd")
const DiagnosticsTool = preload("res://addons/hera_agent_godot/tools/diagnostics_tool.gd")

var failures: Array[String] = []
var _race_mutex := Mutex.new()
var _race_running := true
var _race_ready := Semaphore.new()
var _late_release := Semaphore.new()

func _initialize() -> void:
	print("before-registration")
	var collector := EditorLog.new()
	collector.start("first")
	_check(collector.capability == "supported", "real initialization callback was not observed")
	if collector.capability != "supported":
		_finish()
		return
	var output := OutputTool.new()
	output.editor_log = collector
	var diagnostics := DiagnosticsTool.new()
	diagnostics.editor_log = collector
	var initial: Dictionary = output.execute({"source": "editor"})["data"]
	_check(initial["total"] == 1, "pre-registration messages must not enter the buffer")
	var cursor: String = initial["cursor"]
	print("ordinary")
	push_warning("warning-evidence")
	push_error("error-evidence")
	_script_error()
	var threads: Array[Thread] = []
	for worker in range(4):
		var thread := Thread.new()
		_check(thread.start(_thread_logs.bind(worker)) == OK, "thread startup failed")
		threads.append(thread)
	for thread in threads:
		thread.wait_to_finish()
	var data: Dictionary = output.execute({"source": "editor", "since": cursor, "lines": 1024})["data"]
	_check(data["total"] == 44, "main-thread and worker logs were lost or duplicated")
	var entries: Array = data["entries"]
	_check(entries[0]["message"].begins_with("ordinary") and entries[0]["severity"] == "log", "normal severity lost")
	_check(entries[1]["severity"] == "warning" and entries[1]["location"]["line"] > 0, "warning location lost")
	_check(entries[2]["severity"] == "error" and entries[2]["error_type"] == 0, "error metadata lost")
	_check(entries[3]["severity"] == "error" and entries[3]["error_type"] == 2, "script error metadata lost")
	var summary: Dictionary = diagnostics.execute({"source": "editor", "since": cursor})["data"]
	_check(summary["error_count"] == 2 and summary["warning_count"] == 1 and not summary["clean"], "severity counts are wrong")
	var empty: Dictionary = diagnostics.execute({"source": "editor", "since": data["cursor"]})["data"]
	_check(empty["available"] and empty["clean"] and empty["error_count"] == 0, "current cursor must yield a clean observed interval")
	for invalid in ["old:0", "first:999999", "broken", "first:-1", "first:01"]:
		_check(output.execute({"source": "editor", "since": invalid})["data"].get("reason", "") == "cursor_expired", "invalid or stale cursor returned a normal empty result")
	for params in [{"lines": 0}, {"lines": 1025}, {"lines": 1.5}, {"lines": "2"}, {"type": []}, {"since": 1}]:
		_check(not collector.read(params, false).get("ok", true), "invalid RPC parameters accepted")
	print("x".repeat(5000))
	var long_entry: Dictionary = output.execute({"source": "editor", "lines": 1})["data"]["entries"][0]
	_check(long_entry["truncated"] and String(long_entry["message"]).length() == 4096, "message storage was not bounded")
	for i in range(1030):
		print("overflow-", i)
	var overflow: Dictionary = output.execute({"source": "editor", "lines": 1024})["data"]
	_check(overflow["total"] == 1024 and overflow["dropped_count"] > 0 and not overflow["complete"], "overflow must bound memory and disclose loss")
	var expired: Dictionary = output.execute({"source": "editor", "since": cursor})["data"]
	_check(expired.get("reason", "") == "cursor_expired" and not expired["available"], "overflow cursor did not expire")
	var restarted: Dictionary = diagnostics.execute({"source": "editor", "since": expired["restart_cursor"]})["data"]
	_check(restarted["complete"] and restarted["clean"], "restart cursor must delimit available history")
	_check(not diagnostics.execute({"source": "editor"})["data"]["clean"], "lost history must not be declared clean")
	var reference: WeakRef = weakref(collector._logger)
	collector.stop()
	print("while-disabled")
	_check(reference.get_ref() == null and not collector.read({}, false)["data"]["available"], "removal leaked the adapter or retained available evidence")
	collector.start("second")
	_check(output.execute({"source": "editor"})["data"]["total"] == 1, "re-enable duplicated callbacks or retained old entries")
	_check(output.execute({"source": "editor", "since": overflow["cursor"]})["data"].get("reason", "") == "cursor_expired", "re-enable accepted an old session")
	collector.stop()
	var retired: WeakRef = _test_close_race(collector)
	_check(retired.get_ref() == null, "retired adapter must be released after the callback and its owning scope exit")
	_finish()

func _test_close_race(collector: RefCounted) -> WeakRef:
	collector.start("race")
	var old_adapter: RefCounted = collector._logger
	var late := Thread.new()
	_check(late.start(_late_callback.bind(old_adapter)) == OK, "late callback worker starts")
	_race_ready.wait()
	var active := Thread.new()
	_check(active.start(_during_close_logs) == OK, "active logging worker starts")
	_race_ready.wait()
	collector.stop()
	collector.start("race")
	_race_mutex.lock()
	_race_running = false
	_race_mutex.unlock()
	active.wait_to_finish()
	var before: Dictionary = collector.read({}, false)["data"]
	_late_release.post()
	late.wait_to_finish()
	late = null
	var after: Dictionary = collector.read({"since": before["cursor"]}, false)["data"]
	_check(after["total"] == 0, "retired callback must neither write nor emit an error after restart")
	var retired: WeakRef = weakref(old_adapter)
	old_adapter = null
	collector.stop()
	return retired

func _late_callback(adapter: RefCounted) -> void:
	_race_ready.post()
	_late_release.wait()
	adapter.call("_log_message", "retired-callback", false)

func _during_close_logs() -> void:
	print("race-worker-start")
	_race_ready.post()
	while true:
		_race_mutex.lock()
		var running := _race_running
		_race_mutex.unlock()
		if not running:
			return
		print("race-worker-active")

func _script_error() -> void:
	var empty: Variant = null
	empty.hera_expected_script_error()

func _thread_logs(worker: int) -> void:
	for i in range(10):
		print("worker-", worker, "-", i)

func _check(condition: bool, failure: String) -> void:
	if not condition:
		failures.append(failure)

func _finish() -> void:
	var evidence := FileAccess.open("res://editor-log-evidence.json", FileAccess.WRITE)
	evidence.store_string(JSON.stringify({"failures": failures}))
	evidence.close()
	quit(0 if failures.is_empty() else 1)
