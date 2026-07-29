extends RefCounted

const GameUIAuditChecks = preload("res://addons/hera_agent_godot/runtime/game_ui_audit_checks.gd")

const DEFAULT_LIMIT := 50
const MAX_LIMIT := 500


static func audit(root: Node, current_scene: Node, max_controls: int, request: Dictionary) -> Dictionary:
	var options: Dictionary = _parse_options(request)
	if not bool(options.get("ok", false)):
		return options
	var start: Node = _resolve_start(root, current_scene, String(options.get("path", "")))
	if start == null:
		return { "ok": false, "error": "node not found: %s" % String(options.get("path", "")) }
	var controls: Array[Control] = []
	_collect_controls(start, controls, max_controls)
	if controls.size() > max_controls:
		return {
			"ok": false,
			"error": "audit incomplete: control limit exceeded; scope with --path",
		}

	var state: Dictionary = _new_state(int(options.get("limit", DEFAULT_LIMIT)))
	var interactive_by_parent: Dictionary = {}
	var blocker_candidates: Array[Control] = []
	for control in controls:
		if not control.is_visible_in_tree():
			continue
		GameUIAuditChecks.check_minimum_size_fit(control, state, options)
		if GameUIAuditChecks.is_interactive(control):
			GameUIAuditChecks.add_interactive_by_parent(control, interactive_by_parent)
			GameUIAuditChecks.check_interactive_rect(control, state, options)
		elif GameUIAuditChecks.is_fullscreen_blocker(control):
			blocker_candidates.append(control)
	GameUIAuditChecks.check_sibling_overlaps(interactive_by_parent, state, options)
	GameUIAuditChecks.check_fullscreen_blockers(blocker_candidates, state, options)
	return _result(start, controls.size(), state, options)


static func _parse_options(request: Dictionary) -> Dictionary:
	var severity := String(request.get("severity", "all"))
	if severity not in ["all", "error", "warning"]:
		return { "ok": false, "error": "severity must be all, error, or warning" }
	var rule := String(request.get("rule", ""))
	if rule != "" and not GameUIAuditChecks.RULES.has(rule):
		return { "ok": false, "error": "unknown UI audit rule: %s" % rule }
	var limit := int(request.get("limit", DEFAULT_LIMIT))
	if limit < 1 or limit > MAX_LIMIT:
		return { "ok": false, "error": "limit must be between 1 and %d" % MAX_LIMIT }
	return {
		"ok": true,
		"path": String(request.get("path", "")),
		"severity": severity,
		"rule": rule,
		"strict": bool(request.get("strict", false)),
		"limit": limit,
	}


static func _resolve_start(root: Node, current_scene: Node, path: String) -> Node:
	if path == "":
		return current_scene
	if path.begins_with("/"):
		return root.get_node_or_null(NodePath(path))
	return current_scene.get_node_or_null(path) if current_scene != null else null


static func _collect_controls(node: Node, out: Array[Control], max_controls: int) -> void:
	if out.size() > max_controls:
		return
	if node is Control:
		out.append(node as Control)
	for child in node.get_children():
		_collect_controls(child, out, max_controls)
		if out.size() > max_controls:
			return


static func _new_state(limit: int) -> Dictionary:
	return {
		"limit": limit,
		"errors": 0,
		"warnings": 0,
		"error_findings": [],
		"warning_findings": [],
	}


static func _result(start: Node, control_count: int, state: Dictionary, options: Dictionary) -> Dictionary:
	var errors := int(state.get("errors", 0))
	var warnings := int(state.get("warnings", 0))
	var limit := int(state.get("limit", DEFAULT_LIMIT))
	var findings: Array = []
	var error_findings: Array = state.get("error_findings", [])
	var warning_findings: Array = state.get("warning_findings", [])
	for finding in error_findings:
		if findings.size() >= limit:
			break
		findings.append(finding)
	for finding in warning_findings:
		if findings.size() >= limit:
			break
		findings.append(finding)
	var strict := bool(options.get("strict", false))
	return {
		"ok": true,
		"passed": errors == 0 and (not strict or warnings == 0),
		"strict": strict,
		"scope": String(start.get_path()),
		"controls": control_count,
		"errors": errors,
		"warnings": warnings,
		"findings": findings,
		"truncated": errors + warnings > findings.size(),
	}
