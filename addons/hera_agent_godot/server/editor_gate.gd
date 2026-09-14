@tool
extends RefCounted

const Response = preload("res://addons/hera_agent_godot/core/tool_response.gd")

var operations: RefCounted
var _owner := ""
var _sequence := 0
var _filesystem: Node
var _scan_pending := false
var _scan_finishing := false

func set_filesystem(filesystem: Node) -> void:
	_filesystem = filesystem
	_filesystem.connect("filesystem_changed", _scan_completed)
	_filesystem.connect("sources_changed", _scan_completed)

func _scan_completed(_changed: bool = false) -> void:
	_scan_pending = false
	_scan_finishing = true

func admit(item: Dictionary) -> bool:
	if _safe_read(item.request):
		return true
	if not _owner.is_empty() or import_busy():
		if item.has("operation_id"):
			operations.reject(item.operation_id, "editor_busy")
			item["response"] = Response.success(operations.lookup(item.operation_id))
		else:
			item["response"] = Response.rejected("editor_busy: editor mutation or import in progress", "editor_busy")
		return false
	_sequence += 1
	item["gate_owner"] = operations.session + ":" + String(item.get("operation_id", "request-%d" % _sequence))
	_owner = item.gate_owner
	return true

func release(item: Dictionary) -> void:
	if item.get("gate_owner", "") == _owner:
		_owner = ""

func retire() -> void:
	_owner = ""
	if is_instance_valid(_filesystem):
		_filesystem.disconnect("filesystem_changed", _scan_completed)
		_filesystem.disconnect("sources_changed", _scan_completed)
	_filesystem = null
	_scan_pending = false
	_scan_finishing = false

func _safe_read(request: Dictionary) -> bool:
	var name: Variant = request.get("tool")
	var params: Variant = request.get("params", {})
	return name == "status" or (name in ["output", "diagnostics"] and params is Dictionary and params.get("source") == "editor")

func import_busy() -> bool:
	if not is_instance_valid(_filesystem):
		return false
	# The scan worker clears is_scanning before the main thread installs its
	# result. Processing remains enabled until that commit starts; retain the
	# observed scan until a completion event AND idle state, including reentry.
	var processing: bool = _filesystem.is_scanning() or _filesystem.is_processing()
	if not processing:
		_scan_finishing = false
	elif not _scan_finishing:
		_scan_pending = true
	return processing or _scan_pending or (_filesystem.has_method("is_importing") and bool(_filesystem.call("is_importing")))

func execute_request(request: Dictionary, registry: RefCounted, owner: String = "") -> Dictionary:
	if operations.retired or (not owner.is_empty() and owner != _owner):
		return Response.rejected("session_mismatch: editor execution ownership ended", "session_mismatch")
	if not _safe_read(request) and (owner.is_empty() or import_busy()):
		return Response.rejected("editor_busy: editor mutation or import in progress", "editor_busy")
	var name := String(request.get("tool", ""))
	var tool: Variant = registry.resolve(name)
	if tool == null:
		return Response.rejected("missing tool name" if name.is_empty() else "unknown tool: %s" % name, "invalid_operation")
	var params: Variant = request.get("params", {})
	if not params is Dictionary:
		params = {}
	var result: Variant
	if name == "batch":
		result = await tool.execute_async(params, Callable(self, "execute_request").bind(registry, owner))
	elif tool.has_method("execute_async"):
		result = await tool.execute_async(params)
	else:
		result = tool.execute(params)
	# A scan can outlive a synchronous tool. Keep ownership through engine work,
	# including failed tools; a disconnected client is not execution cancellation.
	while not operations.retired and not owner.is_empty() and import_busy():
		await _filesystem.get_tree().process_frame
	if operations.retired:
		return Response.failure("session_mismatch: editor session retired during execution")
	if not result is Dictionary or not result.get("ok") is bool:
		return Response.failure("outcome_unknown: tool returned no valid result")
	return result
