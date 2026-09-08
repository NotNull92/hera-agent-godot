extends SceneTree

const HttpServer = preload("res://addons/hera_agent_godot/server/http_server.gd")
const WorkQueue = preload("res://addons/hera_agent_godot/server/work_queue.gd")

var _failed := false
var _server := HttpServer.new()
var _queue := WorkQueue.new()

func _initialize() -> void:
	_run.call_deferred()

func _run() -> void:
	if not OS.get_cmdline_user_args().has("--backpressure"):
		_test_tokens()
	_check(_server.start(18770, 100) > 0, "loopback listener starts")
	_server.auth_token = "test-token"
	await _request("", "{}", 401)
	await _request("X-Hera-Token: wrong\r\n", "{}", 401)
	await _request("X-Hera-Token: test-token\r\nOrigin: https://example.com\r\n", "{}", 403)
	await _request("X-Hera-Token: test-token\r\n", "{}", 200)
	await _request("X-Hera-Token: test-token\r\n", "[]", 400)
	await _request("Content-Length: 1048577\r\n", "", 413)
	var slow: StreamPeerTCP = await _connect()
	slow.put_data(_wire("X-Hera-Token: test-token\r\n", "{}"))
	var pending: Dictionary = await _receive_request()
	_check(not pending.is_empty(), "slow client request enqueued")
	_server._clients[0]["deadline"] = 0
	_server.poll(_queue)
	_check(not _server._clients.is_empty(), "receive deadline does not cancel queued asynchronous work")
	print("HTTP_BACKPRESSURE_BEGIN")
	var started := Time.get_ticks_msec()
	_server.respond(pending["conn"], { "ok": true, "data": "x".repeat(32 * 1024 * 1024) })
	_check(Time.get_ticks_msec() - started < 2000, "respond must not block on a nonreading client")
	_check(not _server._clients.is_empty(), "large response remains tracked until written or timed out")
	var ticks := 0
	var stalled := false
	var previous_offset := -1
	while Time.get_ticks_msec() - started < 5500:
		_server.poll(_queue)
		ticks += 1
		if not _server._clients.is_empty():
			var offset: int = _server._clients[0]["offset"]
			stalled = stalled or (offset > 0 and offset == previous_offset)
			previous_offset = offset
		await process_frame
	_check(ticks > 50, "main loop progresses during backpressure")
	_check(stalled, "nonreading loopback client actually causes partial-write backpressure")
	_check(_server._clients.is_empty(), "output timeout releases pending response")
	_server.respond(pending["conn"], { "ok": true })
	_check(_server._clients.is_empty(), "late response cannot resurrect expired connection")
	slow.disconnect_from_host()
	var abandoned: StreamPeerTCP = await _connect()
	abandoned.put_data(_wire("X-Hera-Token: test-token\r\n", "{}"))
	pending = await _receive_request()
	abandoned.disconnect_from_host()
	for i in 10:
		_server.poll(_queue)
		await process_frame
	_server.respond(pending["conn"], { "ok": true })
	_check(_server._clients.is_empty(), "async response after client disconnect cannot leak a connection")
	await _request("X-Hera-Token: test-token\r\n", "{}", 200)
	_server.stop()
	print("HTTP_TEST_PASS" if not _failed else "HTTP_TEST_FAIL")
	quit(1 if _failed else 0)

func _test_tokens() -> void:
	var env_names := ["USERPROFILE", "HOME", "HERA_AGENT_GODOT_TOKEN"]
	var saved: Dictionary = {}
	for key: String in env_names:
		saved[key] = OS.get_environment(key) if OS.has_environment(key) else null
	var home := OS.get_environment("TEMP") if OS.has_environment("TEMP") else "/tmp"
	home = home.path_join("hera-http-token-test-%d" % OS.get_process_id())
	var token_dir := home.path_join(".hera-agent-godot")
	var token_path := token_dir.path_join("token")
	DirAccess.make_dir_recursive_absolute(token_dir)
	OS.set_environment("USERPROFILE", home)
	OS.set_environment("HOME", home)
	OS.unset_environment("HERA_AGENT_GODOT_TOKEN")
	var result: Variant = HttpServer.load_shared_token()
	_check(result is Dictionary and result.get("token") == "", "missing token intentionally disables auth")
	var file := FileAccess.open(token_path, FileAccess.WRITE)
	file.store_string("  file-token\n")
	file.close()
	result = HttpServer.load_shared_token()
	_check(result is Dictionary and result.get("token") == "file-token", "token file is trimmed")
	OS.set_environment("HERA_AGENT_GODOT_TOKEN", " env-token ")
	result = HttpServer.load_shared_token()
	_check(result is Dictionary and result.get("token") == "env-token", "environment token takes precedence")
	OS.unset_environment("HERA_AGENT_GODOT_TOKEN")
	file = FileAccess.open(token_path, FileAccess.WRITE)
	file.store_string(" \n")
	file.close()
	result = HttpServer.load_shared_token()
	_check(result is Dictionary and result.get("token") == "", "empty file intentionally disables auth")
	DirAccess.remove_absolute(token_path)
	DirAccess.make_dir_absolute(token_path)
	result = HttpServer.load_shared_token()
	_check(result is Dictionary and result.has("error"), "unreadable token path must not disable auth")
	DirAccess.remove_absolute(token_path)
	DirAccess.remove_absolute(token_dir)
	DirAccess.remove_absolute(home)
	for key: String in env_names:
		if saved[key] == null:
			OS.unset_environment(key)
		else:
			OS.set_environment(key, saved[key])

func _connect() -> StreamPeerTCP:
	var client := StreamPeerTCP.new()
	_check(client.connect_to_host("127.0.0.1", _server.port) == OK, "client connects")
	for i in 100:
		client.poll()
		_server.poll(_queue)
		if client.get_status() == StreamPeerTCP.STATUS_CONNECTED:
			return client
		await create_timer(0.01).timeout
	_check(false, "connection timed out")
	return client

func _wire(headers: String, body: String) -> PackedByteArray:
	return ("POST /rpc HTTP/1.1\r\nContent-Length: %d\r\n%s\r\n%s" % [body.to_utf8_buffer().size(), headers, body]).to_utf8_buffer()

func _receive_request() -> Dictionary:
	for i in 100:
		_server.poll(_queue)
		var items := _queue.drain()
		if not items.is_empty():
			return items[0]
		await create_timer(0.01).timeout
	return {}

func _request(headers: String, body: String, status: int) -> void:
	var client: StreamPeerTCP = await _connect()
	client.put_data(_wire(headers, body))
	var received := PackedByteArray()
	for i in 200:
		_server.poll(_queue)
		for item: Dictionary in _queue.drain():
			_server.respond(item["conn"], { "ok": true, "data": "hello 한글".repeat(20000) })
		client.poll()
		var count := client.get_available_bytes() if client.get_status() == StreamPeerTCP.STATUS_CONNECTED else 0
		if count > 0:
			var chunk := client.get_data(count)
			received.append_array(chunk[1])
		if client.get_status() != StreamPeerTCP.STATUS_CONNECTED:
			break
		await create_timer(0.01).timeout
	var text := received.get_string_from_utf8()
	_check(text.begins_with("HTTP/1.1 %d " % status), "HTTP status %d: %s" % [status, text.left(60)])
	if status == 200:
		_check(text.ends_with(JSON.stringify({ "ok": true, "data": "hello 한글".repeat(20000) })), "complete multi-chunk UTF-8 response")
	client.disconnect_from_host()

func _check(value: bool, message: String) -> void:
	if not value:
		_failed = true
		push_error("FAIL: " + message)
