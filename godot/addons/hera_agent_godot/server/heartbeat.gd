extends RefCounted

# Advertises this editor instance to the CLI by writing
# ~/.hera-agent-godot/instances/<pid>.json. The plugin calls write() on a timer
# (~0.5s); the CLI scans the directory and treats an instance as live only while
# its "ts" is fresh. See docs/ARCHITECTURE.md §6.

const DIR_NAME := ".hera-agent-godot"

var port := 0

var _path := ""

func start(listen_port: int) -> void:
	port = listen_port
	var dir := _instances_dir()
	DirAccess.make_dir_recursive_absolute(dir)
	_path = dir.path_join("%d.json" % OS.get_process_id())
	write()

func write() -> void:
	if _path == "":
		return
	var data := {
		"pid": OS.get_process_id(),
		"port": port,
		"project_path": ProjectSettings.globalize_path("res://"),
		"godot_version": String(Engine.get_version_info().get("string", "")),
		"scene": _current_scene(),
		"ts": int(Time.get_unix_time_from_system()),
	}
	var tmp_path := "%s.tmp.%d" % [_path, Time.get_ticks_usec()]
	var file := FileAccess.open(tmp_path, FileAccess.WRITE)
	if file != null:
		file.store_string(JSON.stringify(data))
		file.flush()
		file.close()
		var err := DirAccess.rename_absolute(tmp_path, _path)
		if err != OK:
			DirAccess.remove_absolute(tmp_path)
			push_warning("[hera] failed to publish heartbeat: %s" % error_string(err))

func stop() -> void:
	if _path != "" and FileAccess.file_exists(_path):
		DirAccess.remove_absolute(_path)
	_path = ""
	port = 0

func _current_scene() -> String:
	var root := EditorInterface.get_edited_scene_root()
	if root == null:
		return ""
	return root.scene_file_path

func _instances_dir() -> String:
	return _home_dir().path_join(DIR_NAME).path_join("instances")

# Match Go's os.UserHomeDir(): USERPROFILE on Windows, HOME elsewhere.
func _home_dir() -> String:
	if OS.has_environment("USERPROFILE"):
		return OS.get_environment("USERPROFILE")
	if OS.has_environment("HOME"):
		return OS.get_environment("HOME")
	return OS.get_user_data_dir()
