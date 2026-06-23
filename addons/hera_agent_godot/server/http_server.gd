extends RefCounted

# Minimal HTTP/1.1 server for the addon, built on Godot's non-blocking TCPServer.
#
# Everything runs on the editor main thread: poll() is called from the plugin's
# _process, accepts connections, accumulates bytes, and enqueues a completed
# request onto the WorkQueue. The plugin drains the queue (still on the main
# thread, so editor APIs are safe) and calls respond() to write the reply.
#
# One request per connection (Connection: close). Binds 127.0.0.1 only and
# rejects browser-origin requests. See docs/ARCHITECTURE.md §1, §7.

const ToolResponse = preload("res://addons/hera_agent_godot/core/tool_response.gd")

const CLIENT_TIMEOUT_MSEC := 5000
const MAX_REQUEST_BYTES := 1048576

var port := 0

var _server: TCPServer
var _clients: Array = [] # each entry: { "conn": StreamPeerTCP, "buf": PackedByteArray, "accepted_msec": int }

# Bind to the first free port in [base_port, base_port + attempts).
# Returns the bound port, or 0 on failure.
func start(base_port := 8770, attempts := 16) -> int:
	_server = TCPServer.new()
	for i in attempts:
		var candidate: int = base_port + i
		if _server.listen(candidate, "127.0.0.1") == OK:
			port = candidate
			return port
	_server = null
	port = 0
	return 0

func stop() -> void:
	for entry in _clients:
		(entry["conn"] as StreamPeerTCP).disconnect_from_host()
	_clients.clear()
	if _server != null:
		_server.stop()
		_server = null
	port = 0

# Accept new connections, read available bytes, and enqueue completed requests.
func poll(queue) -> void:
	if _server == null:
		return
	var now_msec := Time.get_ticks_msec()
	while _server.is_connection_available():
		_clients.append({ "conn": _server.take_connection(), "buf": PackedByteArray(), "accepted_msec": now_msec })

	var keep: Array = []
	for entry in _clients:
		var conn: StreamPeerTCP = entry["conn"]
		conn.poll()
		if now_msec - int(entry["accepted_msec"]) > CLIENT_TIMEOUT_MSEC:
			conn.disconnect_from_host()
			continue
		var status := conn.get_status()
		if status == StreamPeerTCP.STATUS_CONNECTING:
			keep.append(entry)
			continue
		if status != StreamPeerTCP.STATUS_CONNECTED:
			continue # dropped/errored — discard

		var available := conn.get_available_bytes()
		if available > 0:
			var chunk := conn.get_data(available)
			if chunk[0] == OK:
				var buf: PackedByteArray = entry["buf"]
				buf.append_array(chunk[1])
				entry["buf"] = buf
				if buf.size() > MAX_REQUEST_BYTES:
					_write_http(conn, 413, "Payload Too Large", ToolResponse.failure("request too large"))
					continue

		var parsed := _parse_request(entry["buf"])
		if not parsed["complete"]:
			keep.append(entry)
			continue

		# Complete request: validate, then enqueue or reject inline.
		if bool(parsed["too_large"]):
			_write_http(conn, 413, "Payload Too Large", ToolResponse.failure("request too large"))
		elif String(parsed["origin"]) != "":
			_write_http(conn, 403, "Forbidden", ToolResponse.failure("forbidden: browser origin not allowed"))
		elif String(parsed["method"]) != "POST" or String(parsed["path"]) != "/rpc":
			_write_http(conn, 404, "Not Found", ToolResponse.failure("not found"))
		elif parsed["body"] == null:
			_write_http(conn, 400, "Bad Request", ToolResponse.failure("invalid json body"))
		else:
			queue.enqueue({ "conn": conn, "request": parsed["body"] })
	_clients = keep

# Write a successful tool response and close the connection.
func respond(conn: StreamPeerTCP, response: Dictionary) -> void:
	_write_http(conn, 200, "OK", response)

func _write_http(conn: StreamPeerTCP, code: int, reason: String, body: Dictionary) -> void:
	var body_bytes := JSON.stringify(body).to_utf8_buffer()
	var header := "HTTP/1.1 %d %s\r\n" % [code, reason]
	header += "Content-Type: application/json\r\n"
	header += "Content-Length: %d\r\n" % body_bytes.size()
	header += "Connection: close\r\n\r\n"
	conn.put_data(header.to_ascii_buffer())
	conn.put_data(body_bytes)
	conn.disconnect_from_host()

# Parse an accumulated buffer into { complete, method, path, origin, body }.
# body is the decoded JSON Dictionary (or null on parse failure / empty).
func _parse_request(buf: PackedByteArray) -> Dictionary:
	var result := { "complete": false, "too_large": false, "method": "", "path": "", "origin": "", "body": null }
	var header_end := _find_header_end(buf)
	if header_end == -1:
		return result # headers not fully received yet

	var header_text := buf.slice(0, header_end).get_string_from_ascii()
	var lines := header_text.split("\r\n", false)
	if lines.size() > 0:
		var request_line: PackedStringArray = lines[0].split(" ", false)
		if request_line.size() >= 2:
			result["method"] = request_line[0]
			result["path"] = request_line[1]

	var content_length := 0
	for i in range(1, lines.size()):
		var line: String = lines[i]
		var colon := line.find(":")
		if colon == -1:
			continue
		var key := line.substr(0, colon).strip_edges().to_lower()
		var value := line.substr(colon + 1).strip_edges()
		if key == "content-length":
			content_length = value.to_int()
		elif key == "origin":
			result["origin"] = value

	if content_length > MAX_REQUEST_BYTES:
		result["complete"] = true
		result["too_large"] = true
		return result

	var body_start := header_end + 4
	if buf.size() - body_start < content_length:
		return result # body still arriving

	result["complete"] = true
	if content_length <= 0:
		result["body"] = {}
		return result

	var body_text := buf.slice(body_start, body_start + content_length).get_string_from_utf8()
	var decoded: Variant = JSON.parse_string(body_text)
	if typeof(decoded) == TYPE_DICTIONARY:
		result["body"] = decoded
	return result

func _find_header_end(buf: PackedByteArray) -> int:
	for i in range(buf.size() - 3):
		if buf[i] == 13 and buf[i + 1] == 10 and buf[i + 2] == 13 and buf[i + 3] == 10:
			return i
	return -1
