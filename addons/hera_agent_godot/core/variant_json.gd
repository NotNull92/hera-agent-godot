extends RefCounted

const MAX_DEPTH := 32
const MAX_ITEMS := 10000
const MAX_BYTES := 65536
const MAX_SAFE_INT := 9007199254740991


static func encode(value: Variant) -> Dictionary:
	var budget: Dictionary = {"items": 0, "bytes": 0, "reason": ""}
	var encoded: Variant = _encode(value, 0, budget)
	if budget.reason != "":
		return {"ok": false, "reason": budget.reason}
	return {"ok": true, "value": encoded}


static func attach_result(payload: Dictionary, value: Variant) -> Dictionary:
	var encoded: Dictionary = encode(value)
	payload.erase("result_value")
	payload.erase("result_value_omitted")
	if encoded.ok:
		payload["result_value"] = encoded.value
	else:
		payload["result_value_omitted"] = encoded.reason
	return payload


static func _encode(value: Variant, depth: int, budget: Dictionary) -> Variant:
	if depth > MAX_DEPTH:
		budget.reason = "depth_limit"
		return null
	var kind: int = typeof(value)
	if kind == TYPE_ARRAY or kind == TYPE_DICTIONARY or kind >= TYPE_PACKED_BYTE_ARRAY:
		var count: int = value.size()
		budget.items += count
		if budget.items > MAX_ITEMS:
			budget.reason = "item_limit"
			return null
		_add_bytes(budget, 2 + maxi(0, count - 1))
		if kind == TYPE_DICTIONARY:
			var object: Dictionary = {}
			for key in value:
				# Check container keys before var_to_str can recurse through them.
				if typeof(key) == TYPE_ARRAY or typeof(key) == TYPE_DICTIONARY:
					var previous_bytes: int = budget.bytes
					_encode(key, depth + 1, budget)
					budget.bytes = previous_bytes
					if budget.reason != "":
						return null
				var text: String = str(key) if typeof(key) == TYPE_STRING or typeof(key) == TYPE_STRING_NAME else var_to_str(key)
				if object.has(text):
					budget.reason = "key_collision"
					return null
				_add_bytes(budget, JSON.stringify(text).to_utf8_buffer().size() + 1)
				if budget.reason != "":
					return null
				object[text] = _encode(value[key], depth + 1, budget)
				if budget.reason != "":
					return null
			return object
		var array: Array = []
		for element in value:
			array.append(_encode(element, depth + 1, budget))
			if budget.reason != "":
				return null
		return array
	var leaf: Variant = value
	match kind:
		TYPE_INT:
			if value > MAX_SAFE_INT or value < -MAX_SAFE_INT:
				leaf = str(value)
		TYPE_FLOAT:
			if not is_finite(value):
				leaf = var_to_str(value)
		TYPE_STRING_NAME, TYPE_NODE_PATH:
			leaf = str(value)
		TYPE_OBJECT:
			leaf = str(value) if is_instance_valid(value) else null
		TYPE_NIL, TYPE_BOOL, TYPE_STRING:
			pass
		_:
			leaf = var_to_str(value)
	if typeof(leaf) == TYPE_STRING and leaf.length() > MAX_BYTES:
		budget.reason = "size_limit"
		return null
	_add_bytes(budget, JSON.stringify(leaf).to_utf8_buffer().size())
	return leaf


static func _add_bytes(budget: Dictionary, count: int) -> void:
	budget.bytes += count
	if budget.bytes > MAX_BYTES:
		budget.reason = "size_limit"
