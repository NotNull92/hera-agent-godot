extends RefCounted

const FULLSCREEN_COVERAGE := 0.95
const PIXEL_TOLERANCE := 1.0

const RULE_EMPTY_RECT := "empty_interactive_rect"
const RULE_OUTSIDE_VIEWPORT := "interactive_outside_viewport"
const RULE_FULLY_CLIPPED := "fully_clipped_interactive"
const RULE_FULLSCREEN_BLOCKER := "fullscreen_mouse_blocker"
const RULE_MINIMUM_EXCEEDS_PARENT := "minimum_size_exceeds_parent"
const RULE_SIBLING_OVERLAP := "overlapping_interactive_siblings"

const RULES: Dictionary = {
	RULE_EMPTY_RECT: true,
	RULE_OUTSIDE_VIEWPORT: true,
	RULE_FULLY_CLIPPED: true,
	RULE_FULLSCREEN_BLOCKER: true,
	RULE_MINIMUM_EXCEEDS_PARENT: true,
	RULE_SIBLING_OVERLAP: true,
}


static func is_interactive(control: Control) -> bool:
	if control is BaseButton or control is LineEdit or control is TextEdit:
		return true
	if control is ItemList or control is Tree or control is TabBar:
		return true
	if control is Range and not control is ProgressBar:
		return true
	if control.focus_mode != Control.FOCUS_NONE:
		return true
	return not control.get_signal_connection_list("gui_input").is_empty()


static func check_interactive_rect(
	control: Control,
	state: Dictionary,
	options: Dictionary
) -> void:
	var rect := control.get_global_rect()
	if rect.size.x <= 0.0 or rect.size.y <= 0.0:
		_record(state, options, {
			"rule": RULE_EMPTY_RECT,
			"severity": "error",
			"path": String(control.get_path()),
			"rect": _rect_data(rect),
		})
		return
	if _has_scroll_ancestor(control):
		return
	var viewport_rect := control.get_viewport().get_visible_rect()
	if not rect.intersects(viewport_rect):
		_record(state, options, {
			"rule": RULE_OUTSIDE_VIEWPORT,
			"severity": "error",
			"path": String(control.get_path()),
			"rect": _rect_data(rect),
			"related_rect": _rect_data(viewport_rect),
		})
		return
	var clip_rect: Rect2 = _effective_clip_rect(control, viewport_rect)
	if not rect.intersects(clip_rect):
		_record(state, options, {
			"rule": RULE_FULLY_CLIPPED,
			"severity": "error",
			"path": String(control.get_path()),
			"rect": _rect_data(rect),
			"related_rect": _rect_data(clip_rect),
		})


static func check_minimum_size_fit(
	control: Control,
	state: Dictionary,
	options: Dictionary
) -> void:
	var parent := control.get_parent()
	if not parent is Control or parent is ScrollContainer:
		return
	var parent_control := parent as Control
	var minimum := control.get_combined_minimum_size()
	var axes: Array[String] = []
	if parent_control.size.x + PIXEL_TOLERANCE < minimum.x:
		axes.append("width")
	if parent_control.size.y + PIXEL_TOLERANCE < minimum.y:
		axes.append("height")
	if axes.is_empty():
		return
	_record(state, options, {
		"rule": RULE_MINIMUM_EXCEEDS_PARENT,
		"severity": "warning",
		"path": String(control.get_path()),
		"rect": _rect_data(control.get_global_rect()),
		"related_path": String(parent_control.get_path()),
		"related_rect": _rect_data(parent_control.get_global_rect()),
		"minimum_size": _size_data(minimum),
		"axes": axes,
	})


static func add_interactive_by_parent(control: Control, groups: Dictionary) -> void:
	var parent := control.get_parent()
	if parent == null:
		return
	var key := parent.get_instance_id()
	if not groups.has(key):
		groups[key] = []
	var siblings: Array = groups[key]
	siblings.append(control)


static func check_sibling_overlaps(
	groups: Dictionary,
	state: Dictionary,
	options: Dictionary
) -> void:
	for raw_siblings in groups.values():
		var siblings: Array = raw_siblings
		for first_index in range(siblings.size()):
			var first := siblings[first_index] as Control
			var first_rect := first.get_global_rect()
			for second_index in range(first_index + 1, siblings.size()):
				var second := siblings[second_index] as Control
				var second_rect := second.get_global_rect()
				var overlap := first_rect.intersection(second_rect)
				if overlap.size.x <= PIXEL_TOLERANCE or overlap.size.y <= PIXEL_TOLERANCE:
					continue
				_record(state, options, {
					"rule": RULE_SIBLING_OVERLAP,
					"severity": "warning",
					"path": String(first.get_path()),
					"related_path": String(second.get_path()),
					"rect": _rect_data(first_rect),
					"related_rect": _rect_data(second_rect),
					"intersection": _rect_data(overlap),
				})


static func is_fullscreen_blocker(control: Control) -> bool:
	if control.mouse_filter == Control.MOUSE_FILTER_IGNORE:
		return false
	var rect := control.get_global_rect()
	var viewport_rect := control.get_viewport().get_visible_rect()
	var viewport_area := viewport_rect.size.x * viewport_rect.size.y
	if viewport_area <= 0.0:
		return false
	var overlap := rect.intersection(viewport_rect)
	return overlap.size.x * overlap.size.y / viewport_area >= FULLSCREEN_COVERAGE


static func check_fullscreen_blockers(
	candidates: Array[Control],
	state: Dictionary,
	options: Dictionary
) -> void:
	for candidate in candidates:
		var has_deeper_candidate := false
		for other in candidates:
			if candidate != other and candidate.is_ancestor_of(other):
				has_deeper_candidate = true
				break
		if has_deeper_candidate:
			continue
		_record(state, options, {
			"rule": RULE_FULLSCREEN_BLOCKER,
			"severity": "warning",
			"path": String(candidate.get_path()),
			"rect": _rect_data(candidate.get_global_rect()),
			"related_rect": _rect_data(candidate.get_viewport().get_visible_rect()),
		})


static func _effective_clip_rect(control: Control, viewport_rect: Rect2) -> Rect2:
	var result := viewport_rect
	var ancestor := control.get_parent()
	while ancestor != null and not ancestor is Viewport:
		if ancestor is Control:
			var parent_control := ancestor as Control
			if parent_control.clip_contents:
				result = result.intersection(parent_control.get_global_rect())
		ancestor = ancestor.get_parent()
	return result


static func _has_scroll_ancestor(control: Control) -> bool:
	var ancestor := control.get_parent()
	while ancestor != null and not ancestor is Viewport:
		if ancestor is ScrollContainer:
			return true
		ancestor = ancestor.get_parent()
	return false


static func _record(state: Dictionary, options: Dictionary, finding: Dictionary) -> void:
	var severity := String(finding.get("severity", ""))
	var rule := String(finding.get("rule", ""))
	if String(options.get("severity", "all")) not in ["all", severity]:
		return
	var selected_rule := String(options.get("rule", ""))
	if selected_rule != "" and selected_rule != rule:
		return
	var count_key := "errors" if severity == "error" else "warnings"
	state[count_key] = int(state.get(count_key, 0)) + 1
	var bucket_key := "error_findings" if severity == "error" else "warning_findings"
	var bucket: Array = state[bucket_key]
	if bucket.size() < int(state.get("limit", 50)):
		bucket.append(finding)


static func _rect_data(rect: Rect2) -> Dictionary:
	return {
		"x": int(rect.position.x),
		"y": int(rect.position.y),
		"width": int(rect.size.x),
		"height": int(rect.size.y),
	}


static func _size_data(size: Vector2) -> Dictionary:
	return { "width": int(size.x), "height": int(size.y) }
