extends RefCounted

const ToolResponse = preload("res://addons/hera_agent_godot/core/tool_response.gd")


func get_name() -> String:
	return "editor"


func execute(params: Dictionary) -> Dictionary:
	var action := String(params.get("action", "state"))
	match action:
		"state":
			return _state()
		"selected":
			return _selected()
		"select":
			return _select_node(params)
		"clear_selection":
			return _clear_selection()
		_:
			return ToolResponse.failure("unknown editor action: %s (want state|selected|select|clear_selection)" % action)


func _state() -> Dictionary:
	var root := EditorInterface.get_edited_scene_root()
	return ToolResponse.success({
		"project_name": String(ProjectSettings.get_setting("application/config/name", "")),
		"project_path": ProjectSettings.globalize_path("res://"),
		"current_scene": "" if root == null else root.scene_file_path,
		"open_scenes": _to_unique_strings(EditorInterface.get_open_scenes()),
		"main_scene": String(ProjectSettings.get_setting("application/run/main_scene", "")),
		"playing": EditorInterface.is_playing_scene(),
		"playing_scene": EditorInterface.get_playing_scene(),
		"selected": _selected_nodes(root),
		"current_script": _script_summary(_current_script()),
	})


func _selected() -> Dictionary:
	var root := EditorInterface.get_edited_scene_root()
	var nodes := _selected_nodes(root)
	return ToolResponse.success({
		"count": nodes.size(),
		"nodes": nodes,
		"selected": {} if nodes.is_empty() else nodes[0],
	})


func _select_node(params: Dictionary) -> Dictionary:
	var root := EditorInterface.get_edited_scene_root()
	if root == null:
		return ToolResponse.failure("no scene is open in the editor")
	var path := String(params.get("path", "."))
	var node := _resolve(root, path)
	if node == null:
		return ToolResponse.failure("node not found: %s" % path)
	var selection: EditorSelection = EditorInterface.get_selection()
	if selection == null:
		return ToolResponse.failure("editor selection is not available")
	if not bool(params.get("add", false)):
		selection.clear()
	selection.add_node(node)
	if EditorInterface.has_method("inspect_object"):
		EditorInterface.call("inspect_object", node)
	return _selected()


func _clear_selection() -> Dictionary:
	var selection: EditorSelection = EditorInterface.get_selection()
	if selection == null:
		return ToolResponse.failure("editor selection is not available")
	selection.clear()
	return _selected()


func _selected_nodes(root: Node) -> Array:
	var out := []
	var selection: EditorSelection = EditorInterface.get_selection()
	if selection == null:
		return out
	for node in selection.get_selected_nodes():
		if node is Node:
			out.append(_node_summary(root, node))
	return out


func _resolve(root: Node, path: String) -> Node:
	if path == "" or path == ".":
		return root
	return root.get_node_or_null(path)


func _node_summary(root: Node, node: Node) -> Dictionary:
	var path := String(node.get_path())
	if root != null and (node == root or root.is_ancestor_of(node)):
		path = String(root.get_path_to(node))
	return {
		"path": path,
		"type": node.get_class(),
		"name": String(node.name),
	}


func _current_script() -> Script:
	if not EditorInterface.has_method("get_script_editor"):
		return null
	var script_editor: Object = EditorInterface.call("get_script_editor")
	if script_editor == null or not script_editor.has_method("get_current_script"):
		return null
	var raw: Variant = script_editor.call("get_current_script")
	if raw is Script:
		return raw as Script
	return null


func _script_summary(script: Script) -> Dictionary:
	if script == null:
		return { "found": false, "path": "" }
	return {
		"found": true,
		"path": script.resource_path,
		"base_type": script.get_instance_base_type(),
	}


func _to_unique_strings(values: Variant) -> Array:
	var result := []
	var seen := {}
	for value in values:
		var text := String(value)
		if seen.has(text):
			continue
		seen[text] = true
		result.append(text)
	return result
