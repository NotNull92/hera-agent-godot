extends RefCounted

const ToolResponse = preload("res://addons/hera_agent_godot/core/tool_response.gd")

const INSTANCE_DIR := "user://hera_game_instances"
const REQUEST_ROOT := "user://hera_game_requests"
const RESPONSE_ROOT := "user://hera_game_responses"
const POLL_INTERVAL_SEC := 0.05
const TIMEOUT_SEC := 3.0
const FRESHNESS_SEC := 5.0

var _host: Node
var _next_request_seq := 0

func set_host(host: Node) -> void:
	_host = host

func get_name() -> String:
	return "game"

func execute(params: Dictionary) -> Dictionary:
	return ToolResponse.failure("game requires async dispatch")

func execute_async(params: Dictionary) -> Dictionary:
	if _host == null:
		return ToolResponse.failure("game host not set")
	var action := String(params.get("action", ""))
	if action == "instances":
		return ToolResponse.success(_instances_payload())
	var target := _select_target(_game_instances(), params, EditorInterface.get_playing_scene(), EditorInterface.is_playing_scene())
	if target.has("error"):
		return ToolResponse.failure(String(target["error"]))
	var request_id := _new_request_id()
	var request := params.duplicate()
	request["id"] = request_id
	request["target_pid"] = int(target["pid"])
	request["target_scene"] = String(target["scene"])
	var write_err := _write_request(request, int(target["pid"]))
	if write_err != "":
		return ToolResponse.failure(write_err)
	var deadline := Time.get_ticks_msec() + int(TIMEOUT_SEC * 1000.0)
	while Time.get_ticks_msec() < deadline:
		var response := _read_response(int(target["pid"]), request_id)
		if not response.is_empty():
			if bool(response.get("ok", false)):
				response.erase("ok")
				response.erase("id")
				if params.has("pid"):
					response["game_pid"] = int(target["pid"])
					response["game_scene"] = String(target["scene"])
				if action == "ui_audit":
					response["ok"] = bool(response.get("passed", false))
					response.erase("passed")
				return ToolResponse.success(response)
			return ToolResponse.failure(String(response.get("error", "game request failed")))
		await _host.get_tree().create_timer(POLL_INTERVAL_SEC).timeout
	return ToolResponse.failure("game request timed out; ensure HeraGameInspector autoload is active")

func _new_request_id() -> String:
	_next_request_seq += 1
	return "%d_%d" % [Time.get_ticks_usec(), _next_request_seq]

func _write_request(request: Dictionary, game_pid: int) -> String:
	var request_id := String(request.get("id", ""))
	if request_id == "":
		return "game request id is missing"
	var dir_err := _ensure_dirs(game_pid)
	if dir_err != "":
		return dir_err
	var response_path := _response_path(game_pid, request_id)
	if FileAccess.file_exists(response_path):
		DirAccess.remove_absolute(ProjectSettings.globalize_path(response_path))
	var file := FileAccess.open(_request_path(game_pid, request_id), FileAccess.WRITE)
	if file == null:
		return "could not write game request"
	file.store_string(JSON.stringify(request))
	file.close()
	return ""

func _read_response(game_pid: int, request_id: String) -> Dictionary:
	var response_path := _response_path(game_pid, request_id)
	if not FileAccess.file_exists(response_path):
		return {}
	var file := FileAccess.open(response_path, FileAccess.READ)
	if file == null:
		return {}
	var text := file.get_as_text()
	file.close()
	# The response may still be being written; a failed parse just means poll
	# again, so use the quiet parser rather than the printing one.
	var json := JSON.new()
	if json.parse(text) != OK:
		return {}
	var decoded: Variant = json.data
	if typeof(decoded) != TYPE_DICTIONARY:
		return {}
	if String(decoded.get("id", "")) != request_id:
		return {}
	DirAccess.remove_absolute(ProjectSettings.globalize_path(response_path))
	return decoded

func _ensure_dirs(game_pid: int) -> String:
	var request_err := DirAccess.make_dir_recursive_absolute(ProjectSettings.globalize_path(_request_dir(game_pid)))
	if request_err != OK:
		return "could not create game request directory"
	var response_err := DirAccess.make_dir_recursive_absolute(ProjectSettings.globalize_path(_response_dir(game_pid)))
	if response_err != OK:
		return "could not create game response directory"
	return ""

func _request_dir(game_pid: int) -> String:
	return "%s/%d" % [REQUEST_ROOT, game_pid]

func _response_dir(game_pid: int) -> String:
	return "%s/%d" % [RESPONSE_ROOT, game_pid]

func _request_path(game_pid: int, request_id: String) -> String:
	return "%s/%s.json" % [_request_dir(game_pid), request_id]

func _response_path(game_pid: int, request_id: String) -> String:
	return "%s/%s.json" % [_response_dir(game_pid), request_id]

func _select_target(instances: Array, params: Dictionary, scene: String, editor_playing: bool) -> Dictionary:
	if params.has("pid"):
		var raw_pid: Variant = params["pid"]
		if (typeof(raw_pid) != TYPE_INT and typeof(raw_pid) != TYPE_FLOAT) or float(raw_pid) != floorf(float(raw_pid)) or int(raw_pid) <= 0:
			return { "error": "game pid must be a positive integer" }
		var requested_pid := int(raw_pid)
		for instance in instances:
			if int(instance.get("pid", 0)) == requested_pid:
				return instance
		return { "error": "no live Hera game process found for pid %d" % requested_pid }
	if not editor_playing:
		return { "error": "no game is running; start one with `hera run --current --wait` or pass --pid" }
	var matches := []
	for inst in instances:
		if scene == "" or String(inst.get("scene", "")) == scene:
			matches.append(inst)
	if matches.is_empty():
		return { "error": "no Hera game process found for scene %s; wait a moment or restart the play session" % scene }
	if matches.size() > 1:
		return { "error": "multiple Hera game processes found for scene %s (%s); pass --pid" % [scene, _pids(matches)] }
	return matches[0]

func _instances_payload() -> Dictionary:
	var records := _collect_game_heartbeats()
	var live: Array = records["live"]
	_mark_shared_user_data(live)
	var data := { "instances": live }
	var stale: Array = records["stale"]
	if not stale.is_empty():
		data["stale"] = stale
	return data

func _game_instances() -> Array:
	return _collect_game_heartbeats()["live"]

func _collect_game_heartbeats() -> Dictionary:
	var live := []
	var stale := []
	var dir := DirAccess.open(INSTANCE_DIR)
	if dir == null:
		return { "live": live, "stale": stale }
	var now := Time.get_unix_time_from_system()
	dir.list_dir_begin()
	var file_name := dir.get_next()
	while file_name != "":
		if not dir.current_is_dir() and file_name.ends_with(".json"):
			var inst := _read_instance("%s/%s" % [INSTANCE_DIR, file_name])
			if not inst.is_empty():
				var age := now - float(inst.get("ts", 0.0))
				if age <= FRESHNESS_SEC:
					live.append(inst)
				else:
					inst["age_sec"] = int(age)
					stale.append(inst)
		file_name = dir.get_next()
	dir.list_dir_end()
	live.sort_custom(func(a: Dictionary, b: Dictionary) -> bool: return float(a.get("ts", 0.0)) > float(b.get("ts", 0.0)))
	stale.sort_custom(func(a: Dictionary, b: Dictionary) -> bool: return float(a.get("ts", 0.0)) > float(b.get("ts", 0.0)))
	return { "live": live, "stale": stale }

func _mark_shared_user_data(instances: Array) -> void:
	var counts := {}
	for inst in instances:
		var dir := String(inst.get("user_data_dir", ""))
		if dir == "":
			continue
		counts[dir] = int(counts.get(dir, 0)) + 1
	for i in range(instances.size()):
		var inst: Dictionary = instances[i]
		var dir := String(inst.get("user_data_dir", ""))
		if dir != "" and int(counts.get(dir, 0)) > 1:
			inst["shared_user_data"] = true
			instances[i] = inst

func _read_instance(path: String) -> Dictionary:
	var file := FileAccess.open(path, FileAccess.READ)
	if file == null:
		return {}
	var text := file.get_as_text()
	file.close()
	# Same race as the response file: a heartbeat being rewritten reads short.
	var json := JSON.new()
	if json.parse(text) != OK:
		return {}
	var decoded: Variant = json.data
	if typeof(decoded) != TYPE_DICTIONARY:
		return {}
	return decoded

func _pids(instances: Array) -> String:
	var values: Array[String] = []
	for inst in instances:
		values.append("pid %d" % int(inst.get("pid", 0)))
	return ", ".join(values)
