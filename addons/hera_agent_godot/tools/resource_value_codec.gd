extends RefCounted

const ValueCodec = preload("res://addons/hera_agent_godot/core/value_codec.gd")

const ProjectPathSafety = preload("res://addons/hera_agent_godot/tools/project_path_safety.gd")

static func apply_props(res: Resource, raw_props: Variant) -> Dictionary:
	if typeof(raw_props) != TYPE_DICTIONARY:
		return { "ok": false, "error": "props must be an object" }
	var set_props := {}
	var values: Dictionary = {}
	var props: Dictionary = raw_props
	for raw_name in props.keys():
		var prop_name := String(raw_name)
		var prop_info := _property_info(res, prop_name)
		if prop_info.is_empty():
			return { "ok": false, "error": "resource has no property: %s" % prop_name }
		var coerced := _coerce(props[raw_name], prop_info)
		if not bool(coerced.get("ok", false)):
			return { "ok": false, "error": String(coerced.get("error", "invalid property value")) }
		values[prop_name] = coerced.get("value")
	for prop_name in values:
		res.set(prop_name, values[prop_name])
		set_props[prop_name] = str(res.get(prop_name))
	return { "ok": true, "properties": set_props }

static func _property_info(res: Resource, prop: String) -> Dictionary:
	for p in res.get_property_list():
		if String(p.get("name", "")) == prop:
			return p
	return {}

static func _coerce(raw: Variant, prop_info: Dictionary) -> Dictionary:
	return ValueCodec.coerce(raw, prop_info, {"numeric": true, "parse_node_path": true, "load_resource": _coerce_resource_object})

static func _coerce_resource_object(text: String, prop_info: Dictionary) -> Dictionary:
	if text == "null":
		return { "ok": true, "value": null }
	if not (text.begins_with("res://") or text.begins_with("user://")):
		return { "ok": false, "error": "object property expects a resource path or null: %s" % String(prop_info.get("name", "")) }
	if text.begins_with("res://") and not ProjectPathSafety.is_safe_res_path(text):
		return { "ok": false, "error": "resource path must stay inside res://" }
	if not ResourceLoader.exists(text):
		return { "ok": false, "error": "resource not found: %s" % text }
	var loaded := ResourceLoader.load(text)
	if loaded == null or not (loaded is Resource):
		return { "ok": false, "error": "not a resource: %s" % text }
	var expected := String(prop_info.get("class_name", ""))
	if expected != "" and not loaded.is_class(expected):
		return { "ok": false, "error": "resource type %s is not compatible with property %s (%s)" % [loaded.get_class(), String(prop_info.get("name", "")), expected] }
	return { "ok": true, "value": loaded }
