extends RefCounted

const ToolResponse = preload("res://addons/hera_agent_godot/core/tool_response.gd")
const HeraSettings = preload("res://addons/hera_agent_godot/core/hera_settings.gd")


func get_name() -> String:
	return "guidance"


func execute(params: Dictionary) -> Dictionary:
	var action := String(params.get("action", "ui"))
	match action:
		"ui":
			return _ui_guidance()
		"game_feel":
			return _game_feel_guidance()
		_:
			return ToolResponse.failure("unknown guidance action: %s (want ui|game_feel)" % action)


func _ui_guidance() -> Dictionary:
	var game_feel_enabled := HeraSettings.get_game_feel_ui_mode_enabled()
	var mode := "game_feel" if game_feel_enabled else "standard"
	return ToolResponse.success({
		"mode": mode,
		"game_feel_ui_mode": game_feel_enabled,
		"setting": HeraSettings.GAME_FEEL_UI_MODE_SETTING,
		"instruction": _instruction(game_feel_enabled),
		"checklist": _checklist(game_feel_enabled),
	})


func _game_feel_guidance() -> Dictionary:
	var enabled := HeraSettings.get_game_feel_mode_enabled()
	var mode := "game_feel" if enabled else "standard"
	return ToolResponse.success({
		"mode": mode,
		"game_feel_mode": enabled,
		"setting": HeraSettings.GAME_FEEL_MODE_SETTING,
		"instruction": _game_feel_instruction(enabled),
		"checklist": _game_feel_checklist(enabled),
		"topics": ["ethics_checklist", "control_feel", "screen_shake", "hit_stop", "camera", "sound", "particles", "tweening_easing"],
		"game_qa_patterns": _game_feel_qa_patterns(enabled),
	})


func _instruction(enabled: bool) -> String:
	if enabled:
		return "Build UI with Game Feel: immediate feedback, crisp transitions, expressive state changes, satisfying input response, and runtime visual QA."
	return "Build UI with the standard Hera guidance: clear layout, readable state, predictable controls, and runtime visual QA."


func _checklist(enabled: bool) -> Array[String]:
	if enabled:
		return [
			"Give every primary input immediate pressed/hover/disabled feedback.",
			"Use brief motion, squash, pulse, flash, or count-up feedback where it clarifies state.",
			"Make success, failure, damage, reward, and progress changes visibly satisfying.",
			"Keep framed children inside local padding; compare sibling rects and multiline minimum sizes, then verify text and disabled-state contrast.",
			"Give paired sibling panels one shared outer rectangle; compare resolved live bounds and enforce content-density budgets.",
			"Use semantic bounded child visuals for tokens/markers; keep interactive frames stable.",
			"Derive grid insets from frame size, cell count, and gaps instead of guessed offsets.",
			"Use live viewport bounds for backgrounds/playfields/HUDs unless fixed resolution is explicit.",
			"Leave inspection handoff states stable, representative, and visibly expressive.",
			"Keep motion bounded and verify it does not cause clipping or layout shift.",
			"Capture runtime UI tree and screenshot analysis after interaction.",
		]
	return [
		"Keep layout readable and stable.",
		"Expose clear state labels and reachable controls.",
		"Verify Control rects through runtime UI tree.",
		"Capture screenshot analysis for visual changes.",
	]


func _game_feel_instruction(enabled: bool) -> String:
	if enabled:
		return "Build gameplay feel with concrete Game Feel parameters: responsive controls, honest juice, camera/audio/particle feedback, accessibility options, and runtime QA."
	return "Build gameplay normally; query game_feel topics when you need concrete feel parameters."


func _game_feel_checklist(enabled: bool) -> Array[String]:
	if enabled:
		return [
			"Start from `game_feel` topic lookup before tuning control, camera, impact, sound, particles, rewards, or presentation.",
			"Keep feedback proportional to real achievement: Honest Juice first.",
			"Add reduce-motion, shake/flash intensity, or off options for strong feedback.",
			"Verify the feel in Play Mode with runtime tree, interactions, screenshot analysis, and output errors.",
			"Before reporting done, validate against `game_feel ethics_checklist` and `game_feel checklist_all`.",
		]
	return [
		"Use `game_feel list` to discover concrete topics when gameplay feel matters.",
		"Run the game and observe the player-facing result before calling work done.",
	]


func _game_feel_qa_patterns(enabled: bool) -> Array[String]:
	if not enabled:
		return []
	return [
		"Scope: `guidance game-feel`=gameplay; `guidance ui`=Control layout/input.",
		"Node2D HUD: root Control uses `Control.MOUSE_FILTER_IGNORE`; keep buttons/sidebar interactive.",
		"Realtime/physics: expose restart/start, deterministic step, and targeted event helpers for score/removal/damage/lives.",
		"Delayed/locked state: trigger+step helpers; assert lock flags, visible state, disabled control counts.",
		"Hidden-state rules: avoid preconditions created by earlier QA steps or reset them explicitly.",
		"Runtime gates: affected-scene load on empty registration; current project helper discovery; typed-copy Dictionary/Variant arrays; build packed collections at runtime instead of in const initializers.",
		"AI/automated turns: expose priority setup helpers and document/prove the undo boundary.",
		"Automated/temporary locks: gate actions at one input boundary with explicit feedback; do not bulk-toggle dense control collections unless unavailability is persistent.",
		"Autonomous loops: provide a restart-paused helper and one-step advancement; freeze exact assertions or combine setup+action atomically, and use tolerances only for intentional real-time checks.",
		"Inspection/visual feel: stable handoff; smoothed path/lane samples are baked once and drive movement+drawing; style resources stay out of frame loops; expose target/channels/duration/intensity.",
		"Collision/impact: forced-overlap helpers prove collision, feedback, end-state, and feel evidence.",
		"Wave/economy checks: isolate placement, spawn, reward, spend, leak, and loss; avoid full-run-only QA.",
		"Large synthetic states: split QA-only state construction, rule evaluation, and render refresh; cover requirements on the bounded rule step.",
		"Runtime QA order: state-changing runtime QA is ordered; do not parallelize clicks, input, or `qa_*` calls.",
		"Primary input scheme: drive keyboard/mouse/touch/controller through `game input`, then assert a gameplay state delta; injection or input-log evidence alone is insufficient.",
		"Stateful controls: read current text or use stable paths before semantic clicks on toggles/modes.",
		"Visible selectors: programmatic state/config changes must update the user-facing selector or label in the same transaction.",
		"Terminal states: preserve terminal-state instruction text and normalize every dependent control label/enabled state in the same terminal, modal, automated, or restart transition.",
		"Viewport layout: draw from the live viewport; keep playfield/HUD rects inside padded bounds.",
		"Game Feel evidence: expose target, channel list, duration, intensity/scope, and at least two observable channels; preserve the last event snapshot after active effects expire.",
		"High-volume UI: scope `game ui tree` with `--type`, `--fields`, `--path`, `--text`, and `--depth`.",
	]
