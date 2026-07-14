extends CharacterBody2D

## Demo player for the Hera README walkthrough.
## Drives itself across the viewport so a running session shows live motion
## that `hera game node get` can read back.

const SPEED := 220.0

func _ready() -> void:
	velocity = Vector2(SPEED, 0.0)

func _physics_process(_delta: float) -> void:
	move_and_slide()
	var bounds := get_viewport_rect().size
	if global_position.x <= 0.0 or global_position.x >= bounds.x:
		velocity.x = -velocity.x
