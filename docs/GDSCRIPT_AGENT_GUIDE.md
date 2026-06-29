# GDScript Agent Guide

This guide is for agents writing or editing GDScript in this repository. It is
based on the official Godot stable documentation:

- GDScript basics: https://docs.godotengine.org/en/stable/tutorials/scripting/gdscript/gdscript_basics.html
- Control class reference: https://docs.godotengine.org/en/stable/classes/class_control.html

Read this before generating or editing `.gd` code. Do not rely on syntax from
JavaScript, C#, Python, or memory when Godot's syntax differs.

## Core Rule

When uncertain about syntax, stop and check the official Godot docs or an
existing project example. Do not invent GDScript syntax.

Every GDScript change must pass at least one parser/runtime check before the
work is called done:

```sh
godot --headless --path . --check-only --script res://path/to/script.gd
```

If a local `godot` command is unavailable, use the live editor through Hera:

```sh
hera diagnostics --lines 80
hera run --current
hera output --type error --lines 80
```

## File Shape

Typical script order:

```gdscript
extends Node2D
class_name PlayerController

const SPEED: float = 240.0

@export var health: int = 3
@onready var sprite: Sprite2D = $Sprite2D

var velocity: Vector2 = Vector2.ZERO


func _ready() -> void:
	sprite.visible = true


func _process(delta: float) -> void:
	position += velocity * delta
```

Use `extends` deliberately. It controls inherited methods, signals, and which
constants are available without qualification.

## Comments And Regions

Use `#` for comments. Use doc comments only for API-facing symbols.

```gdscript
# Internal note for future maintainers.
var move_count: int = 0

## Public helper shown in generated documentation.
func reset_board() -> void:
	move_count = 0
```

Regions are editor-folding aids, not logic boundaries:

```gdscript
#region AI
func choose_move() -> Vector2i:
	return Vector2i.ZERO
#endregion
```

Use comments sparingly. Prefer clear names and small functions.

## Identifiers And Keywords

Use snake_case for variables and functions, PascalCase for classes, and
SCREAMING_SNAKE_CASE for constants.

```gdscript
const BOARD_SIZE: int = 8

var current_player: int = 1

func apply_move(cell: Vector2i) -> void:
	pass
```

Do not use keywords as identifiers:

```gdscript
# Invalid.
var class := "Warrior"
```

## Built-In Types And Literals

Prefer explicit built-in types for values that cross function or collection
boundaries.

```gdscript
var count: int = 3
var ratio: float = 0.5
var title: String = "Othello"
var enabled: bool = true
var cell: Vector2i = Vector2i(3, 4)
var color: Color = Color("#245d2d")
```

Use Godot collection types with element annotations when possible:

```gdscript
var directions: Array[Vector2i] = [
	Vector2i(-1, -1),
	Vector2i(0, -1),
	Vector2i(1, -1),
]

var scores: Dictionary[String, int] = {
	"black": 2,
	"white": 2,
}
```

Untyped collections return `Variant` in many cases. Annotate locals when reading
from them:

```gdscript
var score: int = int(scores["black"])
```

## Variables And Type Inference

Use `:=` only when the right-hand side has a concrete type Godot can infer.

```gdscript
var button := Button.new()
var origin := Vector2.ZERO
```

Use explicit types when a value may be `Variant`, comes from an untyped
container, or comes from a dynamic Godot API.

```gdscript
var node: Node = get_node("Board")
var player_name: String = str(data["name"])
var r: int = row + int(direction.y)
```

Avoid this pattern when the right-hand side is dynamic:

```gdscript
# Risky: Godot may not infer the type.
var r := row + direction.y
```

Prefer initialized declarations:

```gdscript
var legal_moves: Dictionary[Vector2i, Array[Vector2i]] = {}
```

If a value is assigned later, still declare its type:

```gdscript
var selected_cell: Vector2i
selected_cell = Vector2i(2, 3)
```

## Constants, Enums, And Owner-Qualified Values

Use typed constants for script-owned values:

```gdscript
const EMPTY: int = 0
const BLACK: int = 1
const WHITE: int = 2
```

Use enums when values form a closed set:

```gdscript
enum Player {
	BLACK,
	WHITE,
}

var current_player: Player = Player.BLACK
```

Qualify engine enum values, constants, and flags with the owning class unless
the constant is declared in the same script scope.

```gdscript
bg.set_anchors_preset(Control.PRESET_FULL_RECT)
panel.size_flags_horizontal = Control.SIZE_EXPAND_FILL
mouse_filter = Control.MOUSE_FILTER_STOP
```

Do not assume bare names are visible:

```gdscript
# Invalid from a Node2D script.
bg.set_anchors_preset(PRESET_FULL_RECT)
```

When using a Godot class constant for the first time, check the class reference.

## Operators

Use GDScript logical keywords in generated code:

```gdscript
if is_ready and not game_over:
	start_turn()
```

Avoid borrowing symbolic aliases from other languages:

```gdscript
# Avoid in generated code.
if is_ready && !game_over:
	start_turn()
```

Use `is` for runtime type checks:

```gdscript
if event is InputEventMouseButton:
	handle_click(event)
```

Use `as` only when you are intentionally casting an object reference:

```gdscript
var label := get_node("StatusLabel") as Label
if label != null:
	label.text = "Ready"
```

## Ternary Expressions

GDScript's ternary form is:

```gdscript
var label_text := "AI: ON" if ai_enabled else "AI: OFF"
```

Never use C-style ternaries:

```gdscript
# Invalid GDScript.
var label_text = ai_enabled ? "AI: ON" : "AI: OFF"
```

For multiline logic, use normal `if` statements instead of dense ternaries:

```gdscript
if ai_enabled:
	status_text = "White AI enabled."
else:
	status_text = "White AI disabled."
```

## Functions

Annotate parameters and return types.

```gdscript
func score_move(cell: Vector2i, flips: Array[Vector2i]) -> int:
	return flips.size()
```

Use `-> void` for procedures:

```gdscript
func reset_game() -> void:
	move_count = 0
```

Use default parameters when appropriate:

```gdscript
func show_message(text: String, seconds: float = 2.0) -> void:
	status_label.text = text
```

Static functions cannot access instance state:

```gdscript
static func opponent(player: int) -> int:
	return 2 if player == 1 else 1
```

Avoid untyped helper functions:

```gdscript
# Poor: return type and parameter types are unclear.
func score(x):
	return x.size()
```

## Lambdas And Callables

Use lambdas only for short callbacks where naming a function would add noise.

```gdscript
button.pressed.connect(func() -> void:
	reset_game()
)
```

For reusable callbacks, use a named method:

```gdscript
button.pressed.connect(_on_restart_pressed)


func _on_restart_pressed() -> void:
	reset_game()
```

## Control Flow

Use ordinary `if`/`elif`/`else`:

```gdscript
if black_score > white_score:
	status_text = "Black wins."
elif white_score > black_score:
	status_text = "White wins."
else:
	status_text = "Draw."
```

Use `for` with ranges:

```gdscript
for row in range(8):
	for col in range(8):
		clear_cell(Vector2i(col, row))
```

Use typed loop locals when collection values are dynamic:

```gdscript
for value in raw_values:
	var cell: Vector2i = value
	highlight_cell(cell)
```

Use `while` for directional scans or loops with explicit stopping conditions:

```gdscript
while in_bounds(row, col) and board[row][col] == opponent:
	captured.append(Vector2i(col, row))
	row += step.y
	col += step.x
```

Use `continue` and `break` plainly:

```gdscript
for cell in cells:
	if not is_legal(cell):
		continue
	play(cell)
	break
```

## Match

Use `match` for discrete values and state machines.

```gdscript
match current_player:
	Player.BLACK:
		status_text = "Black to move"
	Player.WHITE:
		status_text = "White to move"
	_:
		push_error("Unknown player")
```

Use array patterns only when the structure is known:

```gdscript
match command:
	["move", var x, var y]:
		play_move(Vector2i(int(x), int(y)))
	["restart"]:
		reset_game()
	_:
		push_warning("Unknown command")
```

Do not replace clear `match` state handling with chained string comparisons.

## Classes And Inheritance

Use `class_name` only when the type should be globally available.

```gdscript
extends Node
class_name BoardState
```

Inner classes are useful for small local data structures:

```gdscript
class Move:
	var cell: Vector2i
	var flips: Array[Vector2i]

	func _init(new_cell: Vector2i, new_flips: Array[Vector2i]) -> void:
		cell = new_cell
		flips = new_flips
```

Call superclass methods intentionally:

```gdscript
func _ready() -> void:
	super()
	reset_game()
```

Only call `super()` when the parent method exists and should run.

## Loading Resources

Use `preload` for fixed dependencies known at parse time:

```gdscript
const BoardViewScene := preload("res://scenes/BoardView.tscn")
```

Use `load` for dynamic paths:

```gdscript
var resource: Resource = load(selected_path)
```

After `load`, check or cast the type before use:

```gdscript
var script := load("res://scripts/board_view.gd") as Script
if script == null:
	push_error("Failed to load board script")
	return
```

## Properties: get And set

Use properties for small state normalization. Do not hide expensive logic in a
property setter.

```gdscript
var score: int = 0:
	set(value):
		score = max(value, 0)
```

For side-effect-heavy updates, prefer a named method:

```gdscript
func set_current_player(player: Player) -> void:
	current_player = player
	update_turn_label()
```

## Exports

Use `@export` for editor-tunable values:

```gdscript
@export var board_size: int = 8
@export var piece_radius: float = 28.0
@export var board_color: Color = Color("#245d2d")
```

Use ranges for numeric editor controls:

```gdscript
@export_range(0.0, 1.0, 0.05) var ai_delay: float = 0.25
```

Export node references when scene wiring is stable:

```gdscript
@export var status_label: Label
```

If a node is found by path, use `@onready` and type it:

```gdscript
@onready var status_label: Label = $Panel/StatusLabel
```

## Annotations

Use annotations exactly as Godot documents them.

```gdscript
@tool
extends Node2D

@export var preview_enabled: bool = true

@onready var board: Node2D = $Board
```

`@tool` runs code in the editor. Guard runtime-only behavior:

```gdscript
func _ready() -> void:
	if Engine.is_editor_hint():
		return
	start_game()
```

Avoid adding `@tool` just to fix rendering unless editor-time execution is
required and safe.

## Node Paths And Scene Tree Access

Prefer exported node references or typed `@onready` paths.

```gdscript
@onready var restart_button: Button = $UI/RestartButton
```

When using `get_node`, annotate or cast:

```gdscript
var restart_button: Button = get_node("UI/RestartButton") as Button
if restart_button == null:
	push_error("Missing RestartButton")
	return
```

Use `get_node_or_null` when a node may be absent:

```gdscript
var debug_panel := get_node_or_null("DebugPanel") as Control
if debug_panel != null:
	debug_panel.visible = false
```

## Signals

Connect signals to named methods for normal gameplay UI.

```gdscript
restart_button.pressed.connect(_on_restart_pressed)


func _on_restart_pressed() -> void:
	reset_game()
```

Use lambda callbacks only for tiny local behavior:

```gdscript
close_button.pressed.connect(func() -> void:
	hide()
)
```

Define custom signals with typed parameters:

```gdscript
signal move_played(cell: Vector2i, player: int)


func play_move(cell: Vector2i) -> void:
	move_played.emit(cell, current_player)
```

## Await And Coroutines

Use `await` with signals or functions that intentionally suspend.

```gdscript
await get_tree().create_timer(0.25).timeout
play_ai_move()
```

Do not use arbitrary waits to paper over missing state checks. Prefer signals or
explicit conditions.

```gdscript
await animation_player.animation_finished
```

For delayed game actions such as AI moves, keep a generation token and check it
after every `await`. Increment the token on restart, undo, scene reset, or mode
toggle so stale delayed work cannot mutate the new state.

```gdscript
var turn_token: int = 0


func restart_game() -> void:
	turn_token += 1
	reset_board()


func queue_ai_turn() -> void:
	turn_token += 1
	var request_id: int = turn_token
	_ai_turn(request_id)


func _ai_turn(request_id: int) -> void:
	await get_tree().create_timer(0.25).timeout
	if request_id != turn_token:
		return
	play_ai_move()
```

For board and grid games, prefer localized state changes over whole-state
rebuilds in hot paths. Full-board scans are fine for tiny boards, but keep the
rule function pure and isolated, store only the changed cells for undo when that
is enough, and refresh only derived state that depends on the last move.

```gdscript
var undo_stack: Array[Dictionary] = []


func apply_move(cell: Vector2i, flips: Array[Vector2i]) -> void:
	undo_stack.append({
		"cell": cell,
		"flips": flips.duplicate(),
		"player": current_player,
	})
	board[cell.y][cell.x] = current_player
	for flip in flips:
		board[flip.y][flip.x] = current_player
```

## Assertions And Errors

Use `assert` for programmer invariants, not player-facing validation.

```gdscript
assert(board.size() == 8)
```

Use `push_error` or `push_warning` for recoverable diagnostics:

```gdscript
if not legal_moves.has(cell):
	push_warning("Rejected illegal move")
	return
```

Use user-visible labels or UI for expected gameplay states:

```gdscript
status_label.text = "That move is not legal."
```

## Memory And Object Lifetime

Use `queue_free()` for nodes in the scene tree:

```gdscript
for child in board_container.get_children():
	child.queue_free()
```

Do not keep references to nodes after freeing them:

```gdscript
selected_piece.queue_free()
selected_piece = null
```

Use `Resource` and `RefCounted` types normally; do not call `free()` unless the
class lifecycle requires it.

## Common Parser Errors To Prevent

`Unexpected "?" in source`

```gdscript
# Wrong.
var text = enabled ? "On" : "Off"

# Right.
var text := "On" if enabled else "Off"
```

`Identifier "PRESET_FULL_RECT" not declared`

```gdscript
# Wrong from a Node2D script.
bg.set_anchors_preset(PRESET_FULL_RECT)

# Right.
bg.set_anchors_preset(Control.PRESET_FULL_RECT)
```

`Cannot infer the type of "x" variable`

```gdscript
# Risky when data is Variant or untyped.
var x := data["x"]

# Right.
var x: int = int(data["x"])
```

`Invalid call. Nonexistent function`

```gdscript
# Risky: guessed API name.
node.set_full_rect()

# Right: check class docs and use real API.
control.set_anchors_preset(Control.PRESET_FULL_RECT)
```

## Hera Verification Checklist

Before finishing GDScript work:

- The live editor state was inspected with Hera before structural scene changes.
- Godot class constants and methods were checked against docs or existing code.
- No C-style `? :` ternaries exist.
- No unqualified engine constants copied from class references exist.
- No `:=` is assigned from `Variant`, untyped `Array`, untyped `Dictionary`, or a
  dynamic API result.
- `hera diagnostics --lines 80` is clean.
- Runtime output has no errors after `hera run --current` when runtime behavior
  changed.
- UI or visual changes have a Hera screenshot, preferably runtime when a game
  viewport is available.
