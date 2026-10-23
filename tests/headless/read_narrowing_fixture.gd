@tool
extends Node2D

@export var amount: int = 7
@export var count: int = 3

func _property_can_revert(property: StringName) -> bool:
	return property == &"amount"

func _property_get_revert(property: StringName) -> Variant:
	return 9 if property == &"amount" else null
