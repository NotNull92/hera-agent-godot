@tool
extends RefCounted

const Records = preload("res://addons/hera_agent_godot/core/operation_records.gd")
const Response = preload("res://addons/hera_agent_godot/core/tool_response.gd")
var operations := Records.new()

var _items: Array[Dictionary] = []
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

func enqueue(item: Dictionary) -> void:
	var request: Dictionary = item.get("request", {})
	if request.get("tool") == "operation":
		_prepare_operation(item)
	_items.append(item)

func configure_operations(session: String) -> void:
	operations.session = session

func _prepare_operation(item: Dictionary) -> void:
	var params: Variant = item.request.get("params")
	if not params is Dictionary or not params.get("id") is String:
		item["response"] = Response.failure("invalid_operation: params and string id required")
		return
	var id: String = params.id
	var action: Variant = params.get("action")
	if Records.id_parts(id).is_empty():
		item["response"] = Response.failure("invalid_operation: malformed ID")
	elif action == "status":
		item["response"] = Response.success(operations.lookup(id))
	elif action == "cancel":
		item["response"] = Response.success(operations.cancel(id))
	elif action == "submit":
		var raw: Variant = params.get("request")
		if not raw is String or raw.to_utf8_buffer().size() > 16384 or params.get("digest") != raw.sha256_text():
			item["response"] = Response.failure("invalid_operation: request size or digest mismatch")
			return
		var json := JSON.new()
		if json.parse(raw) != OK:
			item["response"] = Response.failure("invalid_operation: malformed request JSON")
			return
		var input: Variant = json.data
		if not input is Dictionary or not input.get("params") is Dictionary:
			item["response"] = Response.failure("invalid_operation: request needs tool and params")
			return
		var p: Dictionary = input.params
		if not p.get("path", ".") is String:
			item["response"] = Response.failure("invalid_operation: path must be a string")
			return
		if input.get("tool") == "node" and p.get("action") in ["set", "set_guarded"] and p.get("expected") is Dictionary:
			input.params.action = "set_guarded"
		elif input.get("tool") == "game" and p.get("action") in ["set", "call"] and p.get("runtime_session_id") is String and not p.runtime_session_id.is_empty() and typeof(p.get("pid")) in [TYPE_INT, TYPE_FLOAT] and float(p.pid) == floorf(float(p.pid)) and float(p.pid) > 0:
			pass
		else:
			item["response"] = Response.failure("capability_unavailable: operations support guarded node set or session-targeted game set/call")
			return
		var accepted := operations.accept(id, input)
		if accepted.has("error"):
			item["response"] = Response.failure(accepted.error)
		elif accepted.has("receipt"):
			item["response"] = Response.success(accepted.receipt)
		else:
			item["operation_id"] = id
			item.request = input
			if input.tool == "game":
				input.params["operation_id"] = id
				if p.action == "call":
					operations.records[id].persistence = "unknown"
	else:
		item["response"] = Response.failure("invalid_operation: action must be submit, status, or cancel")

func begin(item: Dictionary) -> bool:
	if item.has("response"):
		return false
	if item.has("operation_id") and not operations.begin(item.operation_id):
		item["response"] = Response.success(operations.lookup(item.operation_id))
		return false
	if operations.retired:
		item["response"] = Response.failure("session_mismatch: editor session retired")
		return false
	if not _safe_read(item.request):
		if not _owner.is_empty() or _import_busy():
			item["response"] = complete(item, Response.failure("editor_busy: editor mutation or import in progress"))
			return false
		_sequence += 1
		item["gate_owner"] = operations.session + ":" + String(item.get("operation_id", "request-%d" % _sequence))
		_owner = item.gate_owner
	return true

func complete(item: Dictionary, response: Dictionary) -> Dictionary:
	if item.get("gate_owner", "") == _owner:
		_owner = ""
	if not item.has("operation_id"):
		return response
	operations.finish(item.operation_id, response)
	return Response.success(operations.lookup(item.operation_id))

func retire() -> void:
	operations.retire()
	_owner = ""
	_items.clear()
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

func _import_busy() -> bool:
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
		return Response.failure("session_mismatch: editor execution ownership ended")
	if not _safe_read(request) and (owner.is_empty() or _import_busy()):
		return Response.failure("editor_busy: editor mutation or import in progress")
	var name := String(request.get("tool", ""))
	var tool: Variant = registry.resolve(name)
	if tool == null:
		return Response.failure("missing tool name" if name.is_empty() else "unknown tool: %s" % name)
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
	while not operations.retired and not owner.is_empty() and _import_busy():
		await _filesystem.get_tree().process_frame
	if operations.retired:
		return Response.failure("session_mismatch: editor session retired during execution")
	if not result is Dictionary or not result.get("ok") is bool:
		return Response.failure("outcome_unknown: tool returned no valid result")
	return result

func drain() -> Array[Dictionary]:
	var drained := _items
	_items = []
	return drained
