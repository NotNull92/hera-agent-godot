extends RefCounted

const ToolResponse = preload("res://addons/hera_agent_godot/core/tool_response.gd")

const DATA_PATH := "res://addons/hera_agent_godot/data/game_feel_1.0.jsonl"
const CATEGORY_ORDER := ["ethics", "theory", "technique", "ui", "workflow", "anti_pattern", "checklist"]

var _entries: Dictionary = {}
var _keys: Array[String] = []
var _load_error := ""
var _loaded := false


func get_name() -> String:
	return "game_feel"


func execute(params: Dictionary) -> Dictionary:
	var topic := _topic_from_params(params)
	_ensure_loaded()
	if _entries.is_empty():
		return ToolResponse.failure("bundled game-feel data is unavailable: %s" % _load_error)
	if topic == "" or topic == "list":
		return ToolResponse.success({
			"count": _entries.size(),
			"topics": _build_index(),
		})
	var entry: Dictionary = _entries.get(topic, {})
	if not entry.is_empty():
		return ToolResponse.success({
			"key": String(entry.get("key", "")),
			"category": String(entry.get("category", "")),
			"title": String(entry.get("title", "")),
			"body": String(entry.get("body", "")),
		})
	var suggestions := _suggest_similar(topic)
	return ToolResponse.failure("no game-feel topic matches '%s'; suggestions: %s" % [topic, ", ".join(suggestions)])


func _topic_from_params(params: Dictionary) -> String:
	var topic := String(params.get("topic", "")).strip_edges()
	if topic != "":
		return topic
	var args: Variant = params.get("args", [])
	if typeof(args) == TYPE_ARRAY and (args as Array).size() > 0:
		return String((args as Array)[0]).strip_edges()
	return ""


func _ensure_loaded() -> void:
	if _loaded:
		return
	_loaded = true
	var file := FileAccess.open(DATA_PATH, FileAccess.READ)
	if file == null:
		_load_error = error_string(FileAccess.get_open_error())
		return
	while not file.eof_reached():
		var line := file.get_line().strip_edges()
		if line == "":
			continue
		var parsed: Variant = JSON.parse_string(line)
		if typeof(parsed) != TYPE_DICTIONARY:
			continue
		var entry := parsed as Dictionary
		var key := String(entry.get("key", ""))
		if key == "":
			continue
		_entries[key] = entry
		_keys.append(key)


func _build_index() -> Array:
	var grouped := {}
	var extras: Array[String] = []
	for key in _keys:
		var entry: Dictionary = _entries[key]
		var category := String(entry.get("category", "misc"))
		if not grouped.has(category):
			grouped[category] = []
			if not CATEGORY_ORDER.has(category):
				extras.append(category)
		grouped[category].append({
			"key": key,
			"title": String(entry.get("title", "")),
		})
	var ordered := CATEGORY_ORDER.duplicate()
	ordered.append_array(extras)
	var out := []
	for category in ordered:
		if not grouped.has(category):
			continue
		var topics: Array = grouped[category]
		topics.sort_custom(func(a: Dictionary, b: Dictionary) -> bool: return String(a.get("key", "")) < String(b.get("key", "")))
		out.append({
			"category": category,
			"topics": topics,
		})
	return out


func _suggest_similar(query: String, max_distance: int = 3, max_count: int = 5) -> Array[String]:
	var candidates := []
	for key in _keys:
		var distance := _levenshtein_bounded(query, key, max_distance)
		if distance <= max_distance:
			candidates.append({"key": key, "distance": distance})
	candidates.sort_custom(func(a: Dictionary, b: Dictionary) -> bool: return int(a.get("distance", 0)) < int(b.get("distance", 0)))
	var out: Array[String] = []
	for item in candidates:
		out.append(String(item.get("key", "")))
		if out.size() >= max_count:
			break
	return out


func _levenshtein_bounded(a: String, b: String, max_distance: int) -> int:
	if abs(a.length() - b.length()) > max_distance:
		return max_distance + 1
	var previous: Array[int] = []
	for i in range(b.length() + 1):
		previous.append(i)
	for i in range(1, a.length() + 1):
		var current: Array[int] = [i]
		var row_min := i
		for j in range(1, b.length() + 1):
			var cost := 0 if a[i - 1] == b[j - 1] else 1
			var value: int = min(min(previous[j] + 1, current[j - 1] + 1), previous[j - 1] + cost)
			current.append(value)
			row_min = min(row_min, value)
		if row_min > max_distance:
			return max_distance + 1
		previous = current
	return previous[b.length()]
