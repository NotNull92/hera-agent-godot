extends RefCounted

const HeraSettings = preload("res://addons/hera_agent_godot/core/hera_settings.gd")


static func hint_for_node_type(type_name: String) -> String:
	if not HeraSettings.get_game_feel_mode_enabled():
		return ""
	var topics := _topics_for_node_type(type_name)
	if topics.is_empty():
		return ""
	var also := ""
	if topics.size() > 1:
		also = " (also: %s)" % ", ".join(topics.slice(1))
	return "[Hera] Game Feel Mode (Beta) is on - before wiring feel/feedback for this %s, run `game_feel %s`%s for concrete parameters. Honest Juice: presentation intensity must match real achievement." % [type_name, topics[0], also]


static func _topics_for_node_type(type_name: String) -> Array[String]:
	if ClassDB.is_parent_class(type_name, "Camera2D") or ClassDB.is_parent_class(type_name, "Camera3D"):
		return ["camera", "screen_shake"]
	if ClassDB.is_parent_class(type_name, "GPUParticles2D") or ClassDB.is_parent_class(type_name, "GPUParticles3D") or ClassDB.is_parent_class(type_name, "CPUParticles2D") or ClassDB.is_parent_class(type_name, "CPUParticles3D"):
		return ["particles"]
	if ClassDB.is_parent_class(type_name, "AudioStreamPlayer") or ClassDB.is_parent_class(type_name, "AudioStreamPlayer2D") or ClassDB.is_parent_class(type_name, "AudioStreamPlayer3D"):
		return ["sound"]
	if ClassDB.is_parent_class(type_name, "CharacterBody2D") or ClassDB.is_parent_class(type_name, "CharacterBody3D") or ClassDB.is_parent_class(type_name, "RigidBody2D") or ClassDB.is_parent_class(type_name, "RigidBody3D"):
		return ["control_feel", "knockback"]
	if ClassDB.is_parent_class(type_name, "Light2D") or ClassDB.is_parent_class(type_name, "Light3D"):
		return ["dynamic_lighting"]
	if ClassDB.is_parent_class(type_name, "AnimationPlayer") or ClassDB.is_parent_class(type_name, "AnimationTree") or ClassDB.is_parent_class(type_name, "Tween"):
		return ["squash_stretch", "tweening_easing"]
	return []
