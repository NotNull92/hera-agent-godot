# Roadmap

Phased build plan. Each phase is independently testable. The current repository
has the core CLI/addon surface implemented, the v0.8.0 GitHub Release
published, and the v0.8.0 Godot Asset Store upload submitted
(pending store approval as of 2026-07-13). Phases 7–9 chart the
standardization arc from v0.8 to v1.0: contract, distribution, then a
stability declaration — the goal is that agents treat Hera as the default way
to drive Godot from a shell.

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
- [x] v0.6.0 release prep: scoped low-token UI/editor reads, runtime QA helper
      discovery, requirement-covered `game qa` scenarios, prompt-game workflow
      guidance, and Hera identity documentation.
- [x] Godot Asset Store submission for the addon (`v0.6.0`).
- [x] v0.7.0 release prep: Hera Game Feel panel/settings, bundled `game_feel`
      topics, gameplay-wide guidance, runtime input injection/logging, input
      steps in `game qa`, and prompt-game QA findings promoted into reusable
      guidance.
- [x] Godot Asset Store upload `v0.7.0` (2026-07-08).

## Phase 7 — Contract & trust groundwork (v0.8)

Goal: make Hera something other tools can safely build on — a documented,
tested output contract plus an explicit trust story — before asking anyone to
standardize on it.

- [x] `docs/CONTRACT.md`: per-command output contract — JSON fields, exit
      codes, error shapes — with each command marked **stable** or
      **experimental**.
- [x] Contract tests in CI: golden compact-JSON outputs for stable commands so
      an accidental breaking change fails the build
      (`cmd/contract_golden_test.go`; also implemented the previously
      documented-only `--timeout <ms>` global flag).
- [x] Godot support matrix: verify the true minimum editor version
      (spot-check 4.2–4.6), publish the matrix, and run the CI GDScript gate
      on the oldest supported version as well as the newest
      ([SUPPORT_MATRIX.md](./SUPPORT_MATRIX.md): verified minimum 4.2 —
      check-only + plugin load + live `status` on every 4.2–4.6 stable; CI
      gate now runs 4.2-stable and 4.7-stable).
- [x] One name: unify the invocation name on `hera` across release binaries,
      installers, and docs (keep `hera-agent-godot` as a transitional alias).
      Release assets already shipped a `hera` binary; installers now also
      create the alias, and help/docs invoke `hera` everywhere.
- [x] Trust: document the localhost HTTP threat model; add opt-in shared-token
      auth between CLI and addon (still `127.0.0.1`-only)
      ([SECURITY.md](./SECURITY.md); `X-Hera-Token` via
      `~/.hera-agent-godot/token` or `HERA_AGENT_GODOT_TOKEN`, 401 → exit 1).
- [x] Asset Store upload `v0.8.0` (submitted 2026-07-13, pending store
      approval; marked Min `Godot 4.2` / Max `Godot 4.7`, Stable).

## Phase 8 — Reach & agent-side distribution (v0.9)

Goal: put Hera where agents (not just humans) pick their tools, and remove
"does it run in my setup?" friction.

- [x] Package managers: Homebrew tap
      ([NotNull92/homebrew-hera](https://github.com/NotNull92/homebrew-hera)),
      Scoop bucket (in-repo `bucket/hera.json`), and winget manifest
      (`packaging/winget/`, validates locally; winget-pkgs submission
      pending), alongside the existing one-line installers. Per-release
      bump steps: [packaging/README.md](../packaging/README.md).
- [x] Agent harness kits: a Claude Code marketplace/plugin with an auto-invoked
      skill, a Cursor rule template, and a copy-paste `AGENTS.md` snippet
      ([`integrations/`](../integrations/)) — each stays within the ~1k-token
      single-document surface.
- [ ] Thin MCP bridge (`hera mcp`): a minimal stdio server that shells out to
      the CLI with a few coarse tools, so MCP-only clients can adopt Hera
      without Hera abandoning the low-token position.
- [ ] Headless/CI recipe: a documented GitHub Actions workflow that boots a
      headless editor with the addon and runs `smoke` + `game qa` scenarios;
      define which commands earn a supported headless tier.
- [ ] Social proof: demo GIF/video in the README, awesome-godot listing,
      showcase projects, and write-ups with real agent transcripts.
- [ ] Asset Store upload `v0.9.0`.

## Phase 9 — Standard declaration (v1.0)

Goal: declare the contract stable and make depending on Hera boring.

- [ ] Freeze the stable command surface and adopt semver with a written
      deprecation policy (deprecate in a minor, remove no earlier than the
      next major).
- [ ] `v1.0.0` release: 0.x migration notes and the compatibility promise
      stated in the README.
- [ ] Contributor on-ramp: `CONTRIBUTING.md`, issue/PR templates, and a
      labeled starter-issue set.
- [ ] Browsable docs site (GitHub Pages) with the contract and support matrix
      front and center.
- [ ] Asset Store upload `v1.0.0` + announcement posts (Godot forums,
      Discord, sibling Unity repo cross-link).

## Open questions to revisit

- Reflection vs explicit tool registry as the surface grows (currently explicit).
- How far `eval` should go (Expression only vs dynamically-loaded `@tool` GDScript).
- Whether a native HTTP server or StreamPeerTCP gives the cleaner Godot-only transport.
