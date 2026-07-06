# Hera Agent Godot — editor addon

Hera gives agents eyes, hands, and proof in the live Godot editor.

This folder is the **distributable addon**. To use it in your own project:

1. Use a **Godot 4.7+** build.
2. Copy this entire `hera_agent_godot/` folder into your project's `res://addons/`.
3. Enable **Project → Project Settings → Plugins → Hera Agent Godot**.

The plugin starts a localhost HTTP server and advertises the editor to the
`hera-agent-godot` CLI via `~/.hera-agent-godot/instances/`.

## Layout

| Path | Role |
|------|------|
| `plugin.cfg` | Addon manifest, points at `hera_agent_plugin.gd`. |
| `hera_agent_plugin.gd` | `@tool` `EditorPlugin`; owns server, queue, heartbeat, registry. |
| `core/` | response helpers and `ToolRegistry`. |
| `server/` | `http_server`, `work_queue`, `heartbeat`. |
| `tools/` | Handlers for status, run, scene, node, signal, resource, eval, output, diagnostics, screenshot, batch, and the game bridge. |
| `runtime/` | Runtime autoload for live game inspection/control during play sessions. |

The entry script uses `@tool`, so it runs inside the editor. Full design and CLI
docs: <https://github.com/NotNull92/hera-agent-godot>.
