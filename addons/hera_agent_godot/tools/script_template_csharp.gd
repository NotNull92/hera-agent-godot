extends RefCounted

const KEYWORDS := "abstract as base bool break byte case catch char checked class const continue decimal default delegate do double else enum event explicit extern false finally fixed float for foreach goto if implicit in int interface internal is lock long namespace new null object operator out override params private protected public readonly ref return sbyte sealed short sizeof stackalloc static string struct switch this throw true try typeof uint ulong unchecked unsafe ushort using virtual void volatile while"
const PRIMITIVES := ["bool", "byte", "sbyte", "short", "ushort", "int", "uint", "long", "ulong", "float", "double", "decimal", "char", "string", "object"]
const STUBS := {"ready": "_Ready()", "process": "_Process(double delta)", "physics_process": "_PhysicsProcess(double delta)", "input": "_Input(InputEvent @event)", "unhandled_input": "_UnhandledInput(InputEvent @event)"}


func build(params: Dictionary) -> Dictionary:
	var path := String(params.get("path", ""))
	var script_class := path.get_file().get_basename()
	if path.get_extension() != "cs" or not _identifier(script_class):
		return _error("C# path must end in .cs with a valid C# identifier as its filename")
	var explicit_class := String(params.get("class_name", ""))
	if explicit_class != "" and explicit_class != script_class:
		return _error("C# class_name must match filename: %s" % script_class)
	var base_class := String(params.get("extends", "Node"))
	if not _qualified_identifier(base_class):
		return _error("extends must be a valid C# class name")
	var members: Array[String] = [script_class, "SignalName", "PropertyName", "MethodName"]
	for raw_key in STUBS:
		var key := String(raw_key)
		if bool(params.get(key, false)):
			members.append(String(STUBS[key]).get_slice("(", 0))
	var signal_names: Array[String] = []
	var declarations: Array[String] = []
	for signal_name in _strings(params.get("signals", [])):
		if not _identifier(signal_name) or signal_name in members or signal_name + "EventHandler" in members:
			return _error("signal name must be a valid, non-conflicting C# identifier: %s" % signal_name)
		members.append(signal_name)
		members.append(signal_name + "EventHandler")
		signal_names.append(signal_name)
		declarations.append("\t[Signal]\n\tpublic delegate void %sEventHandler();" % signal_name)
	var export_specs := _strings(params.get("exports", []))
	for spec in export_specs:
		var equal_index := spec.find("=")
		var declaration := spec if equal_index == -1 else spec.substr(0, equal_index)
		var colon_index := declaration.find(":")
		if colon_index <= 0:
			return _error("export must use Name:Type or Name:Type=value: %s" % spec)
		var member_name := declaration.substr(0, colon_index).strip_edges()
		var member_type := declaration.substr(colon_index + 1).strip_edges()
		if not _identifier(member_name) or member_name in members or not _safe_type(member_type):
			return _error("export requires a non-conflicting C# identifier and valid type: %s" % spec)
		var line := "\t[Export] public %s %s { get; set; }" % [member_type, member_name]
		if equal_index != -1:
			var value := spec.substr(equal_index + 1).strip_edges()
			if value == "" or value.contains("\n") or value.contains("\r"):
				return _error("C# export default must be a non-empty single-line C# expression: %s" % spec)
			line += " = %s;" % value
		members.append(member_name)
		declarations.append(line)
	var lines: Array[String] = ["using Godot;", ""]
	if bool(params.get("tool", false)):
		lines.append("[Tool]")
	lines.append("public partial class %s : %s" % [script_class, base_class])
	lines.append("{")
	for declaration in declarations:
		lines.append(declaration)
		lines.append("")
	for raw_key in STUBS:
		var key := String(raw_key)
		if bool(params.get(key, false)):
			lines.append("\tpublic override void %s\n\t{\n\t}\n" % String(STUBS[key]))
	lines.append("}")
	lines.append("")
	return {"ok": true, "text": "\n".join(lines), "extends": base_class, "class_name": script_class, "tool": bool(params.get("tool", false)), "signals": signal_names, "exports": export_specs.size()}


func _identifier(value: String) -> bool:
	if value == "" or value in KEYWORDS.split(" "):
		return false
	for index in range(value.length()):
		var code := value.unicode_at(index)
		if not (code == 95 or (code >= 65 and code <= 90) or (code >= 97 and code <= 122) or (index > 0 and code >= 48 and code <= 57)):
			return false
	return true


func _qualified_identifier(value: String) -> bool:
	for part in value.split("."):
		if not _identifier(part):
			return false
	return value != ""


func _safe_type(value: String) -> bool:
	if value.ends_with("[]"):
		return _safe_type(value.trim_suffix("[]"))
	if value.ends_with("?"):
		return _safe_type(value.trim_suffix("?"))
	var generic_start := value.find("<")
	if generic_start != -1:
		if not value.ends_with(">") or not _qualified_identifier(value.substr(0, generic_start)):
			return false
		var depth := 0
		var start := generic_start + 1
		for index in range(start, value.length() - 1):
			var character := value.substr(index, 1)
			if character == "<":
				depth += 1
			elif character == ">":
				depth -= 1
				if depth < 0:
					return false
			elif character == "," and depth == 0:
				if not _safe_type(value.substr(start, index - start).strip_edges()):
					return false
				start = index + 1
		return depth == 0 and _safe_type(value.substr(start, value.length() - 1 - start).strip_edges())
	return value in PRIMITIVES or _qualified_identifier(value)


func _strings(raw: Variant) -> Array[String]:
	var values: Array[String] = []
	if raw is Array:
		for value in raw:
			values.append(String(value))
	return values


func _error(message: String) -> Dictionary:
	return {"ok": false, "error": message}
