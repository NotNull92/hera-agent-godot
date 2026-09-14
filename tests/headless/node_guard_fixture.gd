@tool
extends Node2D

@export var amount: float = 1.0:
	set(value):
		amount = clampf(value, 0.0, 10.0)

@export var retire: bool = false:
	set(value):
		retire = value
		if value:
			queue_free()

@export var integer: int = 9007199254740993
@export var text: String = "hello"
@export var number: float = 1.23456789123456
