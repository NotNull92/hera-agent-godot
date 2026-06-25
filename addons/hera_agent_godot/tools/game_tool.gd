extends RefCounted

const ToolResponse = preload("res://addons/hera_agent_godot/core/tool_response.gd")

const REQUEST_DIR := "user://hera_game_requests"
const RESPONSE_DIR := "user://hera_game_responses"
const POLL_INTERVAL_SEC := 0.05
const TIMEOUT_SEC := 3.0

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
	if not EditorInterface.is_playing_scene():
		return ToolResponse.failure("no game is running; start one with `hera run --current --wait`")
	var request_id := _new_request_id()
	var request := params.duplicate()
	request["id"] = request_id
	var write_err := _write_request(request)
	if write_err != "":
		return ToolResponse.failure(write_err)
	var deadline := Time.get_ticks_msec() + int(TIMEOUT_SEC * 1000.0)
	while Time.get_ticks_msec() < deadline:
		var response := _read_response(request_id)
		if not response.is_empty():
			if bool(response.get("ok", false)):
				response.erase("ok")
				response.erase("id")
				return ToolResponse.success(response)
			return ToolResponse.failure(String(response.get("error", "game request failed")))
		await _host.get_tree().create_timer(POLL_INTERVAL_SEC).timeout
	return ToolResponse.failure("game request timed out; ensure HeraGameInspector autoload is active")

func _new_request_id() -> String:
	_next_request_seq += 1
	return "%d_%d" % [Time.get_ticks_usec(), _next_request_seq]

func _write_request(request: Dictionary) -> String:
	var request_id := String(request.get("id", ""))
	if request_id == "":
		return "game request id is missing"
	var dir_err := _ensure_dirs()
	if dir_err != "":
		return dir_err
	var response_path := _response_path(request_id)
	if FileAccess.file_exists(response_path):
		DirAccess.remove_absolute(ProjectSettings.globalize_path(response_path))
	var file := FileAccess.open(_request_path(request_id), FileAccess.WRITE)
	if file == null:
		return "could not write game request"
	file.store_string(JSON.stringify(request))
	file.close()
	return ""

func _read_response(request_id: String) -> Dictionary:
	var response_path := _response_path(request_id)
	if not FileAccess.file_exists(response_path):
		return {}
	var file := FileAccess.open(response_path, FileAccess.READ)
	if file == null:
		return {}
	var text := file.get_as_text()
	file.close()
	var decoded: Variant = JSON.parse_string(text)
	if typeof(decoded) != TYPE_DICTIONARY:
		return {}
	if String(decoded.get("id", "")) != request_id:
		return {}
	DirAccess.remove_absolute(ProjectSettings.globalize_path(response_path))
	return decoded

func _ensure_dirs() -> String:
	var request_err := DirAccess.make_dir_recursive_absolute(ProjectSettings.globalize_path(REQUEST_DIR))
	if request_err != OK:
		return "could not create game request directory"
	var response_err := DirAccess.make_dir_recursive_absolute(ProjectSettings.globalize_path(RESPONSE_DIR))
	if response_err != OK:
		return "could not create game response directory"
	return ""

func _request_path(request_id: String) -> String:
	return "%s/%s.json" % [REQUEST_DIR, request_id]

func _response_path(request_id: String) -> String:
	return "%s/%s.json" % [RESPONSE_DIR, request_id]
