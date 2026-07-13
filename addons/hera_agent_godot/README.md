# Hera Agent Godot — editor addon

Hera gives agents eyes, hands, and proof in the live Godot editor.

This folder is the **distributable addon** for the current `v0.8.0` baseline. To
use it in your own project:

1. Use any **Godot 4.2–4.7** stable build (4.7 recommended — it gets the full
   QA treatment; see the repo's `docs/SUPPORT_MATRIX.md`).
2. Copy this entire `hera_agent_godot/` folder into your project's `res://addons/`.
3. Enable **Project → Project Settings → Plugins → Hera Agent Godot**.

The plugin starts a localhost HTTP server and advertises the editor to the
`hera` CLI via `~/.hera-agent-godot/instances/`. Optional
shared-token auth: put a random string in `~/.hera-agent-godot/token` (or set
`HERA_AGENT_GODOT_TOKEN`) and reload the plugin — see the repo's
`docs/SECURITY.md`.

`v0.8.0` widens verified Godot support to **4.2–4.7** and adds opt-in
shared-token auth, a documented CLI output contract backed by golden tests,
and quieter startup in user projects — on top of the v0.7.0 surface: the Hera
main-screen panel with Game Feel Mode controls, `guidance ui`,
`guidance game-feel`, bundled `game_feel` topics, scoped runtime UI reads,
runtime input injection, input diagnostics, deterministic QA helper discovery,
and requirement-covered QA scenarios.

## Layout

| Path | Role |
|------|------|
| `plugin.cfg` | Addon manifest, points at `hera_agent_plugin.gd`. |
| `hera_agent_plugin.gd` | `@tool` `EditorPlugin`; owns server, queue, heartbeat, registry. |
| `core/` | response helpers, settings, `ToolRegistry`, and the Hera main-screen panel. |
| `server/` | `http_server`, `work_queue`, `heartbeat`. |
| `tools/` | Handlers for status, guidance, game feel, run, scene, editor, script, project, classdb, node, signal, resource, eval, output, diagnostics, screenshot, batch, and the game bridge. |
| `runtime/` | Runtime autoload for live game inspection/control, UI tree reads, semantic clicks, input injection, input logs, assertions, and screenshot analysis during play sessions. |

The entry script uses `@tool`, so it runs inside the editor. Full design and CLI
docs: <https://github.com/NotNull92/hera-agent-godot>.
