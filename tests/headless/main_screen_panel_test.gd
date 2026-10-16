# Hera test: editor
extends SceneTree

const MainPanel = preload("res://addons/hera_agent_godot/core/main_screen_panel.gd")
const Settings = preload("res://addons/hera_agent_godot/core/hera_settings.gd")
var failed := false

func _initialize() -> void:
	_run.call_deferred()

func check(ok: bool, message: String) -> void:
	if not ok:
		failed = true
		push_error(message)

func _run() -> void:
	print("panel test: strings and version states")
	for key: String in MainPanel.STRINGS:
		var pair: Array = MainPanel.STRINGS[key]
		check(pair.size() == 2 and not String(pair[0]).is_empty() and not String(pair[1]).is_empty(), "Missing translation: " + key)
	check(MainPanel.version_state("", "1.2.0", false) == "not_seen", "no request")
	check(MainPanel.version_state("", "1.2.0", true) == "unknown", "old CLI")
	check(MainPanel.version_state("dev", "1.2.0", true) == "unknown", "development CLI")
	check(MainPanel.version_state("v1.2.9", "1.2.0", true) == "match", "patch match")
	check(MainPanel.version_state("1.1.0", "1.2.0", true) == "mismatch", "minor mismatch")
	check(MainPanel.version_state("2.2.0", "1.2.0", true) == "mismatch", "major mismatch")
	check(MainPanel.version_state("garbage1.2.0", "1.2.0", true) == "unknown", "invalid version")
	var source := FileAccess.get_file_as_string("res://addons/hera_agent_godot/core/main_screen_panel.gd")
	check(not source.contains("Sexy CLI") and not source.contains("Color("), "theme-only colors and approved copy")
	var refs: Dictionary = MainPanel.create(false, false, Settings.set_game_feel_ui_mode_enabled, Settings.set_game_feel_mode_enabled)
	print("panel test: created")
	var panel: Control = refs["panel"]
	check(panel.content.get_child(0).name == "Header", "header first")
	check(not DirAccess.dir_exists_absolute("res://docs"), "installed-addon fixture must not contain docs/")
	var character := panel.content.get_child(0).find_child("HeaderCharacter", true, false) as TextureRect
	check(character != null and character.texture != null, "header character ships with the addon")
	if character != null and character.texture != null:
		check(character.texture.resource_path == "res://addons/hera_agent_godot/assets/hera-pointing.png", "character uses addon asset")
	check(panel.content.get_child(1).name == "Status", "status second")
	check(panel.content.get_child(2).name == "CopySetup", "copy third")
	check(panel.content.get_child(3).name == "GameFeel", "settings last")
	panel.update_observations("1.1.0", true, 8770, true)
	panel.set_connection("listening", true)
	OS.set_environment("HERA_AGENT_GODOT_TOKEN", "secret-never-copy")
	var setup: String = panel.agent_setup_text()
	check(setup.contains("Auth is on") and setup.contains("--lang gdscript"), "setup guidance")
	check(not setup.contains("secret-never-copy") and not setup.contains("HERA_AGENT_GODOT_TOKEN") and not setup.contains(".hera-agent-godot/token"), "no secrets or token paths")
	panel.set_connection("token_error", false)
	check(panel.agent_setup_text().contains("Auth state is unknown"), "unreadable token is not auth off")
	panel.update_observations("2.0.9", true, 8770, false)
	check(panel.rows["version"].text.contains("Patch versions differ"), "patch difference is explained")
	panel.set_connection("listening", true)
	check(Settings.GAME_FEEL_MODE_SETTING == "hera_agent_godot/game_feel_mode", "game key preserved")
	check(Settings.GAME_FEEL_UI_MODE_SETTING == "hera_agent_godot/ui_juicy_mode", "UI key preserved")
	var toggle: CheckButton = refs["ui_toggle"]
	toggle.button_pressed = true
	check(Settings.get_game_feel_ui_mode_enabled(), "toggle uses existing setting")
	toggle.button_pressed = false
	var custom_theme := Theme.new()
	print("panel test: changing theme")
	custom_theme.set_color("success_color", "Editor", Color.REBECCA_PURPLE)
	panel.theme = custom_theme
	await process_frame
	print("panel test: theme changed")
	check(panel.rows["connection"].get_theme_color("font_color") == panel.get_theme_color("success_color", "Editor"), "theme repaint")
	panel.free()
	print("main_screen_panel_test: PASS" if not failed else "main_screen_panel_test: FAIL")
	quit(1 if failed else 0)

