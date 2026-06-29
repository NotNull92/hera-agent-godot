extends RefCounted


func inspect(source: String) -> Dictionary:
	var script_class_name := ""
	var extends_name := ""
	var functions := []
	var signals := []
	var exports := []
	var export_next := false
	for raw_line in source.split("\n", false):
		var line := _code_only(String(raw_line)).strip_edges()
		if line == "" or line.begins_with("#"):
			continue
		if line.begins_with("class_name "):
			script_class_name = _first_identifier(line.substr("class_name ".length()))
		elif line.begins_with("extends "):
			extends_name = line.substr("extends ".length()).strip_edges()
		elif line.begins_with("signal "):
			signals.append(_callable_summary(line.substr("signal ".length())))
		elif line.begins_with("func "):
			functions.append(_callable_summary(line.substr("func ".length())))
		elif line.begins_with("@export"):
			var var_index := line.find("var ")
			if var_index != -1:
				exports.append(_variable_summary(line.substr(var_index + "var ".length())))
				export_next = false
			else:
				export_next = true
		elif export_next and line.begins_with("var "):
			exports.append(_variable_summary(line.substr("var ".length())))
			export_next = false
		elif not line.begins_with("@"):
			export_next = false
	return {
		"class_name": script_class_name,
		"extends": extends_name,
		"functions": functions,
		"signals": signals,
		"exports": exports,
	}


func _code_only(line: String) -> String:
	var out := ""
	var quote := ""
	var escaped := false
	for index in range(line.length()):
		var ch := line.substr(index, 1)
		if quote != "":
			if escaped:
				escaped = false
			elif ch == "\\":
				escaped = true
			elif ch == quote:
				quote = ""
			continue
		if ch == "#":
			break
		if ch == "\"" or ch == "'":
			quote = ch
			continue
		out += ch
	return out


func _callable_summary(text: String) -> Dictionary:
	var signature := text.strip_edges()
	var paren := signature.find("(")
	var name := signature if paren == -1 else signature.substr(0, paren).strip_edges()
	return {
		"name": name,
		"signature": signature,
	}


func _variable_summary(text: String) -> Dictionary:
	var declaration := text.strip_edges()
	var end := declaration.length()
	for marker in [":", "=", " "]:
		var index := declaration.find(marker)
		if index != -1 and index < end:
			end = index
	return {
		"name": declaration.substr(0, end),
		"declaration": declaration,
	}


func _first_identifier(text: String) -> String:
	var words := text.strip_edges().split(" ", false)
	return "" if words.is_empty() else String(words[0])
