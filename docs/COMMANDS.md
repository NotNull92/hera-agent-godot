# Commands

> Status: implemented command surface. Output is compact by default to stay
> low-token.

Each command maps 1:1 to an addon tool and sends a single JSON request to the
selected editor instance.

| Command | Tool | Status | Description |
|---------|------|--------|-------------|
| `status` | `status` | ☑ | Show the connected editor: project path, Godot version, active scene. |
| `run [--scene <res://...>] [--current] [--wait]` | `run` | ☑ | Play the main scene (default), the current scene (`--current`), or a specific scene (`--scene`). `--wait` polls until the matching runtime scene is inspectable. |
| `stop [--wait]` | `run` | ☑ | Stop the running scene. `--wait` polls until stopped. |
| `output [--type log\|error\|warning\|all] [--lines N]` | `output` | ☑ | Tail the project log file (`user://logs/godot.log`), optionally filtered (`log` excludes error/warning lines). Needs `debug/file_logging` enabled. |
| `diagnostics [--lines N]` | `diagnostics` | ☑ | Summarize project log errors and warnings, returning counts plus the latest matching lines. Needs `debug/file_logging` enabled. |
| `scene tree` | `scene` | ☑ | Print the edited scene's node tree (compact: path/type/name). |
| `scene list` | `scene` | ☑ | List open scenes and the current one. |
| `scene open <res://...>` | `scene` | ☑ | Request opening a scene in the editor. |
| `scene save` | `scene` | ☑ | Save the edited scene. |
| `scene create <res://...> [--root <type>] [--force] [--open]` | `scene` | ☑ | Create a new `.tscn` with an instantiable node root; refuses overwrite unless `--force` is passed. |
| `scene save-as <res://...> [--force]` | `scene` | ☑ | Save the edited scene to a new `.tscn`; refuses overwrite unless `--force` is passed. |
| `editor state` | `editor` | ☑ | Show editor context: current scene, open scenes, main scene, play state, selected nodes, and current script. |
| `editor selected` | `editor` | ☑ | Return the current editor node selection with scene-relative paths when possible. |
| `editor select <node> [--add]` | `editor` | ☑ | Select a node in the edited scene; clears the previous selection unless `--add` is passed. Editor-state mutation only. |
| `editor clear-selection` | `editor` | ☑ | Clear the editor node selection. Editor-state mutation only. |
| `script current` | `script` | ☑ | Inspect the currently focused script in the Godot script editor when it is a readable `.gd` file; otherwise return compact script metadata. |
| `script inspect <res://script.gd>` | `script` | ☑ | Read a GDScript file and return low-token metadata: class name, extends, functions, signals, exports, and line count. |
| `script open <res://script.gd> [--line N] [--column N]` | `script` | ☑ | Open a GDScript resource in the Godot script editor, optionally at a 1-based line/column. Editor-state mutation only. |
| `script create <res://script.gd> [--extends <Class>] [--class-name <Name>] [--force] [--tool] [--ready] [--process] [--physics-process] [--input] [--unhandled-input] [--signal <name> ...] [--export <name:type[=value]> ...]` | `script` | ☑ | Create a GDScript file and refresh the editor filesystem; optional flags add `@tool`, lifecycle stubs, signal declarations, and typed exported variables. |
| `project info` | `project` | ☑ | Show project name, root path, Godot version, current scene, and file counts by type. |
| `project list-files [--type all\|scene\|script\|resource\|asset\|shader] [--pattern <p>] [--limit N]` | `project` | ☑ | List project files from `res://`, with compact type tags and optional filtering. |
| `project scan` | `project` | ☑ | Request a Godot editor resource filesystem scan so newly written files are visible to editor tools. Editor filesystem mutation only. |
| `project reimport <res://file> ...` | `project` | ☑ | Ask Godot to reimport one or more safe `res://` project files through `EditorFileSystem.reimport_files`. Persistent import metadata/cache change. |
| `project mkdir <res://dir>` | `project` | ☑ | Create a project directory under `res://` and refresh the editor filesystem. |
| `project set-main-scene <res://scene.tscn>` | `project` | ☑ | Set `application/run/main_scene` in `project.godot` for the targeted live editor project. |
| `node find [query] [--type <Class>]` | `node` | ☑ | Find nodes by name substring and/or class. |
| `node get <path>` | `node` | ☑ | Dump a node's editor-visible properties. |
| `node add <type> [--parent <path>] [--name <n>]` | `node` | ☑ | Add a node under a parent (undoable). |
| `node instance <res://scene.tscn> [--parent <path>] [--name <n>]` | `node` | ☑ | Instance a PackedScene under a parent after validating the scene path (undoable). |
| `node set <path> --prop <name> --value <v>` | `node` | ☑ | Set a node property (undoable; value coerced to the property's type). |
| `node set-resource <path> --prop <name> --resource <res://...>` | `node` | ☑ | Set an object/resource property from a Resource file, with path and type compatibility checks (undoable). |
| `node remove <path>` | `node` | ☑ | Remove a node (undoable). |
| `node attach-script <path> <res://script.gd>` | `node` | ☑ | Attach a script resource to a node after validating the path and script base type (undoable). |
| `node detach-script <path>` | `node` | ☑ | Clear a node's script (undoable). |
| `signal list <node>` | `signal` | ☑ | List the signals a node exposes (name + arg names) and scene-local connections; editor-internal targets are counted as `external_connections`. |
| `signal connect <from> <sig> <to> <method>` | `signal` | ☑ | Connect a node's signal to a method on another node (undoable; persistent, saved with the scene). |
| `signal disconnect <from> <sig> <to> <method>` | `signal` | ☑ | Remove that connection (undoable). |
| `resource get <res://...>` | `resource` | ☑ | Load a resource (`.tres`/`.res`/`.tscn`/any `res://`) and dump its class, name, and editor-visible properties. Read-only; no scene needs to be open. |
| `resource uid <res://...>` | `resource` | ☑ | Return Godot's resource UID plus the `.uid` sidecar content when present. |
| `resource list [res://dir] [--type <Class>] [--pattern <text>] [--limit N]` | `resource` | ☑ | Recursively list project resources from a safe `res://` path, optionally filtering by resource class, path substring, and result limit. |
| `resource set <res://...> --prop <name=value> ...` | `resource` | ☑ | Load a resource, coerce Godot literal strings to the target property types, set editor-visible properties, and save it back to disk. Persistent filesystem change. |
| `resource create <Class> <res://out.tres> [--force] [--prop <name=value> ...]` | `resource` | ☑ | Create an instantiable `Resource` class, optionally set editor-visible properties using Godot literal strings, and save it as `.tres`/`.res`. |
| `resource resave <res://...>` | `resource` | ☑ | Load and save a resource to refresh serialized data and UID metadata. Persistent filesystem change. |
| `resource update-uids` | `resource` | ☑ | Resave project resources/scripts that Godot can load, useful after migrations that need UID sidecars refreshed. Persistent filesystem change. |
| `resource export-mesh-library <res://scene.tscn> <res://out.tres> [--item <name> ...]` | `resource` | ☑ | Build a `MeshLibrary` from top-level scene children containing `MeshInstance3D` nodes, optionally filtered by item name. |
| `classdb info <Class>` | `classdb` | ☑ | Show ClassDB metadata: parent, instantiability, Node/Resource ancestry. |
| `classdb methods <Class>` | `classdb` | ☑ | List ClassDB methods with compact argument and return type summaries. |
| `classdb properties <Class>` | `classdb` | ☑ | List ClassDB properties with type, class, hint, and hint string. |
| `classdb inherits <Class> <BaseClass>` | `classdb` | ☑ | Check inheritance using Godot ClassDB. |
| `game tree` | `game` | ☑ | Print the running game's live node tree. Requires a play session and the Hera runtime autoload; requests are isolated to the matching game process. |
| `game instances` | `game` | ☑ | List Hera runtime game processes seen by the editor, including pid, scene, and heartbeat age. Useful for stale process diagnosis. |
| `game screenshot [--path <p>] [--analyze]` | `game` | ☑ | Capture the running game viewport to PNG and return the path. `--analyze` adds generic image/layout metrics (`nonblank`, dimensions, sampled color count, brightness, edge content by side, clipping and low-detail hints). |
| `game click --x N --y N` | `game` | ☑ | Send a left mouse click to the running game viewport at pixel coordinates. Runtime-only and useful for surface-level QA. |
| `game node get <path> [--prop <name>\|--props <a,b>]` | `game` | ☑ | Dump a live runtime node's editor-visible properties, or only selected properties for low-token QA. Absolute paths like `/root/Main` are accepted. |
| `game node set <path> --prop <name> --value <v>` | `game` | ☑ | Set a live runtime node property. Runtime-only, not undoable, and lost when play stops. |
| `game node call <path> <method> [--arg <v> ...]` | `game` | ☑ | Call a live runtime node method and return the stringified result. Runtime-only and may have side effects. |
| `game assert <path> <prop> <eq\|ne\|contains\|gt\|lt\|exists> [value]` | `game` | ☑ | Assert a live runtime node property with a compact pass/fail response. Designed for generic QA, not a specific game. |
| `game qa --file <scenario.json> [--continue]` | local + tools | ☑ | Run a generic JSON QA scenario made of `run`, `stop`, `wait`, `game.node.get`, `game.node.set`, `game.node.call`, `game.click`, `game.assert`, `screenshot.runtime`, and `diagnostics` steps; runtime screenshots are analyzed by default and the command returns a compact step summary. |
| `eval <expression>` | `eval` | ☑ | Evaluate one GDScript expression (`Expression` class, scene root as base) and return the result. |
| `instances` | local | ☑ | List all live Hera-enabled Godot editors discovered from `~/.hera-agent-godot/instances/`. |
| `screenshot [--path <p>] [--width N] [--height N] [--transparent] [--runtime] [--analyze]` | `screenshot` | ☑ | Render the edited scene off-screen to PNG, or capture the running game viewport with `--runtime`. `--analyze` is supported for runtime captures and returns generic image/layout metrics, including per-edge content ratios and possible clipping. |
| `batch [--file <p>] [--continue]` | `batch` | ☑ | Run a JSON array of `{tool, params}` (stdin or `--file`) in one request, sequentially, including async tools such as `game` and `screenshot`. |
| `smoke [--run-game\|--skip-game]` | local + tools | ☑ | Run a quick live-editor smoke check. `--run-game` also plays the current scene, checks `game tree`, captures/analyzes a runtime screenshot, then stops. |

> **Note (`run`):** use `project set-main-scene <res://scene.tscn>` when changing
> the main scene from Hera. Newly added scenes can still require a filesystem
> refresh or project reload before the editor resolves them as PackedScenes.

> **Note (mutations):** `node add/instance/set/set-resource/remove`, `node attach-script/detach-script`,
> `scene open/save/create/save-as`, `editor select/clear-selection`, `script open/create`, `resource set/create`, `project mkdir/scan/reimport`,
> `project set-main-scene`,
> `resource resave/update-uids/export-mesh-library`, and `signal connect/disconnect`
> are mutation commands and enforce the single-editor guard. Node and signal
> mutations register with the editor's undo history where Godot exposes UndoRedo;
> file, import, resource, scene, and project setting changes are persistent
> filesystem/project changes. `signal connect` uses `CONNECT_PERSIST`, so the wiring is saved with
> the scene like the editor's "Connect a Signal" dialog. `eval` runs a single
> expression via the `Expression` class (not full GDScript statements), with the
> edited scene root as the base instance. Expressions can call methods with side
> effects and are not registered with UndoRedo. `game node set/call` and `game click` target the
> running game process, so it is not undoable and its effects disappear when play
> stops. Hera assumes one live editor per project; mutation commands enforce that
> precondition unless `--instance <pid>` is passed explicitly.

## Global flags

Global flags go **before** the command (e.g. `hera-agent-godot --ids node find`,
`hera-agent-godot --instance 2840 node add Node2D`).

| Flag | Status | Meaning |
|------|--------|---------|
| `--json` | ☑ | Pretty-print the response Data. |
| `--ids` | ☑ | Print only node paths (for `scene tree` / `node find`); compact JSON otherwise. |
| (default) | ☑ | Compact JSON — minimal tokens. |
| `--instance <pid>` | ☑ | Explicitly target an editor by pid (from `status`); also satisfies the single-editor mutation guard. Accepts `--instance N` or `--instance=N`. |
| `--timeout <ms>` | ☐ | Request timeout. |

See [ARCHITECTURE.md](./ARCHITECTURE.md) for the request lifecycle and
[ROADMAP.md](./ROADMAP.md) for delivery order.
