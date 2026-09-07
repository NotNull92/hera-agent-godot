extends RefCounted

const BUILD_WARNING := "C# metadata comes from the loaded assembly and may be stale. Build the .NET project and reload the assembly after source changes; Hera does not run builds or create .csproj/.sln files."


static func available() -> bool:
	return ClassDB.class_exists("CSharpScript")


static func inspect(path: String, script: Script) -> Dictionary:
	var base_type := "" if script == null else String(script.get_instance_base_type())
	var data := {
		"language": "csharp",
		"class_name": path.get_file().get_basename(),
		"extends": base_type,
		"base_type": base_type,
		"functions": [],
		"signals": [],
		"exports": [],
		"metadata_source": "assembly",
		"assembly_loaded": base_type != "",
		"csharp_supported": available(),
		"build_warning": BUILD_WARNING,
	}
	if base_type == "":
		return data
	data["functions"] = _callables(script.get_script_method_list())
	data["signals"] = _callables(script.get_script_signal_list())
	var exports: Array[Dictionary] = []
	for property in script.get_script_property_list():
		var usage := int(property.get("usage", 0))
		if (usage & PROPERTY_USAGE_EDITOR) == 0 or (usage & PROPERTY_USAGE_SCRIPT_VARIABLE) == 0:
			continue
		exports.append({"name": String(property.get("name", "")), "type": type_string(int(property.get("type", TYPE_NIL)))})
	data["exports"] = exports
	return data


static func _callables(methods: Array[Dictionary]) -> Array[Dictionary]:
	var summaries: Array[Dictionary] = []
	for method in methods:
		var args: Array = method.get("args", [])
		var names: Array[String] = []
		for raw_arg in args:
			var arg: Dictionary = raw_arg
			names.append(String(arg.get("name", "")))
		var name := String(method.get("name", ""))
		summaries.append({"name": name, "signature": "%s(%s)" % [name, ", ".join(names)]})
	return summaries
