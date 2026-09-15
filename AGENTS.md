# Working with hera-agent-godot (for AI agents)

`hera` (repo: hera-agent-godot) is a low-token CLI that lets you inspect and
control a **live Godot 4.x editor**. Use it to act on the *real* editor and
check the result — don't guess scene structure or whether a change worked from
memory. The binary installs as `hera`; `hera-agent-godot` is a transitional
alias for the same CLI.

This file is the **driver card**: setup, everyday commands, and safety.
The full flag/response reference is [docs/COMMANDS.md](docs/COMMANDS.md).
If you are changing Hera itself, follow [docs/CO_DEVELOPMENT.md](docs/CO_DEVELOPMENT.md).

For Godot engine/API facts, prefer the official engine and docs repositories
over recollection; Hera docs remain authoritative for Hera contracts.

## When to use it

- You need the actual state of the open scene (node tree, a node's properties).
- You want to change the scene (add/set/remove nodes) and confirm it stuck.
- You want to run a scene, then read the log for errors.
- You want an off-screen preview render of the edited scene (`screenshot`) or a
  live game viewport capture (`screenshot --runtime` / `game screenshot`).

If the user is not running the Godot editor with the **Hera Agent** plugin
enabled, commands fail with "no live Godot editor found" — ask them to enable it.

## Setup (once)

1. Open the project in a **Godot 4.x** editor.
2. Enable **Project → Project Settings → Plugins → Hera Agent Godot**. The Output
   panel should show `[hera] ... listening on 127.0.0.1:<port>`.
3. Get the CLI: install a release binary (see the README's Install section) or
   build from source with `go build -o hera .` (from the repo root).

The CLI finds the editor automatically via `~/.hera-agent-godot/instances/`.

## Commands

Everyday driver surface (full flags and response fields: [docs/COMMANDS.md](docs/COMMANDS.md)):

```
hera status                                  # project, version, scene, session; not the editor console
hera status --capabilities                   # include experimental API capability map
hera instances                               # live editors; stale heartbeats listed separately
hera --ids scene tree                        # edited scene paths
hera node get <path> --prop p                # one property; omit --prop only when you need the dump
hera node add|set|remove ...                 # undoable editor mutations
hera diagnostics [--lines N]                 # project file log; source is file, not the Output panel
hera diagnostics --source editor             # opt-in editor-session errors (capability-gated)
hera run --current --wait / hera stop --wait
hera game ui tree --type Button --fields name,path,text,disabled
hera --ids game ui tree                      # Control paths only
hera guidance ui                             # UI mode before building UI
```

Global flags go **before** the command: `--json` (pretty-print), `--ids` (print
only paths for `scene tree` / `node find` / `game ui tree`), `--instance <pid>`
(explicitly target a pid shown by `status`), `--timeout <ms>` (per-request HTTP
timeout, default 5000). Default output is compact JSON.

## Conventions & safety

- **Choose script language by extension.** `script create` accepts `.gd` or
  `.cs`; optional `--lang gdscript|csharp` must match. C# create/open/attach
  require Godot .NET (`status.csharp_supported` reports editor capability, not
  SDK installation). C# templates use a filename-matching partial class;
  `--class-name`, if supplied, must match that filename. Build and reload the
  assembly before attachment; Hera does not build or generate solution files.
  C# inspect/current metadata comes from the loaded assembly and may be stale;
  empty metadata with `assembly_loaded: false` is not proof of an empty script.
  `script current` cannot observe an external IDE's focused document. Runtime
  QA discovery accepts `qa_*` and `Qa` followed by an uppercase letter, such as
  `QaReady`; call C# methods with their exact case. `eval` stays a GDScript
  expression in both project languages. See [docs/CSHARP_SUPPORT.md](docs/CSHARP_SUPPORT.md).
- **Output is compact by default** to stay low-token. Use `--ids` to get just
  node paths when scanning, `--json` only when you need the full structure.
- **Validate disk code before running it.** `script validate res://file.gd`
  uses the selected editor's engine in a separate bounded process. It does not
  check unsaved editor text or compile C#. Native dependency loading may execute
  initializers; success is not a warning-free guarantee. `--timeout` also bounds
  this process (default 5 seconds). Validation requires explicit `--instance`
  when multiple editors are live.
- **Replay actions on physics frames.** `game input sequence --file events.json`
  accepts 1–128 ordered `{frame,action,pressed}` entries with offsets 0–120.
  It requires existing, unheld InputMap actions, an unpaused tree, time scale 1,
  and physics rate >=60 Hz; held inputs are released on completion/cancellation.
  The runtime deadline is 2.5 seconds. See `docs/COMMANDS.md` for QA integration.
- **UI work reads the live guidance mode first.** Before agent-driven UI work,
  run `hera guidance ui`. If it reports `game_feel_ui_mode: true`, implement
  UI around Game Feel: immediate input feedback, expressive state changes,
  satisfying bounded motion, and runtime visual QA for those effects.
- **Mutations are undoable where Godot exposes editor undo.**
  `node add/instance/set/remove/reparent`, `node attach-script/detach-script`, and
  `signal connect/disconnect` register with the editor's undo history, so the
  user can Ctrl+Z those changes.
- **Run one live editor per project.** Hera is designed for a single active
  Godot editor. Mutation-capable commands (`node add/instance/set/remove/reparent`,
  `node attach-script/detach-script`, `signal connect/disconnect`,
  `scene open/reload/save/create/save-as`, `editor select/clear-selection`, `script open/create`, `resource set/create`, `project mkdir/scan/reimport`,
  `project set-main-scene`, `eval`, `game node set/call`, `game clock` (except a
  snapshot with no flags), `game input`, `smoke --run-game`,
  and `batch`) enforce that by
  refusing to run when several editors are live unless `--instance <pid>` is
  passed explicitly.
- **`eval` is powerful.** It runs one GDScript expression (not statements) with
  the edited scene root as base, so `get_node("X").something()` works — and can
  have side effects. It is **not** registered with undo. Prefer `node set` for
  property changes.
- **Opt-in token auth.** If commands fail with `unauthorized: ...`, the editor
  requires the shared token; the CLI reads it automatically from
  `HERA_AGENT_GODOT_TOKEN` or `~/.hera-agent-godot/token`
  (see [docs/SECURITY.md](docs/SECURITY.md)).
- **GDScript guide authority, low-token mode.**
  [docs/GDSCRIPT_AGENT_GUIDE.md](docs/GDSCRIPT_AGENT_GUIDE.md) is the
  authoritative source and must be followed, but do not reload the whole guide
  mechanically for routine edits. Use this quick gate first, then open the full
  guide only when the change touches syntax/API not covered here, diagnostics
  fail, the guide changed, or you are uncertain:
  - Do not invent syntax; when uncertain, check official Godot docs/engine
    sources or existing code.
  - Use explicit types for function parameters/returns, dynamic API results,
    `Variant`, and untyped `Array`/`Dictionary` reads.
  - Use `:=` only when Godot can infer a concrete non-Variant type.
  - Use GDScript ternaries (`a if condition else b`), never C-style `? :`.
  - Qualify engine constants/enums/flags with their owner, e.g.
    `Control.PRESET_FULL_RECT`.
  - Prefer `and`/`or`/`not`, typed `@onready` or exported node references,
    named signal handlers for normal UI, and `@tool` only when editor-time
    execution is required and guarded.
  - After any GDScript edit, run `godot --headless --path . --check-only` on the
    affected scene or script before calling the work done. If `godot` is not
    available, use Hera diagnostics/run/output as described in the guide.
- **`game node set/call`, `game click`, `game input`, and `game clock` are runtime-only.** They change the running game process,
  are not registered with undo, and are lost when the play session stops. `game clock --pause` sets `SceneTree.paused`; `--step` leaves the tree paused. The runtime inspector keeps processing while paused.
- **Runtime game requests are process-isolated.** If stale Godot game processes
  are still alive, `game instances` shows them and mutation/read requests refuse
  ambiguous targets instead of accepting an old response. Use `game --pid N ...`
  to select a fresh heartbeat explicitly; `--instance` continues to select the
  editor. A missing or expired PID fails instead of falling back to another game.
  Parallel games that share `user://` are flagged with `shared_user_data`;
  `--instance` / `--pid` do not isolate save files. Use a separate user data
  directory (isolated `HOME`/`USERPROFILE`, per [DEV_MACHINE.md](docs/DEV_MACHINE.md))
  for concurrent QA.
- **Runtime captures report the live viewport.** `game screenshot` and
  `screenshot --runtime` return the PNG as captured (`width`/`height`) plus
  window, visible-rect, and project viewport sizes. They do not upscale a
  smaller embedded window to the project resolution. Input coordinates are in
  that actual viewport. `possible_clipping` is not a resolution check.
- **Expired editor heartbeats are not the same as a missing editor.** `hera
  instances` lists stale files separately. `--instance <pid>` against an expired
  heartbeat says so instead of reporting "no live Godot editor found".
- **The runtime inspector is excluded from exports.** The editor plugin removes
  its owned autoload while Godot assembles exported project settings, then
  restores it for editor play. Disabling Hera removes an owned persisted
  autoload, while a same-named user autoload at another path is left alone.
- **Prefer low-token QA reads.** Use `game ui audit`, `game ui tree`, `game node get --prop/--props`,
  `game assert`, `game qa discover`, `screenshot --runtime --analyze`, and
  `game qa --file` before dumping full node properties during automated QA.
  Runtime screenshot analysis
  reports per-edge content ratios and `possible_clipping` so layouts that only
  fail at the viewport boundary are easier to catch. Compare `width`/`height`
  with `project_width`/`project_height` before treating a capture as the
  designed resolution.
- **Tie QA to the user's requirements.** For prompt implementation QA, prefer a
  `game qa --file` object with top-level `requirements` and per-step `covers`
  entries so missing requested behavior fails the scenario instead of being
  buried in prose notes.
- **File, scene, resource, and project setting changes are persistent.** `script create`,
  `resource set/create`, `project mkdir/reimport`, `project set-main-scene`, `scene create`, and
  `scene save-as` write project files; use `--force` only when overwriting is
  intended.
- **Use a single writer for scene files.** Before external `.tscn` edits, stop the
  running game with `hera stop --wait`; after external edits, run
  `hera scene reload [res://Path.tscn]` before saving through the editor.
- **`node set` value** is coerced to the property's type via the engine's own
  `str_to_var`, so pass Godot variant text (the form a `.tscn` stores) for
  complex types: `--value "Vector2(10, 20)"`, `--value "Color(0.3, 0.8, 1, 1)"`,
  and packed arrays as a flat list, e.g.
  `--value "PackedVector2Array(0, 0, 48, 0, 48, 48)"`. A wrong form now fails
  with an example in the error. Object/resource properties are not set this
  way — use `node set-resource <path> --prop <name> --resource res://...`.

## Verify your work (Hera)

Use `--evidence` on script validate, captures, scene save and resource set when
proof provenance matters. Disk validation does not observe buffers/running code;
captures sample separate frames/times from runtime state reads. Capture
`--operation-id` is only caller-supplied correlation; `--runtime-session` guards
the actual incarnation. Save-call success is distinct from disk observation and
selected-property equivalence, with full scene persistence unknown. Evidence-enabled
QA screenshot/get/assert steps fail unavailable evidence. See the linked-evidence
section in `docs/COMMANDS.md`; do not infer rollback after a save failure.

Concurrent mutations and sensitive reads fail `editor_busy` immediately. Status,
operation status/cancel and buffered `--source editor` logs remain available.
Batch children retain parent ownership; a client timeout does not unlock running
work. Wait for known completion before a new mutation, and retain operation IDs
for uncertain outcomes. `import_busy_api` reports external import observability;
older engines without it still guard Hera-owned imports and observed scans.
See `docs/COMMANDS.md` for readiness and ownership limits.

Operation receipts are opt-in and bounded in process. Retain the caller-supplied
session/deadline/nonce ID before submission; after transport failure query or
explicitly resubmit the same ID/input, never automatically repeat a mutation.
Inspect lifecycle/effect/verification/persistence independently. Queued cancellation
does not roll back running work. Old sessions reject submissions; missing history
means unknown effect. See `docs/COMMANDS.md` for supported actions and retention.

For conditional edits, retain the complete `expected` object from
`node get --prop P --snapshot` and send it to `node set --expected JSON
[--verify]`. CLI capability negotiation also protects guarded batch entries.
Conflicts (`session_mismatch`, `state_conflict`, `capability_unavailable`)
fail before mutation. Verification failure/unavailability happens after
mutation and does not roll it back or save the scene. See the bounded type
list and string-valued identity contract in `docs/COMMANDS.md`.

Editor log evidence is opt-in: `output`/`diagnostics --source editor` first
verify `hera status --capabilities` → `capabilities.editor_log_cursor`. File logs remain the default.
Retain the returned cursor for an observed interval; `cursor_expired` requires
an explicit restart from `restart_cursor`. Check `available`, `complete`, and
`dropped_count`; a retained tail is not proof that earlier errors never occurred.
The collector covers only this editor process after registration. Startup
`--log-file` capture remains separate; never relaunch a user's editor for it.

After an edit, **confirm it** instead of assuming:

- After `node add`/`set`: `hera node get <path>` and check the value.
- After structural changes: `hera scene tree` (or `--ids`).
- After `run`: `hera output --type error` (project file log) or
  `hera diagnostics --source editor` when that capability is supported. File-log
  `clean: true` is not a claim about the editor Output panel.
- For UI/visual changes: `hera screenshot` for the edited scene, or
  `hera screenshot --runtime` after `run` for the live game viewport.

Batch a change and its check together when it helps, e.g. pipe a JSON array of
`[{set...}, {get...}]` into `hera batch`.

See [docs/COMMANDS.md](docs/COMMANDS.md) and [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).
For prompt-driven game implementation cycles, follow
[docs/GAME_PROMPT_WORKFLOW.md](docs/GAME_PROMPT_WORKFLOW.md).

## Review correction boundaries

QA scenario diagnostics require readable logs and valid counts; omitted `max_warnings` is unlimited, explicit zero is strict. Node removal and reparent undo preserve descendant ownership. Resource/theme batches validate before mutation, but disk-save failure rollback is not guaranteed. `project set-main-scene` updates and persists through the live editor. Standalone behavioral regressions run with `GODOT_BIN=/absolute/direct/godot bash tests/headless/run.sh`; each uses an isolated project and user directories.
