extends RefCounted

# `resource` — inspect a Resource file (.tres / .res / .tscn / any res://) without
# opening it as a scene.
#   get <res://path>  -> the resource's class, name, and editor-visible properties
#
# Read-only: loads via ResourceLoader and dumps stringified property values
# (capped), mirroring `node get` for standalone resources. No scene needs to be
# open.

const ToolResponse = preload("res://addons/hera_agent_godot/core/tool_response.gd")

const MAX_VALUE_LEN := 200

func get_name() -> String:
	return "resource"

func execute(params: Dictionary) -> Dictionary:
	var action := String(params.get("action", ""))
	match action:
		"get":
			return _describe(params)
		_:
			return ToolResponse.failure("unknown resource action: %s (want get)" % action)

# Named _describe, not _get: _get(StringName) is an Object virtual and a custom
# _get() would clash with it (parse error). Same reason node_tool uses _describe.
func _describe(params: Dictionary) -> Dictionary:
	var path := String(params.get("path", ""))
	if not (path.begins_with("res://") or path.begins_with("user://")):
		return ToolResponse.failure("path must start with res:// or user:// : %s" % path)
	if not ResourceLoader.exists(path):
		return ToolResponse.failure("resource not found: %s" % path)
	var res := ResourceLoader.load(path)
	if res == null:
		return ToolResponse.failure("failed to load resource: %s" % path)

	return ToolResponse.success({
		"path": path,
		"type": res.get_class(),
		"resource_name": res.resource_name,
		"properties": _properties(res),
	})

func _properties(res: Resource) -> Dictionary:
	var result := {}
	for prop in res.get_property_list():
		var usage := int(prop.get("usage", 0))
		if not (usage & PROPERTY_USAGE_EDITOR):
			continue
		if usage & (PROPERTY_USAGE_CATEGORY | PROPERTY_USAGE_GROUP | PROPERTY_USAGE_SUBGROUP):
			continue
		var pname := String(prop.get("name", ""))
		if pname == "":
			continue
		var text := str(res.get(pname))
		if text.length() > MAX_VALUE_LEN:
			text = text.substr(0, MAX_VALUE_LEN) + "…"
		result[pname] = text
	return result
