extends RefCounted

const NodeValueCodec = preload("res://addons/hera_agent_godot/tools/node_value_codec.gd")
const FIELDS := ["editor_session_id", "scene", "node_instance_id", "prop", "type", "value"]
const TYPES := [TYPE_BOOL, TYPE_INT, TYPE_FLOAT, TYPE_STRING, TYPE_STRING_NAME, TYPE_NODE_PATH, TYPE_VECTOR2, TYPE_VECTOR2I, TYPE_VECTOR3, TYPE_VECTOR3I, TYPE_RECT2, TYPE_RECT2I, TYPE_COLOR]

static func validate(expected: Variant) -> String:
	if not expected is Dictionary or expected.size() != FIELDS.size():
		return "invalid expected: requires editor_session_id, scene, node_instance_id, prop, type, value"
	for field in FIELDS:
		if not expected.get(field) is String:
			return "invalid expected: %s must be a string" % field
		if expected[field] == "" and field != "scene" and field != "value":
			return "invalid expected: %s must not be empty" % field
	return ""

static func snapshot(session: String, scene: Node, node: Node, prop: String, info: Dictionary) -> Dictionary:
	var value: Variant = node.get(prop)
	var value_type := int(info.get("type", TYPE_NIL))
	if not TYPES.has(value_type) or typeof(value) != value_type:
		return {}
	var text := str(value) if value_type in [TYPE_STRING, TYPE_STRING_NAME, TYPE_NODE_PATH] else var_to_str(value)
	var parsed := NodeValueCodec.coerce(text, info)
	if not parsed.get("ok", false) or not equal(value, parsed.get("value")):
		return {}
	return {"editor_session_id": session, "scene": scene.scene_file_path, "node_instance_id": str(node.get_instance_id()), "prop": prop, "type": type_string(value_type), "value": text}

static func compare(session: String, scene: Node, node: Node, prop: String, info: Dictionary, value: Variant, expected: Dictionary) -> String:
	if session != expected["editor_session_id"]:
		return "session_mismatch: editor session changed"
	if not is_instance_valid(scene) or scene != EditorInterface.get_edited_scene_root() or scene.scene_file_path != expected["scene"]:
		return "state_conflict: edited scene changed"
	if not is_instance_valid(node) or node.is_queued_for_deletion() or str(node.get_instance_id()) != expected["node_instance_id"]:
		return "state_conflict: node instance changed"
	var value_type := int(info.get("type", TYPE_NIL))
	if prop != expected["prop"] or type_string(value_type) != expected["type"] or typeof(value) != value_type:
		return "state_conflict: property or type changed"
	if not TYPES.has(value_type):
		return "capability_unavailable: property type is not supported by node_set_guard"
	var parsed := NodeValueCodec.coerce(expected["value"], info)
	if not parsed.get("ok", false):
		return "invalid expected: property value cannot be parsed"
	if not equal(value, parsed.get("value")):
		return "state_conflict: property value changed"
	return ""

static func equal(left: Variant, right: Variant) -> bool:
	return typeof(left) == typeof(right) and left == right
