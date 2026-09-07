extends SceneTree

const ExternalLauncher = preload("res://addons/hera_agent_godot/runtime/external_launcher.gd")

func _init() -> void:
	call_deferred("_run")

func _run() -> void:
	var valid := ExternalLauncher.parse_args(PackedStringArray(["res://game/Main.tscn", "1080", "1920", "1900", "20"]))
	check(valid.get("size") == Vector2i(1080, 1920), "native portrait size is accepted")
	check(valid.get("position") == Vector2i(1900, 20), "bounded position is accepted")
	check(ExternalLauncher.parse_args(PackedStringArray(["../Main.tscn", "1080", "1920", "0", "0"])).has("error"), "non-resource scene is rejected")
	check(ExternalLauncher.parse_args(PackedStringArray(["res://game/Main.tscn", "oops", "1920", "0", "0"])).has("error"), "malformed size is rejected")
	check(ExternalLauncher.parse_args(PackedStringArray(["res://game/Main.tscn", "1080", "9000", "0", "0"])).has("error"), "oversized window is rejected")
	if not _failed:
		print("PASS: external launcher arguments")
	quit(1 if _failed else 0)

var _failed := false

func check(value: bool, message: String) -> void:
	if not value:
		_failed = true
		push_error("FAIL: %s" % message)
