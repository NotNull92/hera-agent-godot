@tool
extends EditorExportPlugin

var _owner: EditorPlugin


func setup(owner: EditorPlugin) -> void:
	_owner = owner


func _get_name() -> String:
	return "HeraGameInspector"


func _export_begin(_features: PackedStringArray, _is_debug: bool, _path: String, _flags: int) -> void:
	if _owner != null and is_instance_valid(_owner):
		_owner.suspend_game_autoload_for_export()


func _export_end() -> void:
	if _owner != null and is_instance_valid(_owner):
		_owner.restore_game_autoload_after_export()
