@tool
extends RefCounted

const Records = preload("res://addons/hera_agent_godot/core/operation_records.gd")
const Gate = preload("res://addons/hera_agent_godot/server/editor_gate.gd")
const Response = preload("res://addons/hera_agent_godot/core/tool_response.gd")

var operations := Records.new()
var gate := Gate.new()
var _items: Array[Dictionary] = []
var _owner: String:
	get:
		return gate._owner

func _init() -> void:
	gate.operations = operations

func set_filesystem(filesystem: Node) -> void:
	gate.set_filesystem(filesystem)

func enqueue(item: Dictionary) -> void:
	var request: Dictionary = item.get("request", {})
	if request.get("tool") == "operation":
		operations.prepare(item)
	_items.append(item)

func configure_operations(session: String) -> void:
	operations.session = session
	gate.operations = operations

func begin(item: Dictionary) -> bool:
	if item.has("response"):
		return false
	if operations.retired:
		item["response"] = Response.rejected("session_mismatch: editor session retired", "session_mismatch")
		return false
	if item.has("operation_id") and (not operations.records.has(item.operation_id) or operations.records[item.operation_id].lifecycle != "accepted"):
		item["response"] = Response.success(operations.lookup(item.operation_id))
		return false
	if not gate.admit(item):
		return false
	if item.has("operation_id") and not operations.begin(item.operation_id):
		gate.release(item)
		item["response"] = Response.success(operations.lookup(item.operation_id))
		return false
	return true

func complete(item: Dictionary, response: Dictionary) -> Dictionary:
	gate.release(item)
	if not item.has("operation_id"):
		return response
	operations.finish(item.operation_id, response)
	return Response.success(operations.lookup(item.operation_id))

func retire() -> void:
	operations.retire()
	gate.retire()
	_items.clear()

func execute_request(request: Dictionary, registry: RefCounted, owner: String = "") -> Dictionary:
	return await gate.execute_request(request, registry, owner)

func _import_busy() -> bool:
	return gate.import_busy()

func drain() -> Array[Dictionary]:
	var drained := _items
	_items = []
	return drained
