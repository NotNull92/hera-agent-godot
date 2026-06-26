# GDScript Agent Guide

This guide is for agents writing or editing GDScript in this repository. It is
based on the official Godot stable documentation:

- GDScript basics: https://docs.godotengine.org/en/stable/tutorials/scripting/gdscript/gdscript_basics.html
- Control class reference: https://docs.godotengine.org/en/stable/classes/class_control.html

Read this before generating `.gd` code. Do not rely on syntax from JavaScript,
C#, Python, or memory when Godot's syntax differs.

## Non-negotiable syntax rules

- Use GDScript's ternary form:

  ```gdscript
  var label_text := "AI: ON" if ai_enabled else "AI: OFF"
  ```

  Never write C-style ternaries:

  ```gdscript
  # Invalid GDScript.
  var label_text = ai_enabled ? "AI: ON" : "AI: OFF"
  ```

- Qualify engine enum values, constants, and flags with the owning class unless
  the constant is declared in the same script scope.

  ```gdscript
  bg.set_anchors_preset(Control.PRESET_FULL_RECT)
  panel.size_flags_horizontal = Control.SIZE_EXPAND_FILL
  ```

  Do not assume bare names such as `PRESET_FULL_RECT` or `SIZE_EXPAND_FILL` are
  visible from a script that extends another class.

- Use explicit types when a value may be `Variant`, comes from an untyped
  `Array`/`Dictionary`, or comes from a dynamic Godot API.

  ```gdscript
  var r: int = row + int(direction.y)
  var button: Button = Button.new()
  ```

  Use `:=` only when the assigned value has a concrete type Godot can infer.

- Do not infer from untyped containers.

  ```gdscript
  var moves: Dictionary[Vector2i, Array] = {}
  var directions: Array[Vector2i] = [
  	Vector2i.LEFT,
  	Vector2i.RIGHT,
  ]
  ```

  Dictionary methods and untyped collection access can still return `Variant`,
  so annotate locals at the boundary.

- Prefer GDScript logical keywords over symbolic aliases in generated code:
  `and`, `or`, `not` instead of `&&`, `||`, `!`.

- `extends` controls inherited constants and methods. If a script extends
  `Node2D`, it does not automatically expose `Control` constants as bare names.

## Authoring workflow

1. Inspect the live editor state with Hera before deciding scene structure.
2. Check the relevant Godot class reference before using an engine constant,
   enum, method, signal, or property that is not already used in the project.
3. Write GDScript with explicit types at dynamic boundaries.
4. After any `.gd` edit, run a parser check before runtime QA:

   ```sh
   godot --headless --path . --check-only --script res://path/to/script.gd
   ```

   If a local `godot` binary is not available, use the live editor through Hera:

   ```sh
   hera diagnostics --lines 80
   hera run --current
   hera output --type error --lines 80
   ```

5. For scene or visual work, also verify through the matching Hera surface:
   `scene tree`, `node get`, `run`, `output`, and screenshot commands as
   appropriate.

## Common parser errors to prevent

- `Unexpected "?" in source`: replace `condition ? a : b` with
  `a if condition else b`.
- `Identifier "PRESET_FULL_RECT" not declared`: qualify it as
  `Control.PRESET_FULL_RECT` or use the documented owner class for that enum.
- `Cannot infer the type of ...`: replace `:=` with an explicit type or cast the
  dynamic value before assignment.

## Quick checklist before finishing

- No `? :` ternaries.
- No unqualified engine constants copied from class references.
- No `:=` assigned from `Variant`, untyped `Array`, untyped `Dictionary`, or a
  dynamic API result.
- Hera diagnostics are clean.
- Runtime output has no errors after running the scene when runtime behavior was
  changed.
