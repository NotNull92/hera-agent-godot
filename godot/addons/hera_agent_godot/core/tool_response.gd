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
