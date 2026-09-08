extends RefCounted

static func rect(control: Control) -> Rect2:
	return control.get_global_transform_with_canvas() * Rect2(Vector2.ZERO, control.size)

static func center(control: Control) -> Vector2:
	return control.get_global_transform_with_canvas() * (control.size * 0.5)
