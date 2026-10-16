@tool
extends MarginContainer

const PANEL_NAME := "HeraAgentMainScreen"
const TITLE_FONT_PATH := "res://addons/hera_agent_godot/assets/fonts/cormorant-italic.woff2"
const STRINGS := {
	"title": ["Hera", "Hera"],
	"intro": ["I help your agent see and edit this editor.", "에이전트가 이 에디터를 보고 편집하도록 도와줄게요."],
	"engine": ["Godot 4.7+ recommended · 4.2 minimum", "Godot 4.7+ 권장 · 최소 4.2"],
	"status": ["Status", "상태"],
	"connection": ["Connection", "연결"],
	"version": ["Version", "버전"],
	"csharp": ["C#", "C#"],
	"auth": ["Auth", "인증"],
	"auth_unknown": ["Unknown (token read failed)", "알 수 없음 (토큰 읽기 실패)"],
	"versions": ["CLI %s · Addon %s", "CLI %s · 애드온 %s"],
	"starting": ["Starting local bridge…", "로컬 브리지 시작 중…"],
	"listening": ["Listening 127.0.0.1:%d", "연결 대기 127.0.0.1:%d"],
	"token_error": ["Not connected: shared token read failed", "연결 안 됨: 공유 토큰 읽기 실패"],
	"bind_error": ["Not connected: ports 8770–8785 unavailable", "연결 안 됨: 포트 8770–8785 사용 불가"],
	"not_seen": ["No CLI request yet", "아직 CLI 요청 없음"],
	"unknown": ["CLI version unknown; upgrade the CLI", "CLI 버전 알 수 없음; CLI를 업그레이드하세요"],
	"match": ["Versions match", "버전 일치"],
	"mismatch": ["Upgrade CLI and addon together", "CLI와 애드온을 함께 업그레이드하세요"],
	"patch": ["Patch versions differ", "패치 버전 다름"],
	"support": ["Editor: %s · .csproj: %s", "에디터: %s · .csproj: %s"],
	"yes": ["yes", "있음"],
	"no": ["no", "없음"],
	"on": ["On", "켜짐"],
	"off": ["Off", "꺼짐"],
	"copy": ["Copy agent setup", "에이전트 설정 복사"],
	"copied": ["Copied", "복사됨"],
	"feel": ["Game Feel (Beta)", "게임 필 (베타)"],
	"game_mode": ["Game Feel Mode", "게임 필 모드"],
	"game_hint": ["Guide controls, camera, impact, sound and rewards.", "조작, 카메라, 타격감, 소리와 보상을 안내합니다."],
	"ui_mode": ["Game Feel UI Mode", "게임 필 UI 모드"],
	"ui_hint": ["Guide immediate feedback, motion and satisfying interactions.", "즉각적인 피드백, 움직임과 만족스러운 상호작용을 안내합니다."],
}

var rows: Dictionary = {}
var content: VBoxContainer
var copied: Label
var copy_timer: Timer
var addon_version := ""
var editor_port := 0
var auth_on := false
var auth_known := true
var connected := false
var connection_key := "starting"
var _cards: Array[PanelContainer] = []
var _colors: Dictionary = {}

static func words(key: String, inline: bool = true) -> String:
	var pair: Array = STRINGS[key]
	if pair[0] == pair[1]:
		return String(pair[0])
	return "%s %s" % [pair[1], pair[0]] if inline else "%s\n%s" % [pair[0], pair[1]]

static func version_state(cli: String, addon: String, seen: bool) -> String:
	if not seen:
		return "not_seen"
	var pattern := RegEx.new()
	pattern.compile("^v?([0-9]+)\\.([0-9]+)\\.([0-9]+)(?:[-+][0-9A-Za-z.-]+)?$")
	var left := pattern.search(cli)
	var right := pattern.search(addon)
	if left == null or right == null:
		return "unknown"
	if left.get_string(1) != right.get_string(1) or left.get_string(2) != right.get_string(2):
		return "mismatch"
	return "match"

static func create(ui_enabled: bool, game_feel_enabled: bool, ui_callback: Callable, game_feel_callback: Callable) -> Dictionary:
	var main_screen := EditorInterface.get_editor_main_screen()
	var stale := main_screen.get_node_or_null(PANEL_NAME)
	if stale != null:
		main_screen.remove_child(stale)
		stale.queue_free()
	# Dynamic construction keeps editor-only scripts checkable on Godot 4.2.
	var script: Script = load("res://addons/hera_agent_godot/core/main_screen_panel.gd")
	var panel: MarginContainer = script.new()
	panel.name = PANEL_NAME
	panel.size_flags_vertical = Control.SIZE_EXPAND_FILL
	panel.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	main_screen.add_child(panel)
	return panel.build(ui_enabled, game_feel_enabled, ui_callback, game_feel_callback)

func build(ui_enabled: bool, game_feel_enabled: bool, ui_callback: Callable, game_feel_callback: Callable) -> Dictionary:
	var scale := EditorInterface.get_editor_scale()
	for edge: String in ["left", "top", "right", "bottom"]:
		add_theme_constant_override("margin_" + edge, roundi(24 * scale))
	var config := ConfigFile.new()
	if config.load("res://addons/hera_agent_godot/plugin.cfg") == OK:
		addon_version = String(config.get_value("plugin", "version", ""))
	var scroll := ScrollContainer.new()
	scroll.horizontal_scroll_mode = ScrollContainer.SCROLL_MODE_DISABLED
	add_child(scroll)
	content = VBoxContainer.new()
	content.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	content.add_theme_constant_override("separation", roundi(16 * scale))
	scroll.add_child(content)
	var header := VBoxContainer.new()
	header.name = "Header"
	content.add_child(header)
	var branding := HBoxContainer.new()
	header.add_child(branding)
	if FileAccess.file_exists("res://docs/assets/hera-pointing.png"):
		var image: Texture2D = load("res://docs/assets/hera-pointing.png") as Texture2D
		if image != null:
			var logo := TextureRect.new()
			logo.texture = image
			logo.custom_minimum_size = Vector2(64, 64) * scale
			logo.expand_mode = TextureRect.EXPAND_IGNORE_SIZE
			logo.stretch_mode = TextureRect.STRETCH_KEEP_ASPECT_CENTERED
			branding.add_child(logo)
	var title := _label(branding, "title")
	title.text = STRINGS["title"][0]
	title.add_theme_font_size_override("font_size", roundi(38 * scale))
	if ResourceLoader.exists(TITLE_FONT_PATH, "Font"):
		var font := FontVariation.new()
		font.base_font = load(TITLE_FONT_PATH) as Font
		font.variation_embolden = 0.85
		title.add_theme_font_override("font", font)
	_label(header, "intro", false)
	_label(header, "engine", false)
	var status := _card("Status")
	_label(status, "status")
	for key: String in ["connection", "version", "csharp", "auth"]:
		var row := HBoxContainer.new()
		status.add_child(row)
		var caption := _label(row, key)
		caption.custom_minimum_size.x = 140 * scale
		var value := Label.new()
		value.size_flags_horizontal = Control.SIZE_EXPAND_FILL
		value.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
		row.add_child(value)
		rows[key] = value
	var copy_row := HBoxContainer.new()
	copy_row.name = "CopySetup"
	content.add_child(copy_row)
	var button := Button.new()
	button.text = words("copy")
	button.pressed.connect(_copy_setup)
	copy_row.add_child(button)
	copied = _label(copy_row, "copied")
	copied.hide()
	copy_timer = Timer.new()
	copy_timer.one_shot = true
	copy_timer.wait_time = 2.0
	copy_timer.timeout.connect(copied.hide)
	add_child(copy_timer)
	var feel := _card("GameFeel")
	_label(feel, "feel")
	var game_toggle := _toggle(feel, "game_mode", "game_hint", game_feel_enabled, game_feel_callback)
	var ui_toggle := _toggle(feel, "ui_mode", "ui_hint", ui_enabled, ui_callback)
	update_observations("", false, 0, false)
	_repaint()
	return {"panel": self, "ui_toggle": ui_toggle, "game_feel_toggle": game_toggle}

func _label(parent: Node, key: String, inline: bool = true) -> Label:
	var label := Label.new()
	label.text = words(key, inline)
	parent.add_child(label)
	return label

func _card(node_name: String) -> VBoxContainer:
	var card := PanelContainer.new()
	card.name = node_name
	content.add_child(card)
	_cards.append(card)
	var margin := MarginContainer.new()
	for edge: String in ["left", "top", "right", "bottom"]:
		margin.add_theme_constant_override("margin_" + edge, roundi(16 * EditorInterface.get_editor_scale()))
	card.add_child(margin)
	var stack := VBoxContainer.new()
	stack.add_theme_constant_override("separation", roundi(10 * EditorInterface.get_editor_scale()))
	margin.add_child(stack)
	return stack

func _toggle(parent: Node, key: String, hint: String, enabled: bool, callback: Callable) -> CheckButton:
	var button := CheckButton.new()
	button.text = words(key)
	button.tooltip_text = words(hint, false)
	button.button_pressed = enabled
	button.toggled.connect(callback)
	parent.add_child(button)
	return button

func set_connection(key: String, is_connected: bool) -> void:
	connection_key = key
	connected = is_connected
	if key == "token_error":
		auth_known = false
		rows["auth"].text = words("auth_unknown")
	_connection_text()

func _connection_text() -> void:
	var value: Label = rows["connection"]
	value.text = "● " + words(connection_key, false)
	if connection_key == "listening":
		value.text = value.text % [editor_port, editor_port]
	_colors["connection"] = "success_color" if connected else "error_color"
	_repaint()

func update_observations(cli: String, seen: bool, port: int, authentication: bool) -> void:
	editor_port = port
	auth_on = authentication
	auth_known = true
	_connection_text()
	var state := version_state(cli, addon_version, seen)
	var version_label: Label = rows["version"]
	var cli_text := cli if cli != "" else "—"
	version_label.text = "● %s\n%s" % [words(state), words("versions", false) % [cli_text, addon_version, cli_text, addon_version]]
	_colors["version"] = "warning_color" if state == "mismatch" else "font_color"
	if state == "match":
		_colors["version"] = "success_color"
		if cli.trim_prefix("v") != addon_version.trim_prefix("v"):
			version_label.text += "\n" + words("patch")
			_colors["version"] = "font_color"
	var has_project := false
	for file: String in DirAccess.get_files_at("res://"):
		if file.ends_with(".csproj"):
			has_project = true
	var support: Label = rows["csharp"]
	support.text = words("support", false) % [STRINGS["yes" if ClassDB.class_exists("CSharpScript") else "no"][0], STRINGS["yes" if has_project else "no"][0], STRINGS["yes" if ClassDB.class_exists("CSharpScript") else "no"][1], STRINGS["yes" if has_project else "no"][1]]
	var auth: Label = rows["auth"]
	auth.text = words("on" if auth_on else "off")
	_repaint()

func agent_setup_text() -> String:
	var auth_note := "Auth is on; the CLI reads the shared token itself." if auth_on else "Auth is off."
	if not auth_known:
		auth_note = "Auth state is unknown because the shared token could not be read; the bridge is not connected."
	return "\n".join([
		"Project: " + ProjectSettings.globalize_path("res://"),
		"Editor port: %d; addon version: %s" % [editor_port, addon_version],
		"Install CLI if absent: https://github.com/NotNull92/hera-agent-godot#install",
		"Verify: hera-godot status, then hera-godot smoke --skip-game",
		"eval defaults to C#; pass --lang gdscript for GDScript.",
		"C# debugging uses an IDE .NET debugger, not Hera DAP.",
		"Driver card: https://github.com/NotNull92/hera-agent-godot/blob/main/AGENTS.md",
		auth_note,
	])

func _copy_setup() -> void:
	DisplayServer.clipboard_set(agent_setup_text())
	copied.show()
	copy_timer.start()

func _notification(what: int) -> void:
	if what == NOTIFICATION_THEME_CHANGED and is_inside_tree():
		_repaint.call_deferred()

func _repaint() -> void:
	for card: PanelContainer in _cards:
		var style: StyleBox = get_theme_stylebox("panel", "Panel")
		if card.get_theme_stylebox("panel") != style:
			card.add_theme_stylebox_override("panel", style)
	for key: String in _colors:
		var label: Label = rows[key]
		var color := get_theme_color(String(_colors[key]), "Editor")
		if label.get_theme_color("font_color") != color:
			label.add_theme_color_override("font_color", color)
