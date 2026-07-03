# Godot Editor Analysis Strategy

Hera should not treat Godot like a closed editor binary. Unity-style binary
inspection is possible, but Godot gives us a stronger path: source-aligned
editor API work, live object introspection, and a debug-symbol editor build when
native stepping is needed.

## 1. Source-aligned analysis

Start from the exact Godot editor version reported by `hera status`, then read
the matching Godot source tag. Use source files to decide whether a capability
belongs in Hera, not to guess at runtime state from disk.

| Need | Primary Godot source area | Hera surface |
|---|---|---|
| Open scenes, edited root, play state, editor selection | `editor/editor_interface.cpp`, `editor/editor_node.cpp` | `editor state`, `scene list`, `scene tree`, `run`, `stop` |
| Scene tree edits and undo behavior | `editor/docks/scene_tree_dock.cpp`, `scene/main/node.cpp` | `node add/instance/set/remove`, `signal connect/disconnect` |
| Inspector-visible data | `core/object/object.cpp`, `core/object/class_db.cpp` | `node get`, `resource get`, `classdb *` |
| Script editor and focused script | `editor/plugins/script_editor_plugin.cpp`, `editor/editor_interface.cpp` | `script current`, `script inspect`, `script open` |
| Runtime game process state | `scene/main/scene_tree.cpp`, `scene/main/window.cpp` | `game tree`, `game ui tree`, `game node get/set/call`, `game screenshot` |

Useful upstream references:

- Editor plugins must be `@tool` scripts that inherit `EditorPlugin`:
  https://docs.godotengine.org/en/stable/tutorials/plugins/editor/making_plugins.html
- `EditorInterface` exposes editor scene, script, resource, viewport, and play
  controls:
  https://docs.godotengine.org/en/stable/classes/class_editorinterface.html
- Godot source entry points:
  https://github.com/godotengine/godot/blob/master/editor/editor_interface.cpp
  and https://github.com/godotengine/godot/blob/master/editor/editor_node.cpp

## 2. Expose editor APIs through explicit Hera tools

Prefer an explicit CLI command over generic reflection. Each command should map
to a Godot concept an agent can verify:

- `editor`: editor state, selection, focused script.
- `scene`: open/current scene management.
- `node`: edited-scene node discovery and undoable mutations.
- `resource`: resource inspection and persistent resource edits.
- `signal`: signal metadata and undoable connection changes.
- `classdb`: engine API metadata without loading a large schema into agent
  context.
- `game`: runtime-only inspection/control during a play session.

When a new Godot source investigation finds a missing editor API, add the
smallest explicit command that exposes it, then verify it through a live editor
or a parser/unit test when no live editor is available.

## 3. Introspection contract

Use Godot's own metadata APIs before reaching for binaries:

- `Object.get_property_list()` drives `node get`, `resource get`, runtime
  property reads, and value coercion.
- `Object.get_signal_list()` drives node signal inspection.
- `ClassDB.class_get_method_list()` and `class_get_property_list()` expose
  engine method/property metadata.
- `ClassDB.class_get_signal_list()`, `class_get_integer_constant_list()`, and
  `class_get_enum_list()` expose signals, constants, and enums. Hera includes
  inherited data for these commands so queries such as `Button` include the
  usable signals declared by parent classes such as `BaseButton`; add `--own`
  when you only need metadata declared directly on that class and want smaller
  output.

The `classdb` command is the low-token entry point for this contract:

```text
hera classdb info <Class>
hera classdb methods <Class>
hera classdb properties <Class>
hera classdb signals <Class>
hera classdb constants <Class>
hera classdb enums <Class>
hera classdb signals <Class> --own
hera classdb constants <Class> --own
hera classdb enums <Class> --own
hera classdb inherits <Class> <BaseClass>
```

Godot's ClassDB documentation notes that exported release builds can lack debug
detail for method dictionaries. If a Hera feature depends on full native method
metadata, verify it against the user's actual editor build and fall back to a
debug build when needed:
https://docs.godotengine.org/en/stable/classes/class_classdb.html

## 4. Debug editor build path

Use a debug-symbol editor build only after the API/introspection path cannot
answer the question or a native crash must be stepped.

Recommended loop:

1. Capture observable evidence first: `hera status`, `hera diagnostics`,
   `hera output --type error`, and a minimal reproduction scene.
2. Check the matching Godot source tag for the editor version.
3. Build Godot with debug symbols. Godot's build docs document
   `debug_symbols=yes`, and `dev_build=yes` is useful for editor debugging.
4. Launch the same project with the debug editor and keep Hera enabled.
5. Attach WinDbg, Visual Studio, lldb, or gdb to the debug editor.
6. Reproduce the Hera command, then set breakpoints in the source areas mapped
   in section 1.
7. Convert the finding back into an explicit Hera command or a documented
   limitation. Do not leave the result as a private debugger note.

Build references:

- Godot buildsystem debug symbols:
  https://docs.godotengine.org/en/latest/engine_details/development/compiling/introduction_to_the_buildsystem.html
- Windows editor compilation:
  https://docs.godotengine.org/en/4.4/contributing/development/compiling/compiling_for_windows.html

## Binary analysis boundary

Binary inspection remains a last resort for packaging, signature, dependency,
or crash-dump questions:

- PE/import/export/DLL dependency inspection is valid for local environment
  issues.
- Disassembly is valid for confirming behavior in a stripped third-party binary
  when source and symbols cannot be matched.
- Hera feature design should not depend on patched or private editor internals.
  If the Godot API cannot expose a behavior, prefer a documented limitation or
  a small upstream/GDExtension route over brittle binary coupling.
