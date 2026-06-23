@tool
extends EditorPlugin

const ToolRegistry = preload("res://addons/hera_agent_godot/core/tool_registry.gd")
const ToolResponse = preload("res://addons/hera_agent_godot/core/tool_response.gd")
const HttpServer = preload("res://addons/hera_agent_godot/server/http_server.gd")
const WorkQueue = preload("res://addons/hera_agent_godot/server/work_queue.gd")
const Heartbeat = preload("res://addons/hera_agent_godot/server/heartbeat.gd")
const StatusTool = preload("res://addons/hera_agent_godot/tools/status_tool.gd")
const RunTool = preload("res://addons/hera_agent_godot/tools/run_tool.gd")
const SceneTool = preload("res://addons/hera_agent_godot/tools/scene_tool.gd")
const NodeTool = preload("res://addons/hera_agent_godot/tools/node_tool.gd")
const SignalTool = preload("res://addons/hera_agent_godot/tools/signal_tool.gd")
const EvalTool = preload("res://addons/hera_agent_godot/tools/eval_tool.gd")
const OutputTool = preload("res://addons/hera_agent_godot/tools/output_tool.gd")
const ScreenshotTool = preload("res://addons/hera_agent_godot/tools/screenshot_tool.gd")
const BatchTool = preload("res://addons/hera_agent_godot/tools/batch_tool.gd")

const HEARTBEAT_INTERVAL := 0.5

var _registry: RefCounted
var _server: RefCounted
var _queue: RefCounted
var _heartbeat: RefCounted
var _heartbeat_accum := 0.0

func _enter_tree() -> void:
	set_process(true)
	_registry = ToolRegistry.new()
	_registry.register(StatusTool.new())
	_registry.register(RunTool.new())
	_registry.register(SceneTool.new())
	var node_tool := NodeTool.new()
	node_tool.set_undo_redo(get_undo_redo())
	_registry.register(node_tool)
	var signal_tool := SignalTool.new()
	signal_tool.set_undo_redo(get_undo_redo())
	_registry.register(signal_tool)
	_registry.register(EvalTool.new())
	_registry.register(OutputTool.new())
	var screenshot_tool := ScreenshotTool.new()
	screenshot_tool.set_host(self)
	_registry.register(screenshot_tool)
	var batch_tool := BatchTool.new()
	batch_tool.set_registry(_registry)
	_registry.register(batch_tool)

	_queue = WorkQueue.new()
	_server = HttpServer.new()
	var bound: int = _server.start(8770)
	if bound == 0:
		_server = null
		push_error("[hera] failed to bind HTTP server on 127.0.0.1 (8770-8785)")
		return

	_heartbeat = Heartbeat.new()
	_heartbeat.start(bound)
	print("[hera] Hera Agent Godot listening on 127.0.0.1:%d" % bound)

func _process(delta: float) -> void:
	if _server != null:
		_server.poll(_queue)
		for item in _queue.drain():
			_handle(item)

	if _heartbeat != null:
		_heartbeat_accum += delta
		if _heartbeat_accum >= HEARTBEAT_INTERVAL:
			_heartbeat_accum = 0.0
			_heartbeat.write()

func _exit_tree() -> void:
	set_process(false)
	if _heartbeat != null:
		_heartbeat.stop()
		_heartbeat = null
	if _server != null:
		_server.stop()
		_server = null
	_queue = null
	_registry = null
	print("[hera] Hera Agent Godot exited")

# Respond to one queued request. Tools exposing execute_async() are awaited
# (e.g. screenshot needs a rendered frame); everything else runs synchronously.
func _handle(item: Dictionary) -> void:
	var request: Dictionary = item["request"]
	var tool_name := String(request.get("tool", ""))
	var tool = _registry.resolve(tool_name) if tool_name != "" else null
	if tool != null and tool.has_method("execute_async"):
		var params: Variant = request.get("params", {})
		if typeof(params) != TYPE_DICTIONARY:
			params = {}
		var response: Dictionary = await tool.execute_async(params)
		if _server != null:
			_server.respond(item["conn"], response)
		else:
			(item["conn"] as StreamPeerTCP).disconnect_from_host()
	else:
		_server.respond(item["conn"], _dispatch(request))

func _dispatch(request: Dictionary) -> Dictionary:
	var tool_name := String(request.get("tool", ""))
	if tool_name == "":
		return ToolResponse.failure("missing tool name")
	var tool = _registry.resolve(tool_name)
	if tool == null:
		return ToolResponse.failure("unknown tool: %s" % tool_name)
	var params: Variant = request.get("params", {})
	if typeof(params) != TYPE_DICTIONARY:
		params = {}
	return tool.execute(params)
