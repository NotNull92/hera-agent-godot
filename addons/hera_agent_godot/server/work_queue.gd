@tool
extends RefCounted

const Records = preload("res://addons/hera_agent_godot/core/operation_records.gd")
const Response = preload("res://addons/hera_agent_godot/core/tool_response.gd")
var operations := Records.new()

var _items: Array[Dictionary] = []

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
	return true

func complete(item: Dictionary, response: Dictionary) -> Dictionary:
	if not item.has("operation_id"):
		return response
	operations.finish(item.operation_id, response)
	return Response.success(operations.lookup(item.operation_id))

func drain() -> Array[Dictionary]:
	var drained := _items
	_items = []
	return drained
