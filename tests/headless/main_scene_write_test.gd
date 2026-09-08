extends SceneTree

const ProjectTool = preload("res://addons/hera_agent_godot/tools/project_tool.gd")

func _initialize() -> void:
	var failures: Array[String] = []
	var tool := ProjectTool.new()
	var scene_path := "res://MainSceneWriteFixture.tscn"
	var root := Node.new()
	root.name = "MainSceneWriteFixture"
	var scene := PackedScene.new()
	scene.pack(root)
	ResourceSaver.save(scene, scene_path)
	root.free()
	var original := FileAccess.get_file_as_bytes("res://project.godot")
	var old_main: Variant = ProjectSettings.get_setting("application/run/main_scene", null)
	var old_name: Variant = ProjectSettings.get_setting("application/config/name", null)
	ProjectSettings.set_setting("application/config/name", "Unterminated fixture")
	var project_file := FileAccess.open("res://project.godot", FileAccess.WRITE)
	project_file.store_string("[application]\nconfig/name=\"Unterminated fixture\"")
	project_file.close()
	var response: Dictionary = tool.execute({"action": "set_main_scene", "path": scene_path})
	var data: Dictionary = response.get("data", {})
	var disk := ConfigFile.new()
	if not bool(response.get("ok", false)) or disk.load("res://project.godot") != OK:
		failures.append("main scene setter failed or corrupted project.godot")
	elif disk.get_value("application", "config/name", "") != "Unterminated fixture" or disk.get_value("application", "run/main_scene", "") != scene_path:
		failures.append("engine serialization lost project name or main scene")
	if ProjectSettings.get_setting("application/run/main_scene", "") != scene_path:
		failures.append("main scene was not updated in memory")
	if data.get("main_scene", "") != scene_path or data.get("project_path", "") != ProjectSettings.globalize_path("res://").trim_suffix("/"):
		failures.append("main scene response is incomplete")
	var saved := FileAccess.get_file_as_bytes("res://project.godot")
	for path in ["", "../outside.tscn", "res://../outside.tscn", "res://missing.tscn", "res://bad.gd", "res://a\\bad.tscn"]:
		var rejected: Dictionary = tool.execute({"action": "set_main_scene", "path": path})
		if bool(rejected.get("ok", true)) or ProjectSettings.get_setting("application/run/main_scene", "") != scene_path or FileAccess.get_file_as_bytes("res://project.godot") != saved:
			failures.append("invalid main scene path changed settings: %s" % path)
	ProjectSettings.set_setting("application/run/main_scene", old_main)
	ProjectSettings.set_setting("application/config/name", old_name)
	project_file = FileAccess.open("res://project.godot", FileAccess.WRITE)
	project_file.store_buffer(original)
	project_file.close()
	DirAccess.remove_absolute(ProjectSettings.globalize_path(scene_path))
	for failure in failures:
		push_error(failure)
	quit(0 if failures.is_empty() else 1)
