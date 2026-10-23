extends SceneTree

const NodeCodec = preload("res://addons/hera_agent_godot/tools/node_value_codec.gd")
const ResourceCodec = preload("res://addons/hera_agent_godot/tools/resource_value_codec.gd")
const GameCodec = preload("res://addons/hera_agent_godot/runtime/game_value_codec.gd")

func _initialize() -> void:
	var info := {"name": "target", "type": TYPE_NODE_PATH}
	assert(NodeCodec.coerce("Child", info).value == NodePath("Child"))
	assert(GameCodec.coerce("Child", info).value == NodePath("Child"))
	assert(ResourceCodec._coerce('NodePath("Child")', info).value == NodePath("Child"))
	info.type = TYPE_INT
	assert(typeof(NodeCodec.coerce(2.0, info).value) == TYPE_FLOAT)
	assert(typeof(ResourceCodec._coerce(2.0, info).value) == TYPE_INT)
	assert(not ResourceCodec._coerce(2.5, info).ok)
	info.type = TYPE_OBJECT
	assert(NodeCodec.coerce("null", info).error == "object/resource property: target — set it with `node set-resource <path> --prop target --resource res://...`")
	assert(GameCodec.coerce("null", info).error == "unsupported object/resource property: target")
	assert(ResourceCodec._coerce("null", info).value == null)
	var node := Node2D.new()
	node.name = "LongName"
	assert(NodeCodec.selected_properties(node, ["name"], 4).properties.name == "Long…")
	assert(GameCodec.property_value(node, "name").type == "StringName")
	assert(GameCodec.selected_properties(node, ["position.x"]).properties["position.x"] == "0.0")
	assert(NodeCodec.property_value(node, "", 200).error == "property name is empty")
	assert(GameCodec.property_value(node, "").error == "node has no property: ")
	node.free()
	quit()
