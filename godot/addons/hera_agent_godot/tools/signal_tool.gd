extends RefCounted

# `signal` — inspect and wire node signals in the edited scene.
#   list <node>                            -> signals the node exposes + connections
#   connect <from> <sig> <to> <method>     -> connect a signal to a node method
#   disconnect <from> <sig> <to> <method>  -> remove that connection
#
# connect/disconnect register with EditorUndoRedoManager (Ctrl+Z) and use
# CONNECT_PERSIST, so the wiring is saved with the scene — matching the editor's
# "Connect a Signal" dialog. The plugin injects the undo manager via
# set_undo_redo().

const ToolResponse = preload("res://addons/hera_agent_godot/core/tool_response.gd")

const MAX_SIGNALS := 500

var _undo_redo # EditorUndoRedoManager, injected by the plugin

func set_undo_redo(undo_redo) -> void:
	_undo_redo = undo_redo

func get_name() -> String:
	return "signal"

func execute(params: Dictionary) -> Dictionary:
	var root := EditorInterface.get_edited_scene_root()
	if root == null:
		return ToolResponse.failure("no scene is open in the editor")
	var action := String(params.get("action", ""))
	match action:
		"list":
			return _list(root, params)
		"connect":
			return _wire(root, params, true)
		"disconnect":
			return _wire(root, params, false)
		_:
			return ToolResponse.failure("unknown signal action: %s (want list|connect|disconnect)" % action)

func _list(root: Node, params: Dictionary) -> Dictionary:
	var path := String(params.get("node", "."))
	var node := _resolve(root, path)
	if node == null:
		return ToolResponse.failure("node not found: %s" % path)

	var signals: Array = []
	for sig in node.get_signal_list():
		var sig_name := String(sig.get("name", ""))
		if sig_name == "":
			continue
		var args: Array = []
		for a in sig.get("args", []):
			args.append(String(a.get("name", "")))
		var conns: Array = []
		var external_connection_count := 0
		for c in node.get_signal_connection_list(sig_name):
			var callable: Callable = c.get("callable")
			var target_path := _path_of(root, callable.get_object())
			if target_path == "":
				external_connection_count += 1
				continue
			conns.append({
				"to": target_path,
				"method": String(callable.get_method()),
			})
		var entry := { "name": sig_name, "args": args, "connections": conns }
		if external_connection_count > 0:
			entry["external_connections"] = external_connection_count
		signals.append(entry)

	var truncated := signals.size() > MAX_SIGNALS
	if truncated:
		signals = signals.slice(0, MAX_SIGNALS)
	return ToolResponse.success({
		"node": path,
		"count": signals.size(),
		"truncated": truncated,
		"signals": signals,
	})

func _wire(root: Node, params: Dictionary, do_connect: bool) -> Dictionary:
	var from_path := String(params.get("from", ""))
	var from_node := _resolve(root, from_path)
	if from_node == null:
		return ToolResponse.failure("from node not found: %s" % from_path)
	var sig_name := String(params.get("signal", ""))
	if not from_node.has_signal(sig_name):
		return ToolResponse.failure("node %s has no signal: %s" % [from_path, sig_name])
	var to_path := String(params.get("to", ""))
	var to_node := _resolve(root, to_path)
	if to_node == null:
		return ToolResponse.failure("to node not found: %s" % to_path)
	var method := String(params.get("method", ""))
	if method == "":
		return ToolResponse.failure("%s requires a method name" % ("connect" if do_connect else "disconnect"))

	var callable := Callable(to_node, method)
	var already := from_node.is_connected(sig_name, callable)
	var label := "%s.%s -> %s.%s" % [from_path, sig_name, to_path, method]

	if do_connect:
		if already:
			return ToolResponse.failure("already connected: %s" % label)
		if _undo_redo != null:
			_undo_redo.create_action("Hera: connect %s.%s" % [String(from_node.name), sig_name], 0, root)
			_undo_redo.add_do_method(from_node, "connect", sig_name, callable, CONNECT_PERSIST)
			_undo_redo.add_undo_method(from_node, "disconnect", sig_name, callable)
			_undo_redo.commit_action()
		else:
			from_node.connect(sig_name, callable, CONNECT_PERSIST)
		return ToolResponse.success({
			"connected": { "from": from_path, "signal": sig_name, "to": to_path, "method": method },
			"method_exists": to_node.has_method(method),
		})

	if not already:
		return ToolResponse.failure("not connected: %s" % label)
	if _undo_redo != null:
		_undo_redo.create_action("Hera: disconnect %s.%s" % [String(from_node.name), sig_name], 0, root)
		_undo_redo.add_do_method(from_node, "disconnect", sig_name, callable)
		_undo_redo.add_undo_method(from_node, "connect", sig_name, callable, CONNECT_PERSIST)
		_undo_redo.commit_action()
	else:
		from_node.disconnect(sig_name, callable)
	return ToolResponse.success({
		"disconnected": { "from": from_path, "signal": sig_name, "to": to_path, "method": method },
	})

func _resolve(root: Node, path: String) -> Node:
	return root if path == "." else root.get_node_or_null(path)

# Path of a connection target relative to the scene root; absolute for an
# out-of-scene object, empty for a non-Node (e.g. a freed or resource target).
func _path_of(root: Node, obj: Object) -> String:
	if obj is Node:
		var n := obj as Node
		if n == root or root.is_ancestor_of(n):
			return String(root.get_path_to(n))
	return ""
