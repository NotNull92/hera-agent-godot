extends RefCounted

var _tools: Dictionary = {}

func register(tool: RefCounted) -> void:
	_tools[tool.get_name()] = tool

func resolve(name: String) -> Variant:
	return _tools.get(name)

func names() -> PackedStringArray:
	var result := PackedStringArray()
	for name in _tools.keys():
		result.append(str(name))
	return result
