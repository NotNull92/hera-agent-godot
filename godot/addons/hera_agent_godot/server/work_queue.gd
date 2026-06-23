extends RefCounted

var _items: Array[Dictionary] = []

func enqueue(item: Dictionary) -> void:
	_items.append(item)

func drain() -> Array[Dictionary]:
	var drained := _items
	_items = []
	return drained
