# Roadmap

Phased build plan. Each phase is independently testable. The skeleton in this
repo corresponds to **Phase 0**.

## Phase 0 — Skeleton (this commit)

- [x] Architecture decided and documented ([ARCHITECTURE.md](./ARCHITECTURE.md)).
- [x] Directory layout for Go CLI + Godot 4.7+ GDScript addon.
- [x] Manifests: `go.mod`, `project.godot`, `plugin.cfg`.
- [x] Stub files with responsibilities + `TODO`s for every component.

## Phase 1 — Walking skeleton (`status` end-to-end)

Goal: `hera-agent-godot status` prints info from a live editor.

- [x] Addon: `http_server.gd` (`TCPServer`/`StreamPeerTCP` on `127.0.0.1`, fallback ports 8770–8785).
- [x] Addon: `work_queue.gd` + `_process` drain (main-loop execution).
- [x] Addon: `heartbeat.gd` writing `~/.hera-agent-godot/instances/<pid>.json`.
- [x] Addon: `status_tool.gd`.
- [x] CLI: `discovery` (scan + freshness), `client.Post`, `status` command. (`go build/vet/test` green.)
- [x] End-to-end test: headless Godot 4.7 editor loads the plugin, `status` returns live project/version/scene over HTTP. _(verified)_

## Phase 2 — Run control

- [x] `run_tool.gd`: play main / current / custom scene, stop, state (via `EditorInterface`).
- [x] CLI `run` / `stop` with `--scene` / `--current` / `--wait`. (`go build/vet/test` green; `parseRunArgs` unit-tested.)
- [x] Dev fixture: `scenes/Main.tscn` + `run/main_scene` so bare `run` works out of the box.
- [ ] End-to-end play verification _(pending: not auto-tested to avoid driving a developer's live editor; see manual steps)_

## Phase 3 — Read the editor

- [x] `output_tool.gd`: tail the project log with type/lines filters (the editor Output panel / `EditorLog` isn't exposed to GDScript, so this reads `user://logs/godot.log`; needs `debug/file_logging`).
- [x] `scene_tool.gd`: `tree` (edited scene node list) + `list` (open scenes).
- [x] `node_tool.gd` (read): `find` (name/class) + `get` (property dump) — compact, capped.
- [x] Verified: `go build/vet/test` green; all addon GDScript passes `--check-only` (caught & fixed a `_get` virtual-signature clash).

## Phase 4 — Mutate the scene

- [x] `node_tool.gd` (write): `add` / `set` / `remove`, all registered with `EditorUndoRedoManager` (Ctrl+Z undoable).
- [x] `scene_tool.gd`: `open` / `save`.
- [x] `eval_tool.gd`: single GDScript expression via the `Expression` class (edited scene root as base instance).
- [x] Verified: `go build/vet/test` green; addon GDScript passes `--check-only`.

## Phase 5 — Polish & DX

- [x] `batch_tool.gd`: run a JSON array of commands in one request (sequential; each mutation sub-command keeps its own undo step). CLI reads stdin or `--file`.
- [x] `screenshot_tool.gd`: capture the 2D/3D editor viewport to PNG (GUI-only). **Known limitation:** in Godot 4.7 the editor viewport texture reads back as 2×2, so real captures currently fail with a clear "too small" error — a working capture path (e.g. capturing a played scene) is TODO.
- [x] Output modes: `--json` (pretty) and `--ids` (node paths only); compact JSON is the default.
- [x] Agent rule files (`AGENTS.md`, `CLAUDE.md`).
- [ ] Installers + Asset Library packaging — deferred until a tagged release / binary distribution pipeline exists.
- [x] Verified: `go build/vet/test` green; addon GDScript passes `--check-only`.

## Open questions to revisit

- Reflection vs explicit tool registry as the surface grows (currently explicit).
- How far `eval` should go (Expression only vs dynamically-loaded `@tool` GDScript).
- Whether a native HTTP server or StreamPeerTCP gives the cleaner Godot-only transport.
