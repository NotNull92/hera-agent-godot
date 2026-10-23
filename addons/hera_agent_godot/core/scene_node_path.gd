extends RefCounted

static func resolve(root: Node, path: String) -> Node:
	if root == null:
		return null
	var node: Node = root if path == "" or path == "." else root.get_node_or_null(path)
	return node if node == root or (node != null and root.is_ancestor_of(node)) else null
