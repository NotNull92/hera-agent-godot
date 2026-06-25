# Commands

> Status: implemented command surface. Output is compact by default to stay
> low-token.

Each command maps 1:1 to an addon tool and sends a single JSON request to the
selected editor instance.

| Command | Tool | Status | Description |
|---------|------|--------|-------------|
| `status` | `status` | ☑ | Show the connected editor: project path, Godot version, active scene. |
| `run [--scene <res://...>] [--current] [--wait]` | `run` | ☑ | Play the main scene (default), the current scene (`--current`), or a specific scene (`--scene`). `--wait` polls until the play session starts. |
| `stop [--wait]` | `run` | ☑ | Stop the running scene. `--wait` polls until stopped. |
| `output [--type log\|error\|warning\|all] [--lines N]` | `output` | ☑ | Tail the project log file (`user://logs/godot.log`), optionally filtered (`log` excludes error/warning lines). Needs `debug/file_logging` enabled. |
| `diagnostics [--lines N]` | `diagnostics` | ☑ | Summarize project log errors and warnings, returning counts plus the latest matching lines. Needs `debug/file_logging` enabled. |
| `scene tree` | `scene` | ☑ | Print the edited scene's node tree (compact: path/type/name). |
| `scene list` | `scene` | ☑ | List open scenes and the current one. |
| `scene open <res://...>` | `scene` | ☑ | Request opening a scene in the editor. |
| `scene save` | `scene` | ☑ | Save the edited scene. |
| `scene create <res://...> [--root <type>] [--force] [--open]` | `scene` | ☑ | Create a new `.tscn` with an instantiable node root; refuses overwrite unless `--force` is passed. |
| `scene save-as <res://...> [--force]` | `scene` | ☑ | Save the edited scene to a new `.tscn`; refuses overwrite unless `--force` is passed. |
| `script create <res://script.gd> [--extends <Class>] [--class-name <Name>] [--force]` | `script` | ☑ | Create a GDScript file and refresh the editor filesystem; refuses overwrite unless `--force` is passed. |
| `project mkdir <res://dir>` | `project` | ☑ | Create a project directory under `res://` and refresh the editor filesystem. |
| `node find [query] [--type <Class>]` | `node` | ☑ | Find nodes by name substring and/or class. |
| `node get <path>` | `node` | ☑ | Dump a node's editor-visible properties. |
| `node add <type> [--parent <path>] [--name <n>]` | `node` | ☑ | Add a node under a parent (undoable). |
| `node set <path> --prop <name> --value <v>` | `node` | ☑ | Set a node property (undoable; value coerced to the property's type). |
| `node remove <path>` | `node` | ☑ | Remove a node (undoable). |
| `node attach-script <path> <res://script.gd>` | `node` | ☑ | Attach a script resource to a node (undoable). |
| `node detach-script <path>` | `node` | ☑ | Clear a node's script (undoable). |
| `signal list <node>` | `signal` | ☑ | List the signals a node exposes (name + arg names) and scene-local connections; editor-internal targets are counted as `external_connections`. |
| `signal connect <from> <sig> <to> <method>` | `signal` | ☑ | Connect a node's signal to a method on another node (undoable; persistent, saved with the scene). |
| `signal disconnect <from> <sig> <to> <method>` | `signal` | ☑ | Remove that connection (undoable). |
| `resource get <res://...>` | `resource` | ☑ | Load a resource (`.tres`/`.res`/`.tscn`/any `res://`) and dump its class, name, and editor-visible properties. Read-only; no scene needs to be open. |
| `game tree` | `game` | ☑ | Print the running game's live node tree. Requires a play session and the Hera runtime autoload. |
| `game node get <path>` | `game` | ☑ | Dump a live runtime node's editor-visible properties. Absolute paths like `/root/Main` are accepted. |
| `game node set <path> --prop <name> --value <v>` | `game` | ☑ | Set a live runtime node property. Runtime-only, not undoable, and lost when play stops. |
| `game node call <path> <method> [--arg <v> ...]` | `game` | ☑ | Call a live runtime node method and return the stringified result. Runtime-only and may have side effects. |
| `eval <expression>` | `eval` | ☑ | Evaluate one GDScript expression (`Expression` class, scene root as base) and return the result. |
| `instances` | local | ☑ | List all live Hera-enabled Godot editors discovered from `~/.hera-agent-godot/instances/`. |
| `screenshot [--path <p>] [--width N] [--height N] [--transparent]` | `screenshot` | ☑ | Render the edited scene off-screen to a PNG and return the path. Needs a GUI editor (no render under `--headless`); frames from the world origin unless the scene has a camera. |
| `batch [--file <p>] [--continue]` | `batch` | ☑ | Run a JSON array of `{tool, params}` (stdin or `--file`) in one request, sequentially, including async tools such as `game` and `screenshot`. |
| `smoke [--run-game\|--skip-game]` | local + tools | ☑ | Run a quick live-editor smoke check. `--run-game` also plays the current scene, checks `game tree`, then stops. |

> **Note (`run`):** the `run/main_scene` dev fixture and any newly added scenes
> are read when the project loads. If the editor is already open, reload it
> (Project → Reload Current Project) for `run` (main scene) to pick them up.

> **Note (mutations):** `node add/set/remove`, `node attach-script/detach-script`,
> `scene open/save/create/save-as`, `script create`, `project mkdir`, and
> `signal connect/disconnect`
> register with the editor's undo history, so agent changes are undoable
> (Ctrl+Z) where Godot exposes UndoRedo for that operation. File and scene
> creation create project assets and should be treated as persistent filesystem
> changes. `signal connect` uses `CONNECT_PERSIST`, so the wiring is saved with
> the scene like the editor's "Connect a Signal" dialog. `eval` runs a single
> expression via the `Expression` class (not full GDScript statements), with the
> edited scene root as the base instance. Expressions can call methods with side
> effects and are not registered with UndoRedo. `game node set/call` targets the
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
