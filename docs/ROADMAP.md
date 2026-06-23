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

- [ ] `OutputTool`: log + error/warning entries.
- [ ] `SceneTool`: describe edited scene root, list open scenes.
- [ ] `NodeTool` (read): find nodes, dump properties — compact by default.

## Phase 4 — Mutate the scene

- [ ] `NodeTool` (write): add / set-property / remove.
- [ ] `SceneTool`: open / save.
- [ ] `EvalTool`: GDScript `Expression` evaluation.

## Phase 5 — Polish & DX

- [ ] `batch` (atomic multi-command).
- [ ] Screenshot tool (viewport capture).
- [ ] Token-optimized output modes (`--compact`, `--ids`).
- [ ] Installers + Asset Library packaging for the addon source.
- [ ] Agent rule files (`AGENTS.md`, `CLAUDE.md`).

## Open questions to revisit

- Reflection vs explicit tool registry as the surface grows (currently explicit).
- How far `eval` should go (Expression only vs dynamically-loaded `@tool` GDScript).
- Whether a native HTTP server or StreamPeerTCP gives the cleaner Godot-only transport.
