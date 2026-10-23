@tool
extends EditorPlugin

func _enter_tree() -> void:
	_run.call_deferred()

func _run() -> void:
	await get_tree().create_timer(2.0).timeout
	var settings: EditorSettings = EditorInterface.get_editor_settings()
	var phase := OS.get_environment("HERA_UI_PHASE")
	if phase == "configure":
		var theme_key := "interface/theme/color_preset" if settings.has_setting("interface/theme/color_preset") else "interface/theme/preset"
		settings.set_setting(theme_key, OS.get_environment("HERA_UI_THEME"))
		for property: Dictionary in settings.get_property_list():
			if String(property.name).ends_with("/display_scale"):
				var choices: PackedStringArray = String(property.hint_string).split(",")
				var wanted := "150%" if OS.get_environment("HERA_UI_SCALE") == "1.5" else "100%"
				for index: int in choices.size():
					if choices[index].begins_with(wanted):
						settings.set_setting(property.name, index)
						print("SCALE_SETTING ", property.name, "=", index, " ", property.hint_string)
		get_tree().quit()
		return
	EditorInterface.set_main_screen_editor("HeraAgent")
	await get_tree().create_timer(2.0).timeout
	var panel: Control = EditorInterface.get_editor_main_screen().get_node("HeraAgentMainScreen")
	assert(panel.is_visible_in_tree())
	assert(not DirAccess.dir_exists_absolute("res://docs"))
	var character := panel.find_child("HeaderCharacter", true, false) as TextureRect
	assert(character != null and character.texture != null and character.is_visible_in_tree())
	# Python sends status from the release CLI while the editor keeps processing.
	for attempt: int in range(100):
		if panel.rows["version"].text.contains("Versions match"):
			break
		await get_tree().create_timer(0.1).timeout
	var version_text: String = panel.rows["version"].text
	assert(version_text.contains("Versions match"))
	var expected_version := OS.get_environment("HERA_UI_VERSION")
	assert(not expected_version.is_empty())
	assert(version_text.contains("CLI v" + expected_version) and version_text.contains("Addon " + expected_version))
	print("RELEASE_VERSION_MATCH ", version_text.replace("\n", " | "))
	var expected_scale := float(OS.get_environment("HERA_UI_SCALE"))
	print("OBSERVED_SCALE ", EditorInterface.get_editor_scale())
	assert(is_equal_approx(EditorInterface.get_editor_scale(), expected_scale))
	var original_clipboard := DisplayServer.clipboard_get()
	panel._copy_setup()
	var clipboard_matches: bool = DisplayServer.clipboard_get().replace("\r\n", "\n") == panel.agent_setup_text()
	DisplayServer.clipboard_set(original_clipboard)
	assert(clipboard_matches)
	assert(panel.copied.visible)
	await get_tree().create_timer(2.2).timeout
	assert(not panel.copied.visible)
	DisplayServer.clipboard_set(original_clipboard)
	await RenderingServer.frame_post_draw
	var image: Image = get_tree().root.get_texture().get_image()
	assert(image.save_png(OS.get_environment("HERA_UI_SCREENSHOT")) == OK)
	if expected_scale > 1.0:
		var scroll: ScrollContainer = panel.get_child(0)
		scroll.ensure_control_visible(panel.content.get_node("GameFeel"))
		await get_tree().process_frame
		await RenderingServer.frame_post_draw
		assert(scroll.scroll_vertical > 0)
		image = get_tree().root.get_texture().get_image()
		assert(image.save_png(OS.get_environment("HERA_UI_SCREENSHOT").get_basename() + "-bottom.png") == OK)
	print("HERA_UI_CAPTURE_OK scale=", EditorInterface.get_editor_scale(), " theme=", OS.get_environment("HERA_UI_THEME"))
	get_tree().quit()
