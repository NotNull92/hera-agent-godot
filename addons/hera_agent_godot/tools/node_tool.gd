extends RefCounted

# `node` — read and mutate nodes in the edited scene.
#   find    -> match by name substring (query) and/or class (type)
#   get     -> dump a node's editor-visible properties (values stringified)
#   add     -> instantiate a class and add it under a parent
#   set     -> set a property on a node
#   remove  -> remove a node
#
# All mutations are registered with EditorUndoRedoManager so the developer can
# undo agent changes (Ctrl+Z). The plugin injects it via set_undo_redo().

const ToolResponse = preload("res://addons/hera_agent_godot/core/tool_response.gd")

const MAX_NODES := 1000
const MAX_VALUE_LEN := 200

var _undo_redo # EditorUndoRedoManager, injected by the plugin

func set_undo_redo(undo_redo) -> void:
	_undo_redo = undo_redo

func get_name() -> String:
	return "node"

func execute(params: Dictionary) -> Dictionary:
	var root := EditorInterface.get_edited_scene_root()
	if root == null:
		return ToolResponse.failure("no scene is open in the editor")
	var action := String(params.get("action", ""))
	match action:
		"find":
			return _find(root, params)
		"get":
			return _describe(root, params)
		"add":
			return _add_node(root, params)
		"set":
			return _set_property(root, params)
		"remove":
			return _remove_node(root, params)
		"attach_script":
			return _attach_script(root, params)
		"detach_script":
			return _detach_script(root, params)
		_:
			return ToolResponse.failure("unknown node action: %s (want find|get|add|set|remove|attach_script|detach_script)" % action)

func _find(root: Node, params: Dictionary) -> Dictionary:
	var query := String(params.get("query", "")).to_lower()
	var type_filter := String(params.get("type", ""))
	var matches: Array = []
	_walk(root, root, query, type_filter, matches)
	var truncated := matches.size() > MAX_NODES
	if truncated:
		matches = matches.slice(0, MAX_NODES)
	return ToolResponse.success({ "count": matches.size(), "truncated": truncated, "nodes": matches })

func _walk(node: Node, root: Node, query: String, type_filter: String, out: Array) -> void:
	if out.size() > MAX_NODES:
		return
	var name_ok := query == "" or String(node.name).to_lower().find(query) != -1
	var type_ok := type_filter == "" or node.is_class(type_filter)
	if name_ok and type_ok:
		out.append({
			"path": String(root.get_path_to(node)),
			"type": node.get_class(),
			"name": String(node.name),
		})
	for child in node.get_children():
		_walk(child, root, query, type_filter, out)
		if out.size() > MAX_NODES:
			return

func _describe(root: Node, params: Dictionary) -> Dictionary:
	var node := _resolve(root, String(params.get("path", ".")))
	if node == null:
		return ToolResponse.failure("node not found: %s" % String(params.get("path", ".")))
	return ToolResponse.success({
		"path": String(params.get("path", ".")),
		"type": node.get_class(),
		"name": String(node.name),
		"properties": _properties(node),
	})

func _add_node(root: Node, params: Dictionary) -> Dictionary:
	var type := String(params.get("type", ""))
	if type == "":
		return ToolResponse.failure("add requires a 'type' (node class)")
	if not ClassDB.can_instantiate(type) or not ClassDB.is_parent_class(type, "Node"):
		return ToolResponse.failure("not an instantiable Node class: %s" % type)
	var parent_path := String(params.get("parent", "."))
	var parent := _resolve(root, parent_path)
	if parent == null:
		return ToolResponse.failure("parent not found: %s" % parent_path)

	var node: Node = ClassDB.instantiate(type)
	node.name = String(params.get("name", type))

	if _undo_redo != null:
		_undo_redo.create_action("Hera: add %s" % type)
		_undo_redo.add_do_method(parent, "add_child", node)
		_undo_redo.add_do_method(node, "set_owner", root)
		_undo_redo.add_do_reference(node)
		_undo_redo.add_undo_method(parent, "remove_child", node)
		_undo_redo.commit_action()
	else:
		parent.add_child(node)
		node.set_owner(root)

	return ToolResponse.success({
		"added": String(root.get_path_to(node)),
		"type": node.get_class(),
		"name": String(node.name),
	})

func _set_property(root: Node, params: Dictionary) -> Dictionary:
	var path := String(params.get("path", "."))
	var node := _resolve(root, path)
	if node == null:
		return ToolResponse.failure("node not found: %s" % path)
	var prop := String(params.get("prop", ""))
	var prop_info := _property_info(node, prop)
	if prop == "" or prop_info.is_empty():
		return ToolResponse.failure("node has no property: %s" % prop)

	var old_value: Variant = node.get(prop)
	var coerced := _coerce(params.get("value"), prop_info)
	if not bool(coerced.get("ok", false)):
		return ToolResponse.failure(String(coerced.get("error", "invalid property value")))
	var new_value: Variant = coerced.get("value")

	if _undo_redo != null:
		_undo_redo.create_action("Hera: set %s.%s" % [String(node.name), prop])
		_undo_redo.add_do_property(node, prop, new_value)
		_undo_redo.add_undo_property(node, prop, old_value)
		_undo_redo.commit_action()
	else:
		node.set(prop, new_value)

	return ToolResponse.success({ "path": path, "prop": prop, "value": str(node.get(prop)) })

func _remove_node(root: Node, params: Dictionary) -> Dictionary:
	var path := String(params.get("path", ""))
	if path == "" or path == ".":
		return ToolResponse.failure("cannot remove the scene root")
	var node := root.get_node_or_null(path)
	if node == null:
		return ToolResponse.failure("node not found: %s" % path)
	var parent := node.get_parent()
	if parent == null:
		return ToolResponse.failure("node has no parent: %s" % path)
	var index := node.get_index()
	var old_owner := node.owner

	if _undo_redo != null:
		_undo_redo.create_action("Hera: remove %s" % String(node.name))
		_undo_redo.add_do_method(parent, "remove_child", node)
		_undo_redo.add_undo_method(parent, "add_child", node)
		_undo_redo.add_undo_method(parent, "move_child", node, index)
		_undo_redo.add_undo_method(node, "set_owner", old_owner)
		_undo_redo.add_undo_reference(node)
		_undo_redo.commit_action()
	else:
		parent.remove_child(node)
		node.queue_free()

	return ToolResponse.success({ "removed": path })

func _attach_script(root: Node, params: Dictionary) -> Dictionary:
	var path := String(params.get("path", ""))
	var node := _resolve(root, path)
	if node == null:
		return ToolResponse.failure("node not found: %s" % path)
	var script_path := String(params.get("script", ""))
	if script_path == "":
		return ToolResponse.failure("attach-script requires a script path")
	var loaded := load(script_path)
	if loaded == null or not (loaded is Script):
		return ToolResponse.failure("not a script resource: %s" % script_path)

	var old_script: Variant = node.get_script()
	if _undo_redo != null:
		_undo_redo.create_action("Hera: attach script to %s" % String(node.name))
		_undo_redo.add_do_method(node, "set_script", loaded)
		_undo_redo.add_undo_method(node, "set_script", old_script)
		_undo_redo.add_do_reference(loaded)
		_undo_redo.commit_action()
	else:
		node.set_script(loaded)

	return ToolResponse.success({ "path": path, "script": script_path })

func _detach_script(root: Node, params: Dictionary) -> Dictionary:
	var path := String(params.get("path", ""))
	var node := _resolve(root, path)
	if node == null:
		return ToolResponse.failure("node not found: %s" % path)
	var old_script: Variant = node.get_script()

	if _undo_redo != null:
		_undo_redo.create_action("Hera: detach script from %s" % String(node.name))
		_undo_redo.add_do_method(node, "set_script", null)
		_undo_redo.add_undo_method(node, "set_script", old_script)
		if old_script != null:
			_undo_redo.add_undo_reference(old_script)
		_undo_redo.commit_action()
	else:
		node.set_script(null)

	return ToolResponse.success({ "path": path, "script": "" })

func _resolve(root: Node, path: String) -> Node:
	return root if path == "." else root.get_node_or_null(path)

func _property_info(node: Node, prop: String) -> Dictionary:
	for p in node.get_property_list():
		if String(p.get("name", "")) == prop:
			return p
	return {}

func _coerce(raw: Variant, prop_info: Dictionary) -> Dictionary:
	var target_type := int(prop_info.get("type", TYPE_NIL))
	if typeof(raw) != TYPE_STRING:
		return { "ok": true, "value": raw }
	var s := String(raw)
	match target_type:
		TYPE_STRING:
			return { "ok": true, "value": s }
		TYPE_STRING_NAME:
			return { "ok": true, "value": StringName(s) }
		TYPE_BOOL:
			var lower := s.to_lower()
			if lower == "true" or lower == "1":
				return { "ok": true, "value": true }
			if lower == "false" or lower == "0":
				return { "ok": true, "value": false }
			return { "ok": false, "error": "invalid bool value for property: %s" % s }
		TYPE_INT:
			if not s.is_valid_int():
				return { "ok": false, "error": "invalid int value for property: %s" % s }
			return { "ok": true, "value": int(s) }
		TYPE_FLOAT:
			if not s.is_valid_float():
				return { "ok": false, "error": "invalid float value for property: %s" % s }
			return { "ok": true, "value": float(s) }
		TYPE_NODE_PATH:
			return { "ok": true, "value": NodePath(s) }
		TYPE_NIL:
			if s == "null":
				return { "ok": true, "value": null }
			return { "ok": false, "error": "cannot infer type for null property: %s" % String(prop_info.get("name", "")) }
		TYPE_OBJECT:
			return { "ok": false, "error": "unsupported object/resource property: %s" % String(prop_info.get("name", "")) }
		_:
			var parsed: Variant = str_to_var(s) # e.g. "Vector2(1, 2)", "[1, 2]"
			if parsed == null and s != "null":
				return { "ok": false, "error": "invalid %s value for property: %s" % [type_string(target_type), s] }
			if typeof(parsed) != target_type:
				return { "ok": false, "error": "property expects %s, got %s" % [type_string(target_type), type_string(typeof(parsed))] }
			return { "ok": true, "value": parsed }

func _properties(node: Node) -> Dictionary:
	var result := {}
	for prop in node.get_property_list():
		var usage := int(prop.get("usage", 0))
		if not (usage & PROPERTY_USAGE_EDITOR):
			continue
		if usage & (PROPERTY_USAGE_CATEGORY | PROPERTY_USAGE_GROUP | PROPERTY_USAGE_SUBGROUP):
			continue
		var pname := String(prop.get("name", ""))
		if pname == "":
			continue
		var text := str(node.get(pname))
		if text.length() > MAX_VALUE_LEN:
			text = text.substr(0, MAX_VALUE_LEN) + "…"
		result[pname] = text
	return result
