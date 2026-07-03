# Roadmap

Phased build plan. Each phase is independently testable. The current repository
has the core CLI/addon surface implemented and v0.5.0 release preparation in
progress; remaining work is Asset Store submission.

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
- [x] End-to-end play verification via live `smoke --run-game`.

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
- [x] `screenshot_tool.gd`: render the edited scene into an off-screen `SubViewport` and save a PNG (the editor's own viewport texture is a placeholder in 4.7). Runs async via one SceneTree frame (headless-safe — returns "empty image" rather than hanging; a GUI editor is needed for an actual render). Frames from origin unless the scene has a camera.
- [x] Output modes: `--json` (pretty) and `--ids` (node paths only); compact JSON is the default.
- [x] Agent rule files (`AGENTS.md`, `CLAUDE.md`).
- [x] `diagnostics`: summarize Godot log errors/warnings.
- [x] `instances`: list live Hera-enabled editors.
- [x] `smoke`: live editor smoke check, optionally including a play-session game check.
- [x] `script create`, `project info`, `project list-files`, and `project mkdir`: project file helpers.
- [x] `scene create` and `scene save-as`: scene file helpers.
- [x] `node set-resource`, `node attach-script`, and `node detach-script`: resource/script wiring helpers.
- [x] `game tree`, `game node get`, `game node set`, `game node call`: runtime inspection and control through the `HeraGameInspector` autoload.
- [x] `batch` awaits async tools such as `game` and `screenshot`.
- [x] Verified: `go build/vet/test` green; addon GDScript passes `--check-only`.

## Phase 6 — Distribution & CI

- [x] CI (`.github/workflows/ci.yml`): Go build/vet/test + `gofmt` gate, and
      GDScript `--check-only` over the addon on a real Godot 4.7 headless build.
- [x] `--instance <pid>` targeting so commands (and mutations) can pick one
      editor when several are live.
- [x] `version` command + linker-injected version string.
- [x] `signal` command: `list` a node's signals + connections; `connect` /
      `disconnect` (undoable, `CONNECT_PERSIST`).
- [x] `resource get`, `uid`, `resave`, `update-uids`, and
      `export-mesh-library`: inspect and refresh resource metadata or build a
      `MeshLibrary` from scene meshes.
- [x] `classdb info|methods|properties|signals|constants|enums|inherits`:
      inspect Godot ClassDB without loading tool schemas into the agent context.
- [x] Release pipeline (`.github/workflows/release.yml`): on a `v*` tag,
      cross-compile the CLI (linux/darwin/windows × amd64/arm64), package the
      addon zip + checksums, and publish a GitHub release.
- [x] One-line installers (`install.sh`, `install.ps1`) that fetch the latest
      release binary.
- [x] Tagged release published and assets generated (`v0.2.0` release train).
- [x] v0.3.0 release prep: runtime QA surface, ClassDB/project/resource helpers,
      race-test tooling, and live `smoke --run-game` cleanup hardening.
- [x] v0.4.0 release prep: main-scene project setting, actionable launch
      diagnostics, expanded GDScript agent guidance, runtime viewport click QA,
      and richer screenshot analysis signals.
- [x] v0.5.0 release prep: editor selection/script inspection helpers, resource
      list/set/create workflows, project scan/reimport commands, PackedScene
      instancing, and Asset Library packaging rules for addon-local licensing.
- [ ] Asset Store submission for the addon.

## Open questions to revisit

- Reflection vs explicit tool registry as the surface grows (currently explicit).
- How far `eval` should go (Expression only vs dynamically-loaded `@tool` GDScript).
- Whether a native HTTP server or StreamPeerTCP gives the cleaner Godot-only transport.
