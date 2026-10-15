# Godot Editor Analysis Strategy

Hera should not treat Godot like a closed editor binary. Godot gives a
source-aligned path: editor API work, live object introspection, and a
debug-symbol editor build when native stepping is needed.

## 1. Source-aligned analysis

Start from the exact Godot editor version reported by `hera-godot status`, then read
the matching Godot source tag. Use source files to decide whether a capability
belongs in Hera, not to guess at runtime state from disk.

| Need | Primary Godot source area | Hera surface |
|---|---|---|
| Open scenes, edited root, play state, editor selection | `editor/editor_interface.cpp`, `editor/editor_node.cpp` | `editor state`, `scene list`, `scene tree`, `run`, `stop` |
| Scene tree edits and undo behavior | `editor/docks/scene_tree_dock.cpp`, `scene/main/node.cpp` | `node add/instance/set/remove/reparent`, `signal connect/disconnect` |
| Inspector-visible data | `core/object/object.cpp`, `core/object/class_db.cpp` | `node get`, `resource get`, `classdb *` |
| Script editor and focused script | `editor/plugins/script_editor_plugin.cpp`, `editor/editor_interface.cpp` | `script current`, `script inspect`, `script open` |
| Runtime game process state | `scene/main/scene_tree.cpp`, `scene/main/window.cpp` | `game tree`, `game ui tree`, `game clock`, `game node get/set/call`, `game screenshot` |

Useful upstream references:

- Editor plugins must be `@tool` scripts that inherit `EditorPlugin`:
  https://docs.godotengine.org/en/stable/tutorials/plugins/editor/making_plugins.html
- `EditorInterface` exposes editor scene, script, resource, viewport, and play
  controls:
  https://docs.godotengine.org/en/stable/classes/class_editorinterface.html
- Godot source entry points:
  https://github.com/godotengine/godot/blob/master/editor/editor_interface.cpp
  and https://github.com/godotengine/godot/blob/master/editor/editor_node.cpp

## 2. Engine CLI surface (verified on Godot 4.7.2, 2026-10-15)

`hera-godot` started as a live-editor tool. The engine binary itself already
ships the no-editor half of the workflow, so new commands should wrap that
surface instead of inventing one. Verified locally against
`Godot_v4.7.2-stable_mono_win64.exe` (SHA-256 `45336315…dfab`) by running the
binary in an isolated `HOME`/`USERPROFILE`/`APPDATA` with per-run timeouts. The
full evidence (string RVAs, probe logs, limits) is in
`docs/reports/2026-10-15-godot-472-engine-surface-analysis.md` (local-only:
`docs/reports/` is gitignored).

| Need | Engine surface | Verified result |
|---|---|---|
| Engine identity / build kind | `--version`, `--help` legend `R`/`D`/`X`/`E` | `--version` → `4.7.2.stable.mono.official.ed1daf0bf`; editor builds list `-e, --editor` |
| Headless script harness | `--headless --script <res://tool.gd> -- <args>` | exit 0; args arrive via `OS.get_cmdline_user_args()`; script must extend `SceneTree`/`MainLoop` |
| Bounded runs | `--quit-after <frames>`, `--fixed-fps`, `--scene <path\|uid>` | exit 0; `--scene uid://…` runs a non-main scene |
| Asset import | `--import` | exit 0; writes `.godot/**` caches and `.gd.uid` sidecars; progress text is **localized** |
| Packaging | `--export-pack`, `--export-release\|--export-debug\|--export-patch <preset> <path>` | `--export-pack` succeeds **without export templates**; `--export-release` needs `export_templates/<ver>/…` |
| Deterministic capture | `--write-movie <file.avi\|.png>` + `--fixed-fps` + `--quit-after` | 10-frame AVI (MJPEG) and PNG+WAV frames; `MovieWriterMJPEG`/`MovieWriterPNGWAV`/`MovieWriterOGV` |
| Offline knowledge | `--class-db-json`, `--doctool`, `--dump-extension-api-with-docs`, `--gdscript-docs res://` | ClassDB JSON 6.15 MB + editor variant; 1,077 class XML; 11.87 MB API JSON; per-script project XML |
| Timing | `--benchmark --benchmark-file <path>` | JSON of phase → seconds |
| Code intelligence | `--lsp-port`, `--dap-port` | TCP listeners open; LSP reports type/name errors for **unsaved buffers**; DAP answers `initialize`/`setBreakpoints` (source checksums + `verified`) |
| Agent debug session | `--debug-server tcp://127.0.0.1:<port>` + game `--remote-debug tcp://127.0.0.1:<port>` + DAP `attach` | **Verified end to end (2026-10-15)**: `attach` → `process` event → `setBreakpoints` (`verified: true`, MD5+SHA256) → `configurationDone` → `stopped` (`reason: breakpoint`, `hitBreakpointIds`) → `stackTrace` (`_process` at the breakpoint line) → `scopes` (Locals/Members/Globals) → `variables` (`_d = 0.0166…`) → `evaluate "counter"` → `703` → `continue` → second `stopped` → `disconnect`. The editor does **not** need a scene open and the game can be launched by Hera itself. |
| Runtime protocol | `EditorDebuggerPlugin` + `scene:*` channels | plugin contract documented in `GodotSharpEditor.xml`; `_has_capture("scene")` routes `scene:*` messages |

Engine behavior that new commands must respect:

- **DAP `launch` is not the way in.** In 4.7.2 a GDScript debug session is best
  established as: editor `--debug-server tcp://127.0.0.1:<port>` (+ `--dap-port`),
  game launched with `--remote-debug tcp://127.0.0.1:<port>`, then DAP `attach`.
  `launch` produced no response in both headless and windowed editors, while
  `attach` returned `success: true` and delivered a real breakpoint hit.
- **Send breakpoint checksums.** A `setBreakpoints` request whose `source` omits
  `checksums` still succeeds, but the engine logs
  `Bug: Dictionary::operator[] … key "checksums"` and
  `Couldn't find the given section "res://<script>" and key "state"`. A Hera DAP
  client should send MD5/SHA256 (the adapter advertises
  `supportedChecksumAlgorithms`), which also ties the breakpoint to exact source.
- `--headless` uses the dummy renderer: viewport textures are null and the
  visible rect is 64×64, so captures need a real display driver.
- `--headless` combined with `--write-movie` crashed (signal 11) in testing.
- `--editor --script <EditorScript>` fails (`doesn't inherit from SceneTree or
  MainLoop`); editor automation stays on the addon path.
- `--main-loop <name>` did not run a GDScript `class_name` MainLoop.
- Engine progress/error text is localized and ANSI-colored: parse exit codes
  and machine-readable artifacts (`--benchmark-file`, `--log-file`, output
  files), never stdout text.
- `--doctool` requires an existing directory; `--gdscript-docs` requires a
  project-relative path.
- `--export-*` preset names may contain spaces, and `.godot/export_credentials.cfg`
  / `export_presets.cfg` can hold secrets that must never be printed.
- The Windows `_console.exe` wrapper spawns the real engine as a child, so
  bounded runs must kill the process tree.


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
hera-godot classdb info <Class>
hera-godot classdb methods <Class>
hera-godot classdb properties <Class>
hera-godot classdb signals <Class>
hera-godot classdb constants <Class>
hera-godot classdb enums <Class>
hera-godot classdb signals <Class> --own
hera-godot classdb constants <Class> --own
hera-godot classdb enums <Class> --own
hera-godot classdb inherits <Class> <BaseClass>
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

1. Capture observable evidence first: `hera-godot status`, `hera-godot diagnostics`,
   `hera-godot output --type error`, and a minimal reproduction scene.
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
