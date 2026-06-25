---
name: hyper-mode
description: >-
  Safe implementation workflow for the hera-agent-godot repository. Use when
  modifying Go CLI commands, GDScript addon tools, the HTTP server/heartbeat
  lifecycle, runtime autoloads, command flags/params, output JSON contracts,
  docs for the command surface, release packaging, or any non-trivial bugfix,
  refactor, feature, or release-quality task in this repo. Enforces
  explore-first dependency reading, user-visible plan, implementation,
  Godot-specific preload/@tool/.uid/autoload checks, Go and GDScript validation,
  live Hera smoke testing, artifact cleanup, and documentation synchronization.
---

# hera-agent-godot Hyper Mode

Use this skill for substantial code work in `hera-agent-godot`. Keep changes
grounded in the actual repo shape: Go CLI -> localhost HTTP JSON -> Godot
`@tool` addon -> optional runtime autoload.

## Core Rule

Before editing, read the target files and the connected files that register,
call, parse, serialize, or document the behavior. Do not infer the surface from
memory. After editing, drive the feature through the live Hera CLI whenever a
Godot editor is available.

## 1. Explore First

Read the project rules and the relevant command docs first:

- `AGENTS.md`
- `docs/COMMANDS.md`
- `docs/ARCHITECTURE.md` when changing transport, lifecycle, registry, or addon layout
- `docs/ROADMAP.md` when touching release or phase status

Then read target and connected code:

- Go CLI command changes:
  - `cmd/root.go`
  - the target `cmd/*.go`
  - matching parser tests in `cmd/*_test.go`
  - `cmd/common.go` for discovery, mutation guard, and output formatting
  - `internal/client`, `internal/discovery`, `internal/protocol` if request flow changes
- Godot addon tool changes:
  - `addons/hera_agent_godot/hera_agent_plugin.gd`
  - target `addons/hera_agent_godot/tools/*_tool.gd`
  - `core/tool_registry.gd`, `core/tool_response.gd`
  - `server/http_server.gd`, `server/work_queue.gd`, `server/heartbeat.gd` if lifecycle or transport changes
- Runtime game changes:
  - `addons/hera_agent_godot/tools/game_tool.gd`
  - `addons/hera_agent_godot/runtime/game_inspector.gd`
  - `project.godot` autoload entry
- Docs/help changes:
  - `cmd/root.go` help text
  - `AGENTS.md`
  - `README.md`
  - `README.ko.md`
  - `docs/COMMANDS.md`

Use `rg` for symbols, command names, action names, JSON field names, and paths.
If a changed symbol has callers or consumers, read those consumers before
editing.

## 2. Plan Before Editing

For non-trivial work, state a short plan before editing:

- files to touch
- command or JSON surface being changed
- docs that must sync
- live smoke path
- any persistent project changes, especially `project.godot` autoloads

Do not silently reverse locked design choices: CLI-first transport, explicit
tool registry, compact JSON output, localhost-only HTTP, and one live editor per
project unless `--instance` is supplied.

## 3. Implement With Godot Constraints

Respect these repo-specific constraints:

- GDScript editor code that runs in the editor must use `@tool` when it is a
  standalone script instantiated by the plugin.
- Avoid adding new hard `preload()` dependencies from
  `hera_agent_plugin.gd` unless the file already exists and is stable in the
  addon. A missing preload prevents the whole plugin from parsing.
- For tiny optional tools, prefer keeping them inside an existing loaded script
  over adding a fragile new plugin-entry dependency.
- Keep tool names explicit through `ToolRegistry`; do not add reflection-based
  tool discovery.
- Runtime game inspection goes through `HeraGameInspector` autoload and is not
  undoable.
- Editor scene mutations should use `EditorUndoRedoManager` when practical.
- File and scene creation are persistent filesystem changes; smoke artifacts
  must be cleaned.
- Preserve compact response shapes. If a field changes, update Go tests and docs.
- Never record external research repo names or copied source details in project
  files unless the user explicitly asks.

## 4. Go Discipline

For Go files:

- Add or update parser tests before or with parser changes.
- Keep command files small and focused.
- Run `gofmt` on changed Go files.
- Run:

```powershell
go test ./...
go build ./...
go vet ./...
go test -shuffle=on -count=1 ./...
```

Run `go test -race -shuffle=on -count=1 ./...` when cgo is available. If it
cannot run because cgo or a C compiler is unavailable, report that explicitly.

## 5. Godot Checks

For every changed GDScript file, run Godot `--check-only` with the project path.
Use the installed Godot executable if it is not on `PATH`.

```powershell
& '<Godot.exe>' --headless --path . --check-only --script res://addons/hera_agent_godot/hera_agent_plugin.gd
```

Also check each changed tool/runtime script directly.

Before asking the user to reload the plugin, eliminate parse errors locally.
After reload, check:

```powershell
go run . status
go run . diagnostics --lines 20
go run . smoke --skip-game
```

If output reports a plugin error, fix the error before proceeding.

## 6. Live Smoke

Drive changed features through the real CLI whenever the editor is live.

Typical smoke set:

```powershell
go run . status
go run . instances
go run . diagnostics --lines 20
go run . smoke --skip-game
go run . smoke --run-game
```

For feature-specific smoke:

- Scene changes: create/open/save to a temporary `res://scenes/HeraSmoke*.tscn`,
  then restore the original scene and remove the temp scene.
- Node changes: create a temporary node, inspect it, mutate it, then remove it.
- Script/project changes: create under `res://scripts/hera_smoke_*`, inspect the
  file, then delete `.gd`, `.uid`, and the empty folder.
- Runtime game changes: `run --current --wait`, use `game tree`, `game node get`,
  `game node set`, `game node call`, then `stop --wait`.
- Async batch changes: include an async command such as `game` or `screenshot`
  inside `batch`.

Always finish with diagnostics:

```powershell
go run . diagnostics --lines 20
```

## 7. Cleanup

Remove smoke artifacts before final response:

- temporary `scripts/` files and `.uid`
- temporary `scenes/HeraSmoke*.tscn`
- `.godot/editor/*HeraSmoke*` cache files when created
- `user://hera_game_requests` and `user://hera_game_responses` if empty
- unintended `scenes/Main.tscn` UID or `unique_id` churn

Keep intentional persistent changes, such as the `HeraGameInspector` autoload in
`project.godot`, and call them out.

## 8. Documentation Sync

When command names, flags, params, response shape, or lifecycle behavior changes,
sync:

- `cmd/root.go` help
- `AGENTS.md`
- `docs/COMMANDS.md`
- `README.md`
- `README.ko.md`
- `docs/ARCHITECTURE.md` for transport/lifecycle/layout changes
- `docs/ROADMAP.md` for phase/release status changes

Do not update version numbers or create commits unless the user asks.

## Final Report

Report:

- what changed
- live smoke performed
- static checks performed
- what could not be checked and why
- cleanup performed
- intentional persistent repo changes
