# Hera Agent Godot — editor addon

Hera gives agents eyes, hands, and proof in the live Godot editor.

This folder is the **distributable addon** for the current `v1.1.0` baseline. To
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

`v1.1.0` is a minor cut on the v1 contract: GDScript or C# by filename,
`node reparent`, `game clock`, joypad/axis input, `game input sequence`,
`script validate`, isolated `game --pid` targeting, and safer undo/resource
batches. Upgrade the CLI and addon together and fully restart Godot. The
verified Godot range remains **4.2–4.7**.

## Layout

| Path | Role |
|------|------|
| `plugin.cfg` | Addon manifest, points at `hera_agent_plugin.gd`. |
| `hera_agent_plugin.gd` | `@tool` `EditorPlugin`; owns server, queue, heartbeat, registry. |
| `core/` | response helpers, settings, `ToolRegistry`, and the Hera main-screen panel. |
| `server/` | `http_server`, `work_queue`, `heartbeat`. |
| `tools/` | Handlers for status, guidance, game feel, run, scene, editor, script (including validate), project, classdb, node (including reparent), signal, resource, eval, output, diagnostics, screenshot, batch, and the game bridge. |
| `runtime/` | Runtime autoload for live game inspection/control, play clock, UI tree reads, semantic clicks, keyboard/mouse/joypad input, sequences, input logs, assertions, and screenshot analysis during play sessions. |

The entry script uses `@tool`, so it runs inside the editor. Full design and CLI
docs: <https://github.com/NotNull92/hera-agent-godot>.

`game instances` lists fresh runtime heartbeats, flags shared `user://`, and
keeps expired files under `stale`. Use
`hera --instance <EDITOR_PID> game --pid <GAME_PID> ...` to target an external
or parallel runtime; unqualified requests retain editor-play matching and
ambiguity refusal. Runtime screenshots report the live viewport and are not
upscaled. The plugin temporarily removes its owned runtime inspector
autoload during export, so exported project settings do not reference the
development bridge. It restores the autoload for editor play afterward.
