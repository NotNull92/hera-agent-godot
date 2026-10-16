extends SceneTree

func _initialize() -> void:
	_run.call_deferred()

func _run() -> void:
	var filesystem: EditorFileSystem = EditorInterface.get_resource_filesystem()
	await create_timer(1.0).timeout
	while filesystem.is_scanning():
		await process_frame
	var font: Font = load("res://addons/hera_agent_godot/assets/fonts/cormorant-italic.woff2") as Font
	if font == null:
		push_error("Editor import did not produce a usable title font")
		quit(1)
		return
	print("HERA_IMPORT_OK")
	quit(0)
