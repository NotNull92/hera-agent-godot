@tool
extends RefCounted

const ToolResponse = preload("res://addons/hera_agent_godot/core/tool_response.gd")
const CAPACITY := 1024
const MESSAGE_LIMIT := 4096
# Fixed source is compiled only after API checks; older engines never parse Logger.
const LOGGER_SOURCE := """@tool
extends Logger
var sink: WeakRef
var lease: int
func _log_message(message: String, error: bool) -> void:
	var target: RefCounted = sink.get_ref()
	if target != null:
		target.record(lease, message, "error" if error else "log")
func _log_error(function: String, file: String, line: int, code: String, rationale: String, _editor_notify: bool, error_type: int, _script_backtraces: Array[ScriptBacktrace]) -> void:
	var severity := "warning" if error_type == Logger.ERROR_TYPE_WARNING else "error"
	var target: RefCounted = sink.get_ref()
	if target != null:
		target.record(lease, rationale if not rationale.is_empty() else code, severity, function, file, line, error_type)
"""

var capability := "unverified"
var editor_session_id := ""
var _logger: RefCounted
var _mutex := Mutex.new()
var _entries: Array[Dictionary] = []
var _sequence := 0
var _lease := 0

func start(session: String) -> void:
	stop()
	editor_session_id = session
	capability = "unsupported"
	if not ClassDB.class_exists("Logger") or not OS.has_method("add_logger") or not OS.has_method("remove_logger"):
		return
	capability = "unverified"
	var adapter := GDScript.new()
	adapter.source_code = LOGGER_SOURCE
	if adapter.reload() != OK:
		return
	_entries.resize(CAPACITY)
	_logger = adapter.new()
	_logger.set("sink", weakref(self))
	_logger.set("lease", _lease)
	OS.call("add_logger", _logger)
	print("[hera] editor log capture registered")
	_mutex.lock()
	var observed := _sequence > 0
	_mutex.unlock()
	if observed:
		capability = "supported"
	else:
		stop()

func stop() -> void:
	_mutex.lock()
	_lease += 1
	_entries.clear()
	_sequence = 0
	_mutex.unlock()
	if _logger != null:
		OS.call("remove_logger", _logger)
		_logger = null
	capability = "unverified"

func record(lease: int, message: String, severity: String, function: String = "", file: String = "", line: int = 0, error_type: int = -1) -> void:
	_mutex.lock()
	if lease != _lease or _entries.is_empty():
		_mutex.unlock()
		return
	var entry := {"message": message.left(MESSAGE_LIMIT), "severity": severity,
		"location": {"function": function.left(512), "file": file.left(512), "line": line},
		"error_type": error_type, "truncated": message.length() > MESSAGE_LIMIT or function.length() > 512 or file.length() > 512}
	_sequence += 1
	entry["sequence"] = _sequence
	_entries[(_sequence - 1) % CAPACITY] = entry
	_mutex.unlock()

static func unavailable(session: String = "", state: String = "unverified") -> Dictionary:
	return {"source": "editor", "editor_session_id": session, "available": false,
		"clean": false, "reason": "evidence_unavailable", "capability": state,
		"hint": "Editor logs require a registered collector. Pre-registration and external-process logs are unavailable; retain startup --log-file capture."}

func read(params: Dictionary, diagnostics: bool) -> Dictionary:
	if capability != "supported":
		return ToolResponse.success(unavailable(editor_session_id, capability))
	var max_lines: Variant = params.get("lines", 20 if diagnostics else 100)
	if typeof(max_lines) not in [TYPE_INT, TYPE_FLOAT] or not is_finite(float(max_lines)) or float(max_lines) != floor(float(max_lines)) or max_lines < 1 or max_lines > CAPACITY:
		return ToolResponse.failure("editor source lines must be an integer from 1 to %d" % CAPACITY)
	var type_filter: Variant = params.get("type", "all")
	if type_filter not in ["all", "log", "warning", "error"]:
		return ToolResponse.failure("invalid editor log type")
	if params.has("since") and (typeof(params["since"]) != TYPE_STRING or String(params["since"]).is_empty()):
		return ToolResponse.failure("since must be a nonempty editor cursor")
	_mutex.lock()
	var latest := _sequence
	var first: int = max(1, latest - CAPACITY + 1)
	var snapshot: Array[Dictionary] = []
	for seq in range(first, latest + 1):
		snapshot.append(_entries[(seq - 1) % CAPACITY])
	_mutex.unlock()
	var data := {"source": "editor", "editor_session_id": editor_session_id, "available": true,
		"cursor": "%s:%d" % [editor_session_id, latest], "restart_cursor": "%s:%d" % [editor_session_id, first - 1],
		"dropped_count": first - 1, "complete": first == 1 or params.has("since")}
	var after := first - 1
	if params.has("since"):
		var cursor := String(params["since"]).split(":")
		if cursor.size() != 2 or cursor[0] != editor_session_id or not cursor[1].is_valid_int() or str(int(cursor[1])) != cursor[1] or int(cursor[1]) < first - 1 or int(cursor[1]) > latest:
			data.merge({"available": false, "clean": false, "complete": false, "reason": "cursor_expired"}, true)
			return ToolResponse.success(data)
		after = int(cursor[1])
	var entries: Array[Dictionary] = []
	var errors: Array[Dictionary] = []
	var warnings: Array[Dictionary] = []
	for entry in snapshot:
		if int(entry["sequence"]) <= after:
			continue
		if entry["severity"] == "error":
			errors.append(entry)
		elif entry["severity"] == "warning":
			warnings.append(entry)
		if type_filter == "all" or entry["severity"] == type_filter:
			entries.append(entry)
	if diagnostics:
		data.merge({"clean": data["complete"] and errors.is_empty() and warnings.is_empty(),
			"error_count": errors.size(), "warning_count": warnings.size(), "total_lines": entries.size(),
			"errors": errors.slice(max(0, errors.size() - int(max_lines))), "warnings": warnings.slice(max(0, warnings.size() - int(max_lines)))})
	else:
		data.merge({"type": type_filter, "total": entries.size(), "omitted_count": max(0, entries.size() - int(max_lines)),
			"entries": entries.slice(max(0, entries.size() - int(max_lines)))})
	return ToolResponse.success(data)
