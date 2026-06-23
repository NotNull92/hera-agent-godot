# Commands

> Status: **skeleton.** Signatures below are the planned surface; checked items
> are implemented. Output is compact by default to stay low-token.

Each command maps 1:1 to an addon tool and sends a single JSON request to the
selected editor instance.

| Command | Tool | Status | Description |
|---------|------|--------|-------------|
| `status` | `status` | ☑ | Show the connected editor: project path, Godot version, active scene. |
| `run [--scene <res://...>] [--current] [--wait]` | `run` | ☑ | Play the main scene (default), the current scene (`--current`), or a specific scene (`--scene`). `--wait` polls until the play session starts. |
| `stop [--wait]` | `run` | ☑ | Stop the running scene. `--wait` polls until stopped. |
| `output [--type log\|error\|warning\|all] [--lines N]` | `output` | ☑ | Tail the project log file (`user://logs/godot.log`), optionally filtered (`log` excludes error/warning lines). Needs `debug/file_logging` enabled. |
| `scene tree` | `scene` | ☑ | Print the edited scene's node tree (compact: path/type/name). |
| `scene list` | `scene` | ☑ | List open scenes and the current one. |
| `scene open <res://...>` | `scene` | ☑ | Request opening a scene in the editor. |
| `scene save` | `scene` | ☑ | Save the edited scene. |
| `node find [query] [--type <Class>]` | `node` | ☑ | Find nodes by name substring and/or class. |
| `node get <path>` | `node` | ☑ | Dump a node's editor-visible properties. |
| `node add <type> [--parent <path>] [--name <n>]` | `node` | ☑ | Add a node under a parent (undoable). |
| `node set <path> --prop <name> --value <v>` | `node` | ☑ | Set a node property (undoable; value coerced to the property's type). |
| `node remove <path>` | `node` | ☑ | Remove a node (undoable). |
| `eval <expression>` | `eval` | ☑ | Evaluate one GDScript expression (`Expression` class, scene root as base) and return the result. |
| `screenshot [--path <p>] [--width N] [--height N] [--transparent]` | `screenshot` | ☑ | Render the edited scene off-screen to a PNG and return the path. Needs a GUI editor (no render under `--headless`); frames from the world origin unless the scene has a camera. |
| `batch [--file <p>] [--continue]` | `batch` | ☑ | Run a JSON array of `{tool, params}` (stdin or `--file`) in one request, sequentially. |

> **Note (`run`):** the `run/main_scene` dev fixture and any newly added scenes
> are read when the project loads. If the editor is already open, reload it
> (Project → Reload Current Project) for `run` (main scene) to pick them up.

> **Note (mutations):** `node add/set/remove` register with the editor's undo
> history, so agent changes are undoable (Ctrl+Z). `eval` runs a single
> expression via the `Expression` class (not full GDScript statements), with the
> edited scene root as the base instance. Expressions can call methods with side
> effects and are not registered with UndoRedo. Mutation commands require exactly
> one live editor instance until `--instance <pid>` targeting is implemented.

## Global flags

Output flags go **before** the command (e.g. `hera-agent-godot --ids node find`).

| Flag | Status | Meaning |
|------|--------|---------|
| `--json` | ☑ | Pretty-print the response Data. |
| `--ids` | ☑ | Print only node paths (for `scene tree` / `node find`); compact JSON otherwise. |
| (default) | ☑ | Compact JSON — minimal tokens. |
| `--instance <pid>` | ☐ | Target a specific editor when several are running. |
| `--timeout <ms>` | ☐ | Request timeout. |

See [ARCHITECTURE.md](./ARCHITECTURE.md) for the request lifecycle and
[ROADMAP.md](./ROADMAP.md) for delivery order.
