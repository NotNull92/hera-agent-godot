extends CharacterBody2D

## Demo player for the Hera README walkthrough.
## Auto-drives across the viewport and bounces at the edges, so a running
## session shows live motion `hera game node get` can read back. Holding
## ui_left / ui_right steers it, so `hera game input` can prove input QA:
## inject an action, then read the position back to confirm the game responded.

const SPEED := 220.0

func _ready() -> void:
	velocity = Vector2(SPEED, 0.0)

func _physics_process(_delta: float) -> void:
	var steer := Input.get_axis("ui_left", "ui_right")
	if steer != 0.0:
		velocity.x = steer * SPEED
	move_and_slide()
	var bounds := get_viewport_rect().size
	if (global_position.x <= 0.0 and velocity.x < 0.0) or (global_position.x >= bounds.x and velocity.x > 0.0):
		velocity.x = -velocity.x
