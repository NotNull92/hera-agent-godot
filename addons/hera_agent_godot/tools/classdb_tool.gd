extends RefCounted

const ToolResponse = preload("res://addons/hera_agent_godot/core/tool_response.gd")

const MAX_ITEMS := 500

func get_name() -> String:
	return "classdb"

func execute(params: Dictionary) -> Dictionary:
	var action := String(params.get("action", ""))
	match action:
		"info":
			return _info(params)
		"methods":
			return _methods(params)
		"properties":
			return _properties(params)
		"signals":
			return _signals(params)
		"constants":
			return _constants(params)
		"enums":
			return _enums(params)
		"inherits":
			return _inherits(params)
		_:
			return ToolResponse.failure("unknown classdb action: %s (want info|methods|properties|signals|constants|enums|inherits)" % action)

func _info(params: Dictionary) -> Dictionary:
	var klass := String(params.get("class", ""))
	if not ClassDB.class_exists(klass):
		return ToolResponse.failure("class not found: %s" % klass)
	var parent := ClassDB.get_parent_class(klass)
	return ToolResponse.success({
		"class": klass,
		"parent": parent,
		"can_instantiate": ClassDB.can_instantiate(klass),
		"is_node": klass == "Node" or ClassDB.is_parent_class(klass, "Node"),
		"is_resource": klass == "Resource" or ClassDB.is_parent_class(klass, "Resource"),
	})

func _methods(params: Dictionary) -> Dictionary:
	var klass := String(params.get("class", ""))
	if not ClassDB.class_exists(klass):
		return ToolResponse.failure("class not found: %s" % klass)
	var out := []
	var methods := ClassDB.class_get_method_list(klass, true)
	for method in methods:
		if out.size() >= MAX_ITEMS:
			break
		out.append(_method_summary(method))
	return ToolResponse.success({ "class": klass, "count": out.size(), "truncated": methods.size() > out.size(), "methods": out })

func _properties(params: Dictionary) -> Dictionary:
	var klass := String(params.get("class", ""))
	if not ClassDB.class_exists(klass):
		return ToolResponse.failure("class not found: %s" % klass)
	var out := []
	var props := ClassDB.class_get_property_list(klass, true)
	for prop in props:
		if out.size() >= MAX_ITEMS:
			break
		out.append({
			"name": String(prop.get("name", "")),
			"type": type_string(int(prop.get("type", TYPE_NIL))),
			"class_name": String(prop.get("class_name", "")),
			"hint": int(prop.get("hint", 0)),
			"hint_string": String(prop.get("hint_string", "")),
		})
	return ToolResponse.success({ "class": klass, "count": out.size(), "truncated": props.size() > out.size(), "properties": out })

func _signals(params: Dictionary) -> Dictionary:
	var klass := String(params.get("class", ""))
	if not ClassDB.class_exists(klass):
		return ToolResponse.failure("class not found: %s" % klass)
	var out := []
	var signals := ClassDB.class_get_signal_list(klass, bool(params.get("own", false)))
	for sig in signals:
		if out.size() >= MAX_ITEMS:
			break
		out.append(_signal_summary(sig))
	return ToolResponse.success({ "class": klass, "count": out.size(), "truncated": signals.size() > out.size(), "signals": out })

func _constants(params: Dictionary) -> Dictionary:
	var klass := String(params.get("class", ""))
	if not ClassDB.class_exists(klass):
		return ToolResponse.failure("class not found: %s" % klass)
	var out := []
	var own := bool(params.get("own", false))
	var constants := ClassDB.class_get_integer_constant_list(klass, own)
	for name in constants:
		if out.size() >= MAX_ITEMS:
			break
		var const_name := String(name)
		out.append({
			"name": const_name,
			"value": ClassDB.class_get_integer_constant(klass, const_name),
			"enum": String(ClassDB.class_get_integer_constant_enum(klass, const_name, own)),
		})
	return ToolResponse.success({ "class": klass, "count": out.size(), "truncated": constants.size() > out.size(), "constants": out })

func _enums(params: Dictionary) -> Dictionary:
	var klass := String(params.get("class", ""))
	if not ClassDB.class_exists(klass):
		return ToolResponse.failure("class not found: %s" % klass)
	var out := []
	var own := bool(params.get("own", false))
	var enums := ClassDB.class_get_enum_list(klass, own)
	for enum_name in enums:
		if out.size() >= MAX_ITEMS:
			break
		var enum_text := String(enum_name)
		out.append({
			"name": enum_text,
			"constants": _enum_constants(klass, enum_text, own),
		})
	return ToolResponse.success({ "class": klass, "count": out.size(), "truncated": enums.size() > out.size(), "enums": out })

func _inherits(params: Dictionary) -> Dictionary:
	var klass := String(params.get("class", ""))
	var base := String(params.get("base", ""))
	if not ClassDB.class_exists(klass):
		return ToolResponse.failure("class not found: %s" % klass)
	if not ClassDB.class_exists(base):
		return ToolResponse.failure("base class not found: %s" % base)
	return ToolResponse.success({
		"class": klass,
		"base": base,
		"inherits": klass == base or ClassDB.is_parent_class(klass, base),
	})

func _method_summary(method: Dictionary) -> Dictionary:
	var args := []
	for arg in Array(method.get("args", [])):
		args.append({
			"name": String(arg.get("name", "")),
			"type": type_string(int(arg.get("type", TYPE_NIL))),
			"class_name": String(arg.get("class_name", "")),
		})
	var ret: Dictionary = method.get("return", {})
	return {
		"name": String(method.get("name", "")),
		"return_type": type_string(int(ret.get("type", TYPE_NIL))),
		"return_class": String(ret.get("class_name", "")),
		"args": args,
	}

func _signal_summary(sig: Dictionary) -> Dictionary:
	var args := []
	for arg in Array(sig.get("args", [])):
		args.append({
			"name": String(arg.get("name", "")),
			"type": type_string(int(arg.get("type", TYPE_NIL))),
			"class_name": String(arg.get("class_name", "")),
		})
	return {
		"name": String(sig.get("name", "")),
		"args": args,
	}

func _enum_constants(klass: String, enum_name: String, own: bool) -> Array:
	var out := []
	for name in ClassDB.class_get_enum_constants(klass, enum_name, own):
		var const_name := String(name)
		out.append({
			"name": const_name,
			"value": ClassDB.class_get_integer_constant(klass, const_name),
		})
	return out
