extends RefCounted

# `node` — read nodes in the edited scene.
#   find  -> match by name substring (query) and/or class (type)
#   get   -> dump a node's editor-visible properties (values stringified)

const ToolResponse = preload("res://addons/hera_agent_godot/core/tool_response.gd")

const MAX_NODES := 1000
const MAX_VALUE_LEN := 200

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
		_:
			return ToolResponse.failure("unknown node action: %s (want find|get)" % action)

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
	var path := String(params.get("path", "."))
	var node: Node = root if path == "." else root.get_node_or_null(path)
	if node == null:
		return ToolResponse.failure("node not found: %s" % path)
	return ToolResponse.success({
		"path": path,
		"type": node.get_class(),
		"name": String(node.name),
		"properties": _properties(node),
	})

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
