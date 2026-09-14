extends RefCounted

static func success(data: Variant) -> Dictionary:
	return {
		"ok": true,
		"data": data,
	}

static func failure(error: String) -> Dictionary:
	return {
		"ok": false,
		"error": error,
	}

static func rejected(error: String, code: String = "") -> Dictionary:
	var result := {
		"ok": false,
		"error": error,
		"attempted": false,
	}
	if code != "":
		result["error_code"] = code
	return result
