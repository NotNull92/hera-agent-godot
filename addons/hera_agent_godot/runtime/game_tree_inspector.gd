extends RefCounted

static func tree(root: Node, max_nodes: int) -> Dictionary:
	var nodes: Array = []
	_collect(root, nodes, max_nodes)
	var truncated := nodes.size() > max_nodes
	if truncated:
		nodes = nodes.slice(0, max_nodes)
	return {
		"count": nodes.size(),
		"truncated": truncated,
		"nodes": nodes,
	}

static func _collect(node: Node, out: Array, max_nodes: int) -> void:
	if out.size() > max_nodes:
		return
	out.append({
		"path": String(node.get_path()),
		"type": node.get_class(),
		"name": String(node.name),
	})
	for child in node.get_children():
		_collect(child, out, max_nodes)
		if out.size() > max_nodes:
			return
