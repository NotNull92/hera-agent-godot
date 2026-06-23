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
| `output [--type log\|error\|warning]` | `output` | ☐ | Read editor output / errors / warnings. |
| `scene tree` | `scene` | ☐ | Print the edited scene's node tree (compact). |
| `scene open <res://...>` | `scene` | ☐ | Open a scene in the editor. |
| `scene save` | `scene` | ☐ | Save the current scene. |
| `node find <query>` | `node` | ☐ | Find nodes by name / type / path. |
| `node add <type> --parent <path> [--name <n>]` | `node` | ☐ | Add a node under a parent. |
| `node set <path> --prop <name> --value <v>` | `node` | ☐ | Set a node property. |
| `node remove <path>` | `node` | ☐ | Remove a node. |
| `eval <gdscript-expression>` | `eval` | ☐ | Evaluate a GDScript expression in the editor and return the result. |

> **Note (`run`):** the `run/main_scene` dev fixture and any newly added scenes
> are read when the project loads. If the editor is already open, reload it
> (Project → Reload Current Project) for `run` (main scene) to pick them up.

## Global flags (planned)

| Flag | Meaning |
|------|---------|
| `--instance <pid>` | Target a specific editor when several are running. |
| `--compact` | Minimal output (default for most commands). |
| `--json` | Raw JSON response (for tooling). |
| `--timeout <ms>` | Request timeout. |

See [ARCHITECTURE.md](./ARCHITECTURE.md) for the request lifecycle and
[ROADMAP.md](./ROADMAP.md) for delivery order.
