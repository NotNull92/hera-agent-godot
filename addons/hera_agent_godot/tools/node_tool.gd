extends RefCounted

# `node` — read and mutate nodes in the edited scene.
#   find    -> match by name substring (query) and/or class (type)
#   get     -> dump a node's editor-visible properties (values stringified)
#   add     -> instantiate a class and add it under a parent
#   set     -> set a property on a node
#   set_resource -> set a Resource property on a node
#   remove  -> remove a node
#
# All mutations are registered with EditorUndoRedoManager so the developer can
# undo agent changes (Ctrl+Z). The plugin injects it via set_undo_redo().

const ToolResponse = preload("res://addons/hera_agent_godot/core/tool_response.gd")
const NodeValueCodec = preload("res://addons/hera_agent_godot/tools/node_value_codec.gd")
const NodeSceneInstancer = preload("res://addons/hera_agent_godot/tools/node_scene_instancer.gd")
const ScriptDependencyDiagnostics = preload("res://addons/hera_agent_godot/tools/script_dependency_diagnostics.gd")
const ProjectPathSafety = preload("res://addons/hera_agent_godot/tools/project_path_safety.gd")

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
		"instance":
			return NodeSceneInstancer.execute(root, params, _undo_redo)
		"set":
			return _set_property(root, params)
		"set_resource":
			return _set_resource_property(root, params)
		"remove":
			return _remove_node(root, params)
		"attach_script":
			return _attach_script(root, params)
		"detach_script":
			return _detach_script(root, params)
		_:
			return ToolResponse.failure("unknown node action: %s (want find|get|add|instance|set|set_resource|remove|attach_script|detach_script)" % action)

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
		"properties": NodeValueCodec.properties(node, MAX_VALUE_LEN),
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
	var coerced := NodeValueCodec.coerce(params.get("value"), prop_info)
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

func _set_resource_property(root: Node, params: Dictionary) -> Dictionary:
	var path := String(params.get("path", "."))
	var node := _resolve(root, path)
	if node == null:
		return ToolResponse.failure("node not found: %s" % path)
	var prop := String(params.get("prop", ""))
	var prop_info := _property_info(node, prop)
	if prop == "" or prop_info.is_empty():
		return ToolResponse.failure("node has no property: %s" % prop)
	if int(prop_info.get("type", TYPE_NIL)) != TYPE_OBJECT:
		return ToolResponse.failure("property is not an object/resource property: %s" % prop)
	var resource_path := String(params.get("resource", ""))
	if not (resource_path.begins_with("res://") or resource_path.begins_with("user://")):
		return ToolResponse.failure("resource path must start with res:// or user://")
	if resource_path.begins_with("res://") and not ProjectPathSafety.is_safe_res_path(resource_path):
		return ToolResponse.failure("resource path must stay inside res://")
	if not ResourceLoader.exists(resource_path):
		return ToolResponse.failure("resource not found: %s" % resource_path)
	var loaded := ResourceLoader.load(resource_path)
	if loaded == null or not (loaded is Resource):
		return ToolResponse.failure("not a resource: %s" % resource_path)
	var expected := String(prop_info.get("class_name", ""))
	if expected != "" and not loaded.is_class(expected):
		return ToolResponse.failure("resource type %s is not compatible with property %s (%s)" % [loaded.get_class(), prop, expected])
	var old_value: Variant = node.get(prop)
	if _undo_redo != null:
		_undo_redo.create_action("Hera: set resource %s.%s" % [String(node.name), prop])
		_undo_redo.add_do_property(node, prop, loaded)
		_undo_redo.add_undo_property(node, prop, old_value)
		_undo_redo.add_do_reference(loaded)
		_undo_redo.commit_action()
	else:
		node.set(prop, loaded)
	return ToolResponse.success({ "path": path, "prop": prop, "resource": resource_path, "value": str(node.get(prop)) })

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
	if not script_path.begins_with("res://") or not ProjectPathSafety.is_safe_res_path(script_path):
		return ToolResponse.failure("script path must be a safe res:// path")
	if not (script_path.ends_with(".gd") or script_path.ends_with(".cs")):
		return ToolResponse.failure("script path must end with .gd or .cs")
	if not FileAccess.file_exists(script_path):
		return ToolResponse.failure("script not found: %s" % script_path)
	_scan_editor_filesystem()
	var diagnostics := ScriptDependencyDiagnostics.analyze(script_path)
	var missing_preloads: Array = diagnostics.get("missing_preloads", [])
	if not missing_preloads.is_empty():
		return ToolResponse.failure("script preload dependency missing: %s" % str(missing_preloads))
	var loaded_res := ResourceLoader.load(script_path)
	if loaded_res == null or not (loaded_res is Script):
		return ToolResponse.failure("not a script resource: %s" % script_path)
	var loaded: Script = loaded_res as Script
	var base_type := loaded.get_instance_base_type()
	if base_type != "" and not node.is_class(base_type):
		return ToolResponse.failure("script base type %s is not compatible with node type %s" % [base_type, node.get_class()])

	var old_script: Variant = node.get_script()
	if _undo_redo != null:
		_undo_redo.create_action("Hera: attach script to %s" % String(node.name))
		_undo_redo.add_do_method(node, "set_script", loaded)
		_undo_redo.add_undo_method(node, "set_script", old_script)
		_undo_redo.add_do_reference(loaded)
		_undo_redo.commit_action()
	else:
		node.set_script(loaded)

	return ToolResponse.success({
		"path": path,
		"script": script_path,
		"base_type": base_type,
		"script_diagnostics": diagnostics,
	})

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

func _scan_editor_filesystem() -> void:
	var filesystem := EditorInterface.get_resource_filesystem()
	if filesystem != null:
		filesystem.scan()
