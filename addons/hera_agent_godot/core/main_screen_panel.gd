extends RefCounted

const PANEL_NAME := "HeraAgentMainScreen"
const HERA_LOGO_PATH := "res://docs/assets/hera-pointing.png"
const HERA_TITLE_FONT_PATH := "res://addons/hera_agent_godot/assets/fonts/cormorant-italic.woff2"
const HERA_TITLE_FONT_EMBOLDEN := 0.85
const HERA_DEEP_SPACE := Color(0.0, 0.0, 0.063)
const HERA_NIGHT_PANEL := Color(0.063, 0.125, 0.188)
const HERA_ICE := Color(0.753, 0.878, 0.941)
const HERA_MUTED_BLUE := Color(0.502, 0.690, 0.878)
const HERA_GODOT_BLUE := Color(0.314, 0.502, 0.753)
const HERA_WARM_GOLD := Color(0.878, 0.627, 0.502)
const HERA_TERMINAL_GREEN := Color(0.439, 0.929, 0.627)
const HERA_OFFLINE_RED := Color(0.961, 0.325, 0.325)


static func create(ui_enabled: bool, game_feel_enabled: bool, ui_callback: Callable, game_feel_callback: Callable) -> Dictionary:
	var main_screen := EditorInterface.get_editor_main_screen()
	var stale_panel := main_screen.get_node_or_null(PANEL_NAME)
	if stale_panel != null:
		main_screen.remove_child(stale_panel)
		stale_panel.queue_free()

	var panel := MarginContainer.new()
	panel.name = PANEL_NAME
	panel.set_anchors_preset(Control.PRESET_FULL_RECT)
	panel.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	panel.size_flags_vertical = Control.SIZE_EXPAND_FILL
	panel.add_theme_constant_override("margin_left", 28)
	panel.add_theme_constant_override("margin_top", 28)
	panel.add_theme_constant_override("margin_right", 28)
	panel.add_theme_constant_override("margin_bottom", 28)

	var content := _build_shell(panel)
	_build_header(content)
	var status_refs := _build_status(content)
	var game_feel_toggle := _add_settings_toggle_card(
		content,
		"Game Feel Mode(Beta)",
		"Turn me on and I'll guide gameplay feel: controls, camera, hit stop, screen shake, sound, particles, rewards, and Honest Juice.",
		game_feel_enabled,
		"Tell Hera to point gameplay work at the bundled Game Feel knowledge base.",
		game_feel_callback
	)
	var ui_toggle := _add_settings_toggle_card(
		content,
		"Game Feel UI Mode(Beta)",
		"Turn me on and I'll inject Game Feel into UI: snappy feedback, juicy motion, and satisfying interactions.",
		ui_enabled,
		"Tell Hera to favor Game Feel when guiding UI work.",
		ui_callback
	)

	main_screen.add_child(panel)
	return {
		"panel": panel,
		"status_label": status_refs["label"],
		"status_dot": status_refs["dot"],
		"ui_toggle": ui_toggle,
		"game_feel_toggle": game_feel_toggle,
	}


static func set_status(label: Label, dot: PanelContainer, text: String, connected: bool) -> void:
	var status_color := HERA_TERMINAL_GREEN if connected else HERA_OFFLINE_RED
	if label != null:
		label.text = text
		label.add_theme_color_override("font_color", status_color)
	if dot != null:
		dot.add_theme_stylebox_override("panel", _make_stylebox(status_color, status_color, 0, 6))


static func _build_shell(panel: MarginContainer) -> VBoxContainer:
	var layout := VBoxContainer.new()
	layout.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	layout.size_flags_vertical = Control.SIZE_EXPAND_FILL
	layout.add_theme_constant_override("separation", 14)
	panel.add_child(layout)

	var shell := PanelContainer.new()
	shell.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	shell.size_flags_vertical = Control.SIZE_EXPAND_FILL
	shell.add_theme_stylebox_override("panel", _make_stylebox(HERA_DEEP_SPACE, HERA_WARM_GOLD.darkened(0.16), 1, 10))
	layout.add_child(shell)

	var shell_margin := MarginContainer.new()
	shell_margin.add_theme_constant_override("margin_left", 24)
	shell_margin.add_theme_constant_override("margin_top", 22)
	shell_margin.add_theme_constant_override("margin_right", 24)
	shell_margin.add_theme_constant_override("margin_bottom", 22)
	shell.add_child(shell_margin)

	var content := VBoxContainer.new()
	content.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	content.size_flags_vertical = Control.SIZE_EXPAND_FILL
	content.add_theme_constant_override("separation", 16)
	shell_margin.add_child(content)
	return content


static func _build_header(content: VBoxContainer) -> void:
	var header := HBoxContainer.new()
	header.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	header.add_theme_constant_override("separation", 12)
	content.add_child(header)

	var logo := _make_logo_texture()
	if logo != null:
		header.add_child(logo)

	var title_stack := VBoxContainer.new()
	title_stack.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	title_stack.add_theme_constant_override("separation", 4)
	header.add_child(title_stack)

	var title := Label.new()
	title.text = "It's me Hera"
	title.add_theme_color_override("font_color", HERA_ICE)
	var title_font := _load_display_font(HERA_TITLE_FONT_PATH)
	if title_font != null:
		title.add_theme_font_override("font", title_font)
	title.add_theme_font_size_override("font_size", 32)
	title_stack.add_child(title)

	var summary := Label.new()
	summary.text = "I give your AI agent real-time eyes and hands in Godot 4.7+ - low-token commands to inspect, edit, run, QA, and screenshot the live editor."
	summary.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	summary.add_theme_color_override("font_color", HERA_MUTED_BLUE)
	title_stack.add_child(summary)

	var badge := _make_pill("Sexy CLI", HERA_WARM_GOLD)
	badge.size_flags_horizontal = Control.SIZE_SHRINK_END
	header.add_child(badge)

	var divider := ColorRect.new()
	divider.custom_minimum_size = Vector2(0, 1)
	divider.color = Color(HERA_WARM_GOLD.r, HERA_WARM_GOLD.g, HERA_WARM_GOLD.b, 0.30)
	content.add_child(divider)


static func _build_status(content: VBoxContainer) -> Dictionary:
	var status_card := _make_card()
	content.add_child(status_card)

	var status_row := HBoxContainer.new()
	status_row.add_theme_constant_override("separation", 10)
	status_card.add_child(status_row)

	var status_dot := PanelContainer.new()
	status_dot.custom_minimum_size = Vector2(12, 12)
	status_dot.size_flags_vertical = Control.SIZE_SHRINK_CENTER
	status_dot.add_theme_stylebox_override("panel", _make_stylebox(HERA_OFFLINE_RED, HERA_OFFLINE_RED, 0, 6))
	status_row.add_child(status_dot)

	var status_label := Label.new()
	status_label.text = "Starting local bridge..."
	status_label.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	var status_font := _load_display_font(HERA_TITLE_FONT_PATH)
	if status_font != null:
		status_label.add_theme_font_override("font", status_font)
	status_label.add_theme_font_size_override("font_size", 17)
	status_label.add_theme_color_override("font_color", HERA_OFFLINE_RED)
	status_row.add_child(status_label)

	var locality := _make_pill("127.0.0.1", HERA_TERMINAL_GREEN)
	locality.size_flags_horizontal = Control.SIZE_SHRINK_END
	status_row.add_child(locality)

	return {"label": status_label, "dot": status_dot}


static func _add_settings_toggle_card(content: VBoxContainer, title_text: String, summary_text: String, enabled: bool, tooltip: String, callback: Callable) -> CheckButton:
	var settings_card := _make_card()
	content.add_child(settings_card)

	var settings_row := HBoxContainer.new()
	settings_row.add_theme_constant_override("separation", 14)
	settings_card.add_child(settings_row)

	var setting_text := VBoxContainer.new()
	setting_text.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	setting_text.add_theme_constant_override("separation", 3)
	settings_row.add_child(setting_text)

	var setting_title := Label.new()
	setting_title.text = title_text
	setting_title.add_theme_color_override("font_color", HERA_ICE)
	var setting_title_font := _load_display_font(HERA_TITLE_FONT_PATH)
	if setting_title_font != null:
		setting_title.add_theme_font_override("font", setting_title_font)
	setting_title.add_theme_font_size_override("font_size", 20)
	setting_text.add_child(setting_title)

	var setting_summary := Label.new()
	setting_summary.text = summary_text
	setting_summary.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	setting_summary.add_theme_color_override("font_color", HERA_MUTED_BLUE)
	setting_text.add_child(setting_summary)

	var toggle := CheckButton.new()
	toggle.text = "On/Off"
	toggle.button_pressed = enabled
	toggle.tooltip_text = tooltip
	toggle.size_flags_horizontal = Control.SIZE_SHRINK_END
	toggle.size_flags_vertical = Control.SIZE_SHRINK_CENTER
	toggle.add_theme_color_override("font_color", HERA_ICE)
	toggle.add_theme_color_override("font_pressed_color", HERA_WARM_GOLD)
	toggle.add_theme_color_override("font_hover_color", HERA_ICE)
	toggle.toggled.connect(callback)
	settings_row.add_child(toggle)
	return toggle


static func _make_card() -> PanelContainer:
	var card := PanelContainer.new()
	card.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	card.add_theme_constant_override("margin_left", 14)
	card.add_theme_constant_override("margin_top", 12)
	card.add_theme_constant_override("margin_right", 14)
	card.add_theme_constant_override("margin_bottom", 12)
	card.add_theme_stylebox_override("panel", _make_stylebox(HERA_NIGHT_PANEL, Color(HERA_GODOT_BLUE.r, HERA_GODOT_BLUE.g, HERA_GODOT_BLUE.b, 0.46), 1, 8))
	return card


static func _make_logo_texture() -> TextureRect:
	var image := Image.new()
	var load_error := image.load(ProjectSettings.globalize_path(HERA_LOGO_PATH))
	if load_error != OK:
		return null
	var texture := ImageTexture.create_from_image(image)
	if texture == null:
		return null
	var logo := TextureRect.new()
	logo.texture = texture
	logo.custom_minimum_size = Vector2(86, 86)
	logo.expand_mode = TextureRect.EXPAND_IGNORE_SIZE
	logo.stretch_mode = TextureRect.STRETCH_KEEP_ASPECT_CENTERED
	logo.size_flags_horizontal = Control.SIZE_SHRINK_BEGIN
	logo.size_flags_vertical = Control.SIZE_SHRINK_CENTER
	logo.tooltip_text = "Hera"
	return logo


static func _load_display_font(path: String) -> Font:
	var base_font := ResourceLoader.load(path, "Font") as Font
	if base_font == null:
		return null
	var display_font := FontVariation.new()
	display_font.base_font = base_font
	display_font.variation_embolden = HERA_TITLE_FONT_EMBOLDEN
	return display_font


static func _make_pill(text: String, accent: Color) -> Label:
	var label := Label.new()
	label.text = text
	label.add_theme_color_override("font_color", accent)
	label.add_theme_font_size_override("font_size", 12)
	label.add_theme_stylebox_override("normal", _make_stylebox(Color(accent.r, accent.g, accent.b, 0.10), Color(accent.r, accent.g, accent.b, 0.34), 1, 7))
	return label


static func _make_stylebox(bg: Color, border: Color, border_width: int, radius: int) -> StyleBoxFlat:
	var box := StyleBoxFlat.new()
	box.bg_color = bg
	box.border_color = border
	box.border_width_left = border_width
	box.border_width_top = border_width
	box.border_width_right = border_width
	box.border_width_bottom = border_width
	box.corner_radius_top_left = radius
	box.corner_radius_top_right = radius
	box.corner_radius_bottom_left = radius
	box.corner_radius_bottom_right = radius
	box.content_margin_left = 10
	box.content_margin_top = 6
	box.content_margin_right = 10
	box.content_margin_bottom = 6
	return box
