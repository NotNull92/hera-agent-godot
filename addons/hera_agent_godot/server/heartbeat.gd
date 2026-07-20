extends RefCounted

# Advertises this editor instance to the CLI by writing
# ~/.hera-agent-godot/instances/<pid>.json. The plugin calls write() on a timer
# (~0.5s); the CLI scans the directory and treats an instance as live only while
# its "ts" is fresh. See docs/ARCHITECTURE.md §6.

const DIR_NAME := ".hera-agent-godot"

# The plugin republishes every 0.5s and the CLI treats an instance as live for
# 5s, so it takes ~10 consecutive failures before a missed publish can actually
# hide this editor. Warn just under that, once, instead of on every miss.
const PUBLISH_WARN_AFTER := 8

var port := 0

var _path := ""
var _publish_failures := 0

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
		if err == OK:
			_publish_failures = 0
			return
		DirAccess.remove_absolute(tmp_path)
		# On Windows DirAccess.rename removes the destination before MoveFileW,
		# and that remove fails while another process has <pid>.json open — i.e.
		# whenever the CLI happens to be reading it. The previous file is still
		# on disk and still fresh, and the next tick republishes, so a lone miss
		# is expected rather than a fault. Only a run of them can hide us.
		_publish_failures += 1
		if _publish_failures == PUBLISH_WARN_AFTER:
			push_warning("[hera] heartbeat has failed to publish %d times in a row: %s" % [_publish_failures, error_string(err)])

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
