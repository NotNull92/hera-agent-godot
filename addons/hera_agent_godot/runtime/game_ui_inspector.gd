extends RefCounted

const DEFAULT_FIELDS: Array[String] = ["path", "type", "name", "visible", "rect", "text", "disabled", "pressed"]
const ALLOWED_FIELDS: Dictionary = {
	"path": true,
	"type": true,
	"name": true,
	"visible": true,
	"rect": true,
	"text": true,
	"disabled": true,
	"pressed": true,
}

static func tree(root: Node, current_scene: Node, max_nodes: int, request: Dictionary) -> Dictionary:
	var start := _tree_start(root, current_scene, request)
	if start == null:
		return { "ok": false, "error": "node not found: %s" % String(request.get("path", "")) }
	var fields := _fields_from_request(request)
	if not bool(fields.get("ok", false)):
		return fields
	var max_depth := -1
	if request.has("depth"):
		max_depth = int(request.get("depth", -1))
		if max_depth < 0:
			return { "ok": false, "error": "depth must be non-negative" }
	var controls: Array = []
	_collect_ui(start, controls, max_nodes, request, fields.get("fields", DEFAULT_FIELDS), max_depth, 0)
	var truncated := controls.size() > max_nodes
	if truncated:
		controls = controls.slice(0, max_nodes)
	return {
		"ok": true,
		"count": controls.size(),
		"truncated": truncated,
		"controls": controls,
	}

static func click_target(root: Node, current_scene: Node, request: Dictionary) -> Dictionary:
	var coordinate_mode := request.has("x") or request.has("y")
	var path_mode := request.has("path")
	var text_mode := request.has("text")
	var mode_count := 0
	if coordinate_mode:
		mode_count += 1
	if path_mode:
		mode_count += 1
	if text_mode:
		mode_count += 1
	if mode_count != 1:
		return { "ok": false, "error": "click requires exactly one target: x/y, path, or text" }
	if coordinate_mode:
		if not request.has("x") or not request.has("y"):
			return { "ok": false, "error": "click coordinates require x and y" }
		return { "ok": true, "position": Vector2(float(request.get("x", 0)), float(request.get("y", 0))) }
	if path_mode:
		return _click_target_by_path(root, current_scene, String(request.get("path", "")))
	return _click_target_by_text(root, String(request.get("text", "")))

static func _click_target_by_path(root: Node, current_scene: Node, path: String) -> Dictionary:
	var node := _resolve(root, current_scene, path)
	if node == null:
		return { "ok": false, "error": "node not found: %s" % path }
	if not (node is Control):
		return { "ok": false, "error": "click target is not a Control: %s" % path }
	var control: Control = node as Control
	var control_position := _click_position_for_control(control)
	if not bool(control_position.get("ok", false)):
		return control_position
	control_position["path"] = String(control.get_path())
	return control_position

static func _click_target_by_text(root: Node, text: String) -> Dictionary:
	if text == "":
		return { "ok": false, "error": "click text target is empty" }
	var control := _find_control_by_text(root, text)
	if control == null:
		return { "ok": false, "error": "control with text not found: %s" % text }
	var control_position := _click_position_for_control(control)
	if not bool(control_position.get("ok", false)):
		return control_position
	control_position["path"] = String(control.get_path())
	control_position["text"] = text
	return control_position

static func _click_position_for_control(control: Control) -> Dictionary:
	if not control.is_visible_in_tree():
		return { "ok": false, "error": "click target is not visible: %s" % String(control.get_path()) }
	if control is BaseButton:
		var button: BaseButton = control as BaseButton
		if button.disabled:
			return { "ok": false, "error": "click target is disabled: %s" % String(control.get_path()) }
	var rect := control.get_global_rect()
	if rect.size.x <= 0.0 or rect.size.y <= 0.0:
		return { "ok": false, "error": "click target has an empty rect: %s" % String(control.get_path()) }
	return { "ok": true, "position": rect.position + rect.size * 0.5 }

static func _collect_ui(node: Node, out: Array, max_nodes: int, request: Dictionary, fields: Array, max_depth: int, depth: int) -> void:
	if out.size() > max_nodes:
		return
	if node is Control:
		var control: Control = node as Control
		if _control_matches(control, request):
			out.append(_control_summary(control, fields))
	if max_depth >= 0 and depth >= max_depth:
		return
	for child in node.get_children():
		_collect_ui(child, out, max_nodes, request, fields, max_depth, depth + 1)
		if out.size() > max_nodes:
			return

static func _control_summary(control: Control, fields: Array) -> Dictionary:
	var rect := control.get_global_rect()
	var item := {}
	if fields.has("path"):
		item["path"] = String(control.get_path())
	if fields.has("type"):
		item["type"] = control.get_class()
	if fields.has("name"):
		item["name"] = String(control.name)
	if fields.has("visible"):
		item["visible"] = control.is_visible_in_tree()
	if fields.has("rect"):
		item["rect"] = {
			"x": int(rect.position.x),
			"y": int(rect.position.y),
			"width": int(rect.size.x),
			"height": int(rect.size.y),
		}
	var text := _control_text(control)
	if fields.has("text") and text != "":
		item["text"] = text
	if control is BaseButton:
		var button: BaseButton = control as BaseButton
		if fields.has("disabled"):
			item["disabled"] = button.disabled
		if fields.has("pressed"):
			item["pressed"] = button.button_pressed
	return item

static func _control_matches(control: Control, request: Dictionary) -> bool:
	var type_filter := String(request.get("type", ""))
	if type_filter != "" and not control.is_class(type_filter):
		return false
	var text_filter := String(request.get("text", ""))
	if text_filter != "" and _control_text(control) != text_filter:
		return false
	return true

static func _tree_start(root: Node, current_scene: Node, request: Dictionary) -> Node:
	var path := String(request.get("path", ""))
	if path == "":
		return root
	return _resolve(root, current_scene, path)

static func _fields_from_request(request: Dictionary) -> Dictionary:
	if not request.has("fields"):
		return { "ok": true, "fields": DEFAULT_FIELDS }
	var raw_fields: Variant = request.get("fields", [])
	if typeof(raw_fields) != TYPE_ARRAY:
		return { "ok": false, "error": "fields must be an array" }
	var fields: Array[String] = []
	for raw_field in raw_fields:
		var field := String(raw_field)
		if not ALLOWED_FIELDS.has(field):
			return { "ok": false, "error": "unknown ui field: %s" % field }
		fields.append(field)
	return { "ok": true, "fields": fields }

static func _find_control_by_text(node: Node, text: String) -> Control:
	if node is Control:
		var control: Control = node as Control
		if control.is_visible_in_tree() and _control_text(control) == text:
			return control
	for child in node.get_children():
		var found := _find_control_by_text(child, text)
		if found != null:
			return found
	return null

static func _control_text(control: Control) -> String:
	if not _has_property(control, "text"):
		return ""
	return String(control.get("text"))

static func _has_property(object: Object, prop: String) -> bool:
	for p in object.get_property_list():
		if String(p.get("name", "")) == prop:
			return true
	return false

static func _resolve(root: Node, current_scene: Node, path: String) -> Node:
	if path.begins_with("/"):
		return root.get_node_or_null(NodePath(path))
	return current_scene.get_node_or_null(path) if current_scene != null else null
