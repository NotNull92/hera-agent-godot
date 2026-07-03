@tool
extends EditorPlugin

const ToolRegistry = preload("res://addons/hera_agent_godot/core/tool_registry.gd")
const ToolResponse = preload("res://addons/hera_agent_godot/core/tool_response.gd")
const HttpServer = preload("res://addons/hera_agent_godot/server/http_server.gd")
const WorkQueue = preload("res://addons/hera_agent_godot/server/work_queue.gd")
const Heartbeat = preload("res://addons/hera_agent_godot/server/heartbeat.gd")
const StatusTool = preload("res://addons/hera_agent_godot/tools/status_tool.gd")
const RunTool = preload("res://addons/hera_agent_godot/tools/run_tool.gd")
const SceneTool = preload("res://addons/hera_agent_godot/tools/scene_tool.gd")
const EditorTool = preload("res://addons/hera_agent_godot/tools/editor_tool.gd")
const NodeTool = preload("res://addons/hera_agent_godot/tools/node_tool.gd")
const ScriptTool = preload("res://addons/hera_agent_godot/tools/script_tool.gd")
const SignalTool = preload("res://addons/hera_agent_godot/tools/signal_tool.gd")
const ResourceTool = preload("res://addons/hera_agent_godot/tools/resource_tool.gd")
const ProjectTool = preload("res://addons/hera_agent_godot/tools/project_tool.gd")
const ClassDBTool = preload("res://addons/hera_agent_godot/tools/classdb_tool.gd")
const EvalTool = preload("res://addons/hera_agent_godot/tools/eval_tool.gd")
const OutputTool = preload("res://addons/hera_agent_godot/tools/output_tool.gd")
const DiagnosticsTool = preload("res://addons/hera_agent_godot/tools/diagnostics_tool.gd")
const GameTool = preload("res://addons/hera_agent_godot/tools/game_tool.gd")
const ScreenshotTool = preload("res://addons/hera_agent_godot/tools/screenshot_tool.gd")
const BatchTool = preload("res://addons/hera_agent_godot/tools/batch_tool.gd")

const HEARTBEAT_INTERVAL := 0.5
const GAME_AUTOLOAD_NAME := "HeraGameInspector"
const GAME_AUTOLOAD_PATH := "res://addons/hera_agent_godot/runtime/game_inspector.gd"
const MAIN_SCREEN_PLUGIN_NAME := "HeraAgent"
const MAIN_SCREEN_PANEL_NAME := "HeraAgentMainScreen"
const UI_JUICY_MODE_SETTING := "hera_agent_godot/ui_juicy_mode"
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

var _registry: RefCounted
var _server: RefCounted
var _queue: RefCounted
var _heartbeat: RefCounted
var _heartbeat_accum := 0.0
var _game_autoload_injected := false
var _main_panel: Control
var _main_status_label: Label
var _main_status_dot: PanelContainer
var _ui_juicy_mode_toggle: CheckButton

func _enter_tree() -> void:
	set_process(true)
	_create_main_screen()
	_ensure_game_autoload()
	_registry = ToolRegistry.new()
	_registry.register(StatusTool.new())
	_registry.register(RunTool.new())
	_registry.register(SceneTool.new())
	_registry.register(EditorTool.new())
	_registry.register(ScriptTool.new())
	_registry.register(ProjectTool.new())
	var node_tool := NodeTool.new()
	node_tool.set_undo_redo(get_undo_redo())
	_registry.register(node_tool)
	var signal_tool := SignalTool.new()
	signal_tool.set_undo_redo(get_undo_redo())
	_registry.register(signal_tool)
	_registry.register(ResourceTool.new())
	_registry.register(ClassDBTool.new())
	_registry.register(EvalTool.new())
	_registry.register(OutputTool.new())
	_registry.register(DiagnosticsTool.new())
	var game_tool := GameTool.new()
	game_tool.set_host(self)
	_registry.register(game_tool)
	var screenshot_tool := ScreenshotTool.new()
	screenshot_tool.set_host(self)
	_registry.register(screenshot_tool)
	var batch_tool := BatchTool.new()
	batch_tool.set_registry(_registry)
	_registry.register(batch_tool)

	_queue = WorkQueue.new()
	_server = HttpServer.new()
	var bound: int = _server.start(8770)
	if bound == 0:
		_server = null
		_set_main_status("Not connected on 127.0.0.1:8770", false)
		push_error("[hera] failed to bind HTTP server on 127.0.0.1 (8770-8785)")
		return

	_heartbeat = Heartbeat.new()
	_heartbeat.start(bound)
	_set_main_status("Listening on 127.0.0.1:%d" % bound, true)
	print("[hera] Hera Agent Godot listening on 127.0.0.1:%d" % bound)

func _process(delta: float) -> void:
	if _server != null:
		_server.poll(_queue)
		for item in _queue.drain():
			_handle(item)

	if _heartbeat != null:
		_heartbeat_accum += delta
		if _heartbeat_accum >= HEARTBEAT_INTERVAL:
			_heartbeat_accum = 0.0
			_heartbeat.write()

func _exit_tree() -> void:
	set_process(false)
	if _heartbeat != null:
		_heartbeat.stop()
		_heartbeat = null
	if _server != null:
		_server.stop()
		_server = null
	_queue = null
	_registry = null
	if _game_autoload_injected:
		remove_autoload_singleton(GAME_AUTOLOAD_NAME)
		_game_autoload_injected = false
	if _main_panel != null:
		_main_panel.queue_free()
		_main_panel = null
		_main_status_label = null
		_main_status_dot = null
		_ui_juicy_mode_toggle = null
	print("[hera] Hera Agent Godot exited")

func _has_main_screen() -> bool:
	return true

func _make_visible(visible: bool) -> void:
	if _main_panel != null:
		_main_panel.visible = visible

func _get_plugin_name() -> String:
	return MAIN_SCREEN_PLUGIN_NAME

func _get_plugin_icon() -> Texture2D:
	return EditorInterface.get_editor_theme().get_icon("Node", "EditorIcons")

func _ensure_game_autoload() -> void:
	var key := "autoload/%s" % GAME_AUTOLOAD_NAME
	if ProjectSettings.has_setting(key):
		return
	add_autoload_singleton(GAME_AUTOLOAD_NAME, GAME_AUTOLOAD_PATH)
	_game_autoload_injected = true

func _handle(item: Dictionary) -> void:
	var request: Dictionary = item["request"]
	var tool_name := String(request.get("tool", ""))
	var tool = _registry.resolve(tool_name) if tool_name != "" else null
	if tool != null and tool.has_method("execute_async"):
		var params: Variant = request.get("params", {})
		if typeof(params) != TYPE_DICTIONARY:
			params = {}
		var response: Dictionary = await tool.execute_async(params)
		if _server != null:
			_server.respond(item["conn"], response)
		else:
			(item["conn"] as StreamPeerTCP).disconnect_from_host()
	else:
		_server.respond(item["conn"], _dispatch(request))

func _dispatch(request: Dictionary) -> Dictionary:
	var tool_name := String(request.get("tool", ""))
	if tool_name == "":
		return ToolResponse.failure("missing tool name")
	var tool = _registry.resolve(tool_name)
	if tool == null:
		return ToolResponse.failure("unknown tool: %s" % tool_name)
	var params: Variant = request.get("params", {})
	if typeof(params) != TYPE_DICTIONARY:
		params = {}
	return tool.execute(params)

func _create_main_screen() -> void:
	var main_screen := EditorInterface.get_editor_main_screen()
	var stale_panel := main_screen.get_node_or_null(MAIN_SCREEN_PANEL_NAME)
	if stale_panel != null:
		main_screen.remove_child(stale_panel)
		stale_panel.queue_free()

	_main_panel = MarginContainer.new()
	_main_panel.name = MAIN_SCREEN_PANEL_NAME
	_main_panel.set_anchors_preset(Control.PRESET_FULL_RECT)
	_main_panel.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	_main_panel.size_flags_vertical = Control.SIZE_EXPAND_FILL
	_main_panel.add_theme_constant_override("margin_left", 28)
	_main_panel.add_theme_constant_override("margin_top", 28)
	_main_panel.add_theme_constant_override("margin_right", 28)
	_main_panel.add_theme_constant_override("margin_bottom", 28)

	var layout := VBoxContainer.new()
	layout.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	layout.size_flags_vertical = Control.SIZE_EXPAND_FILL
	layout.add_theme_constant_override("separation", 14)
	_main_panel.add_child(layout)

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
	summary.text = "I give your AI agent real-time eyes and hands in Godot 4.7+ — low-token commands to inspect, edit, run, QA, and screenshot the live editor."
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

	var status_card := _make_card()
	content.add_child(status_card)

	var status_row := HBoxContainer.new()
	status_row.add_theme_constant_override("separation", 10)
	status_card.add_child(status_row)

	_main_status_dot = PanelContainer.new()
	_main_status_dot.custom_minimum_size = Vector2(12, 12)
	_main_status_dot.size_flags_vertical = Control.SIZE_SHRINK_CENTER
	_main_status_dot.add_theme_stylebox_override("panel", _make_stylebox(HERA_OFFLINE_RED, HERA_OFFLINE_RED, 0, 6))
	status_row.add_child(_main_status_dot)

	_main_status_label = Label.new()
	_main_status_label.text = "Starting local bridge..."
	_main_status_label.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	var status_font := _load_display_font(HERA_TITLE_FONT_PATH)
	if status_font != null:
		_main_status_label.add_theme_font_override("font", status_font)
	_main_status_label.add_theme_font_size_override("font_size", 17)
	_main_status_label.add_theme_color_override("font_color", HERA_OFFLINE_RED)
	status_row.add_child(_main_status_label)

	var locality := _make_pill("127.0.0.1", HERA_TERMINAL_GREEN)
	locality.size_flags_horizontal = Control.SIZE_SHRINK_END
	status_row.add_child(locality)

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
	setting_title.text = "Game Feel UI Mode(Beta)"
	setting_title.add_theme_color_override("font_color", HERA_ICE)
	var setting_title_font := _load_display_font(HERA_TITLE_FONT_PATH)
	if setting_title_font != null:
		setting_title.add_theme_font_override("font", setting_title_font)
	setting_title.add_theme_font_size_override("font_size", 20)
	setting_text.add_child(setting_title)

	var setting_summary := Label.new()
	setting_summary.text = "Turn me on and I'll inject Game Feel into UI: snappy feedback, juicy motion, and satisfying interactions."
	setting_summary.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	setting_summary.add_theme_color_override("font_color", HERA_MUTED_BLUE)
	setting_text.add_child(setting_summary)

	_ui_juicy_mode_toggle = CheckButton.new()
	_ui_juicy_mode_toggle.text = "On/Off"
	_ui_juicy_mode_toggle.button_pressed = _get_ui_juicy_mode_enabled()
	_ui_juicy_mode_toggle.tooltip_text = "Tell Hera to favor Game Feel when guiding UI work."
	_ui_juicy_mode_toggle.size_flags_horizontal = Control.SIZE_SHRINK_END
	_ui_juicy_mode_toggle.size_flags_vertical = Control.SIZE_SHRINK_CENTER
	_ui_juicy_mode_toggle.add_theme_color_override("font_color", HERA_ICE)
	_ui_juicy_mode_toggle.add_theme_color_override("font_pressed_color", HERA_WARM_GOLD)
	_ui_juicy_mode_toggle.add_theme_color_override("font_hover_color", HERA_ICE)
	_ui_juicy_mode_toggle.toggled.connect(_on_ui_juicy_mode_toggled)
	settings_row.add_child(_ui_juicy_mode_toggle)

	main_screen.add_child(_main_panel)
	_make_visible(false)

func _set_main_status(text: String, connected: bool = false) -> void:
	var status_color := HERA_TERMINAL_GREEN if connected else HERA_OFFLINE_RED
	if _main_status_label != null:
		_main_status_label.text = text
		_main_status_label.add_theme_color_override("font_color", status_color)
	if _main_status_dot != null:
		_main_status_dot.add_theme_stylebox_override("panel", _make_stylebox(status_color, status_color, 0, 6))

func _get_ui_juicy_mode_enabled() -> bool:
	var settings: EditorSettings = EditorInterface.get_editor_settings()
	if not settings.has_setting(UI_JUICY_MODE_SETTING):
		settings.set_setting(UI_JUICY_MODE_SETTING, false)
		settings.mark_setting_changed(UI_JUICY_MODE_SETTING)
	return bool(settings.get_setting(UI_JUICY_MODE_SETTING))

func _on_ui_juicy_mode_toggled(enabled: bool) -> void:
	var settings: EditorSettings = EditorInterface.get_editor_settings()
	settings.set_setting(UI_JUICY_MODE_SETTING, enabled)
	settings.mark_setting_changed(UI_JUICY_MODE_SETTING)

func _make_card() -> PanelContainer:
	var card := PanelContainer.new()
	card.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	card.add_theme_constant_override("margin_left", 14)
	card.add_theme_constant_override("margin_top", 12)
	card.add_theme_constant_override("margin_right", 14)
	card.add_theme_constant_override("margin_bottom", 12)
	card.add_theme_stylebox_override("panel", _make_stylebox(HERA_NIGHT_PANEL, Color(HERA_GODOT_BLUE.r, HERA_GODOT_BLUE.g, HERA_GODOT_BLUE.b, 0.46), 1, 8))
	return card

func _make_logo_texture() -> TextureRect:
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

func _load_display_font(path: String) -> Font:
	var base_font := ResourceLoader.load(path, "Font") as Font
	if base_font == null:
		return null
	var display_font := FontVariation.new()
	display_font.base_font = base_font
	display_font.variation_embolden = HERA_TITLE_FONT_EMBOLDEN
	return display_font

func _make_pill(text: String, accent: Color) -> Label:
	var label := Label.new()
	label.text = text
	label.add_theme_color_override("font_color", accent)
	label.add_theme_font_size_override("font_size", 12)
	label.add_theme_stylebox_override("normal", _make_stylebox(Color(accent.r, accent.g, accent.b, 0.10), Color(accent.r, accent.g, accent.b, 0.34), 1, 7))
	return label

func _make_stylebox(bg: Color, border: Color, border_width: int, radius: int) -> StyleBoxFlat:
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
