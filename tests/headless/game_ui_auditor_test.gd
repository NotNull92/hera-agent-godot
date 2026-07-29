extends SceneTree

const GameUIAuditor = preload("res://addons/hera_agent_godot/runtime/game_ui_auditor.gd")

var _failures: Array[String] = []


func _initialize() -> void:
	call_deferred("_run")


func _run() -> void:
	var viewport := SubViewport.new()
	viewport.size = Vector2i(400, 300)
	root.add_child(viewport)
	var fixture := _fixture()
	viewport.add_child(fixture)
	await process_frame

	_expect_all_rules(fixture)
	_expect_filters_limits_and_evidence(fixture)
	_expect_warning_strictness(viewport)
	_expect_clean_scene(viewport)
	_expect_control_limit(viewport)

	viewport.queue_free()
	await process_frame
	if _failures.is_empty():
		quit(0)
		return
	for failure in _failures:
		push_error(failure)
	quit(1)


func _fixture() -> Control:
	var fixture := Control.new()
	fixture.name = "AuditFixture"
	fixture.size = Vector2(400, 300)
	fixture.mouse_filter = Control.MOUSE_FILTER_IGNORE

	var empty_control := Control.new()
	empty_control.name = "EmptyControl"
	empty_control.position = Vector2(10, 10)
	empty_control.size = Vector2.ZERO
	empty_control.focus_mode = Control.FOCUS_ALL
	fixture.add_child(empty_control)

	var outside_button := Button.new()
	outside_button.name = "OutsideButton"
	outside_button.position = Vector2(500, 20)
	outside_button.size = Vector2(80, 30)
	fixture.add_child(outside_button)

	var clip_parent := Control.new()
	clip_parent.name = "ClipParent"
	clip_parent.position = Vector2(20, 70)
	clip_parent.size = Vector2(80, 60)
	clip_parent.clip_contents = true
	clip_parent.mouse_filter = Control.MOUSE_FILTER_IGNORE
	fixture.add_child(clip_parent)

	var clipped_button := Button.new()
	clipped_button.name = "ClippedButton"
	clipped_button.position = Vector2(100, 10)
	clipped_button.size = Vector2(40, 30)
	clip_parent.add_child(clipped_button)

	var blocker := Control.new()
	blocker.name = "Blocker"
	blocker.size = Vector2(400, 300)
	blocker.mouse_filter = Control.MOUSE_FILTER_STOP
	fixture.add_child(blocker)

	var small_parent := Control.new()
	small_parent.name = "SmallParent"
	small_parent.position = Vector2(20, 150)
	small_parent.size = Vector2(20, 10)
	small_parent.mouse_filter = Control.MOUSE_FILTER_IGNORE
	fixture.add_child(small_parent)

	var oversized_minimum := Label.new()
	oversized_minimum.name = "OversizedMinimum"
	oversized_minimum.custom_minimum_size = Vector2(100, 40)
	oversized_minimum.mouse_filter = Control.MOUSE_FILTER_IGNORE
	small_parent.add_child(oversized_minimum)

	var first := Button.new()
	first.name = "First"
	first.position = Vector2(180, 180)
	first.size = Vector2(80, 40)
	fixture.add_child(first)

	var second := Button.new()
	second.name = "Second"
	second.position = Vector2(200, 190)
	second.size = Vector2(80, 40)
	fixture.add_child(second)
	return fixture


func _expect_all_rules(fixture: Control) -> void:
	var result := GameUIAuditor.audit(root, fixture, 1000, {})
	_expect(bool(result.get("ok", false)), "audit request should succeed")
	var rules: Dictionary = {}
	for finding in result.get("findings", []):
		rules[String(finding.get("rule", ""))] = true
	for rule in [
		"empty_interactive_rect",
		"interactive_outside_viewport",
		"fully_clipped_interactive",
		"fullscreen_mouse_blocker",
		"minimum_size_exceeds_parent",
		"overlapping_interactive_siblings",
	]:
		_expect(rules.has(rule), "missing rule: %s" % rule)


func _expect_filters_limits_and_evidence(fixture: Control) -> void:
	var empty_only := GameUIAuditor.audit(root, fixture, 1000, {
		"rule": "empty_interactive_rect",
	})
	_expect(int(empty_only.get("errors", 0)) == 1, "empty rule should report one error")
	_expect(int(empty_only.get("warnings", -1)) == 0, "empty rule should exclude warnings")
	var empty_findings: Array = empty_only.get("findings", [])
	_expect(empty_findings.size() == 1, "empty rule should return one finding")
	if not empty_findings.is_empty():
		var empty_finding: Dictionary = empty_findings[0]
		_expect(
			String(empty_finding.get("path", "")).ends_with("/EmptyControl"),
			"empty rule should identify EmptyControl"
		)

	var minimum_only := GameUIAuditor.audit(root, fixture, 1000, {
		"rule": "minimum_size_exceeds_parent",
		"strict": true,
	})
	_expect(not bool(minimum_only.get("passed", true)), "strict minimum warning should fail")
	var minimum_findings: Array = minimum_only.get("findings", [])
	if not minimum_findings.is_empty():
		var minimum_finding: Dictionary = minimum_findings[0]
		_expect(
			String(minimum_finding.get("related_path", "")).ends_with("/SmallParent"),
			"minimum warning should identify its parent"
		)

	var warnings_only := GameUIAuditor.audit(root, fixture, 1000, {
		"severity": "warning",
	})
	_expect(int(warnings_only.get("errors", -1)) == 0, "warning filter should exclude errors")
	for finding in warnings_only.get("findings", []):
		_expect(String(finding.get("severity", "")) == "warning", "warning filter leaked an error")

	var limited := GameUIAuditor.audit(root, fixture, 1000, {"limit": 1})
	_expect(limited.get("findings", []).size() == 1, "limit should cap returned findings")
	_expect(bool(limited.get("truncated", false)), "limited result should be truncated")
	if not limited.get("findings", []).is_empty():
		_expect(
			String(limited.get("findings", [])[0].get("severity", "")) == "error",
			"errors should be ordered before warnings"
		)


func _expect_warning_strictness(viewport: SubViewport) -> void:
	var fixture := Control.new()
	fixture.name = "WarningFixture"
	fixture.size = Vector2(400, 300)
	fixture.mouse_filter = Control.MOUSE_FILTER_IGNORE
	viewport.add_child(fixture)
	var blocker := Control.new()
	blocker.name = "Blocker"
	blocker.size = Vector2(400, 300)
	blocker.mouse_filter = Control.MOUSE_FILTER_STOP
	fixture.add_child(blocker)
	var normal := GameUIAuditor.audit(root, fixture, 1000, {"rule": "fullscreen_mouse_blocker"})
	var strict := GameUIAuditor.audit(root, fixture, 1000, {
		"rule": "fullscreen_mouse_blocker",
		"strict": true,
	})
	_expect(bool(normal.get("passed", false)), "warning-only audit should pass by default")
	_expect(not bool(strict.get("passed", true)), "strict warning audit should fail")
	fixture.queue_free()


func _expect_clean_scene(viewport: SubViewport) -> void:
	var fixture := Control.new()
	fixture.name = "CleanFixture"
	fixture.size = Vector2(400, 300)
	fixture.mouse_filter = Control.MOUSE_FILTER_IGNORE
	viewport.add_child(fixture)
	var result := GameUIAuditor.audit(root, fixture, 1000, {})
	_expect(bool(result.get("passed", false)), "clean audit should pass")
	_expect(int(result.get("errors", -1)) == 0, "clean audit should have zero errors")
	_expect(int(result.get("warnings", -1)) == 0, "clean audit should have zero warnings")
	fixture.queue_free()


func _expect_control_limit(viewport: SubViewport) -> void:
	var fixture := Control.new()
	fixture.name = "LimitFixture"
	viewport.add_child(fixture)
	fixture.add_child(Control.new())
	fixture.add_child(Control.new())
	var result := GameUIAuditor.audit(root, fixture, 2, {})
	_expect(not bool(result.get("ok", true)), "truncated audit should fail")
	fixture.queue_free()


func _expect(condition: bool, message: String) -> void:
	if not condition:
		_failures.append(message)
