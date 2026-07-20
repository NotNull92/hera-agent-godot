extends RefCounted

# `theme` — read and write items inside a Theme resource.
#
# A Theme stores its data behind methods (set_color/set_constant/set_font_size
# on a per-type map), not as properties, so `resource set --prop` cannot reach
# it. This tool exposes that map so project-wide theme values can be inspected
# and converged without hand-writing .tres files.
#
# Writes save the resource to disk. Unlike node edits there is no
# EditorUndoRedoManager step, so a `theme set` is NOT undoable with Ctrl+Z.

const ToolResponse = preload("res://addons/hera_agent_godot/core/tool_response.gd")

func get_name() -> String:
	return "theme"

func execute(params: Dictionary) -> Dictionary:
	var action := String(params.get("action", ""))
	match action:
		"get":
			return _describe(params)
		"set":
			return _set_items(params)
		_:
			return ToolResponse.failure("unknown theme action: %s (want get|set)" % action)


# Named _describe, not _get: Object already declares a virtual
# _get(StringName) -> Variant and a clashing signature is a parse error.
func _describe(params: Dictionary) -> Dictionary:
	var path := String(params.get("path", ""))
	var loaded: Variant = _load_theme(path)
	if loaded is String:
		return ToolResponse.failure(loaded)
	var theme: Theme = loaded

	var wanted := String(params.get("type", ""))
	var types: PackedStringArray = theme.get_type_list()
	types.sort()

	var out := {}
	for type_name in types:
		if wanted != "" and String(type_name) != wanted:
			continue
		var entry := {}
		var colors := {}
		for item in theme.get_color_list(type_name):
			colors[item] = _color_text(theme.get_color(item, type_name))
		if not colors.is_empty():
			entry["colors"] = colors
		var constants := {}
		for item in theme.get_constant_list(type_name):
			constants[item] = theme.get_constant(item, type_name)
		if not constants.is_empty():
			entry["constants"] = constants
		var font_sizes := {}
		for item in theme.get_font_size_list(type_name):
			font_sizes[item] = theme.get_font_size(item, type_name)
		if not font_sizes.is_empty():
			entry["font_sizes"] = font_sizes
		if not entry.is_empty():
			out[type_name] = entry

	if wanted != "" and not out.has(wanted):
		# Distinguish "type has no items" from "type is absent" — a check
		# predicate needs to tell those apart.
		if types.has(wanted):
			out[wanted] = {}
		else:
			return ToolResponse.failure("theme type not found: %s (have %s)" % [wanted, ", ".join(types)])

	return ToolResponse.success({
		"path": path,
		"types": types,
		"items": out,
	})


# Named _set_items for the same reason as _describe above: Object declares a
# virtual _set(StringName, Variant) -> bool.
func _set_items(params: Dictionary) -> Dictionary:
	var path := String(params.get("path", ""))
	var loaded: Variant = _load_theme(path)
	if loaded is String:
		return ToolResponse.failure(loaded)
	var theme: Theme = loaded

	var type_name := String(params.get("type", ""))
	if type_name == "":
		return ToolResponse.failure("theme set requires a 'type'")

	var applied := {}

	var colors: Dictionary = params.get("colors", {})
	for item in colors.keys():
		var text := String(colors[item])
		var parsed: Variant = str_to_var(text)
		if typeof(parsed) != TYPE_COLOR:
			return ToolResponse.failure("invalid Color for %s: %s — use Color(r, g, b, a) like Color(0.3, 0.8, 1, 1)" % [item, text])
		theme.set_color(String(item), type_name, parsed)
		applied[String(item)] = _color_text(parsed)

	var constants: Dictionary = params.get("constants", {})
	for item in constants.keys():
		var text2 := String(constants[item])
		if not text2.is_valid_int():
			return ToolResponse.failure("invalid int for constant %s: %s" % [item, text2])
		theme.set_constant(String(item), type_name, int(text2))
		applied[String(item)] = int(text2)

	var font_sizes: Dictionary = params.get("font_sizes", {})
	for item in font_sizes.keys():
		var text3 := String(font_sizes[item])
		if not text3.is_valid_int():
			return ToolResponse.failure("invalid int for font size %s: %s" % [item, text3])
		theme.set_font_size(String(item), type_name, int(text3))
		applied[String(item)] = int(text3)

	if applied.is_empty():
		return ToolResponse.failure("theme set requires at least one of colors, constants or font_sizes")

	var err := ResourceSaver.save(theme, path)
	if err != OK:
		return ToolResponse.failure("save failed: %s" % error_string(err))

	return ToolResponse.success({
		"path": path,
		"type": type_name,
		"applied": applied,
		"undoable": false,
	})


# Returns a Theme, or a String describing why it could not be loaded.
func _load_theme(path: String) -> Variant:
	if not (path.begins_with("res://") or path.begins_with("user://")):
		return "path must start with res:// or user:// : %s" % path
	if not ResourceLoader.exists(path):
		return "resource not found: %s" % path
	var res := ResourceLoader.load(path)
	if res == null:
		return "failed to load resource: %s" % path
	if not (res is Theme):
		return "not a Theme resource: %s (got %s)" % [path, res.get_class()]
	return res


# Colors are float32, so the raw components print as 0.28999999165535 for a
# value written as 0.29. A `check` predicate compares what it reads against what
# it wrote, so echoing that noise would make every colour check fail. Six
# decimals is well inside float32's precision and round-trips what a caller
# actually typed.
func _color_text(c: Color) -> String:
	return "Color(%s, %s, %s, %s)" % [_num(c.r), _num(c.g), _num(c.b), _num(c.a)]


func _num(v: float) -> String:
	var text := "%.6f" % v
	text = text.rstrip("0")
	if text.ends_with("."):
		text += "0"
	return text
