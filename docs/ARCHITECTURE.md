# Architecture

> `hera` (repo: hera-agent-godot) is a low-token CLI that lets an AI coding
> agent inspect and control a **live Godot 4.7+ editor** in real time.

This is a sibling of [`hera-agent-unity`](https://github.com/NotNull92/hera-agent-unity),
but it is not a port. Godot's scene tree, editor plugin model, and scripting
workflow are different enough that the bridge is designed around Godot-native
concepts.

---

## 1. High-level model

Two processes talk over localhost HTTP:

```text
 ┌─────────────────────┐         HTTP POST /rpc          ┌──────────────────────────────┐
│   Go CLI             │  ─────────────────────────────▶ │  Godot Editor                │
│  hera-agent-godot    │   { "tool": "...", "params": } │   addons/hera_agent_godot/   │
 │                      │ ◀───────────────────────────── │   @tool EditorPlugin         │
 │  cmd/ internal/      │   { "ok": true, "data": ... }  │   GDScript                   │
 └─────────────────────┘                                 └──────────────────────────────┘
          │                                                            │
          │ scans                                                      │ writes every ~0.5s
          ▼                                                            ▼
   ~/.hera-agent-godot/instances/<pid>.json  ◀──── Heartbeat ──────────┘
```

- The CLI is a thin client. It discovers running editors, picks one, and sends a
  single compact JSON request per command.
- `script validate` first resolves the selected editor's engine/project paths,
  then runs a bounded native `--check-only` child process to avoid stale script
  cache results. It captures at most 64 KiB and reports the engine exit status.
- The addon is a GDScript `@tool` `EditorPlugin`. It binds a local HTTP server,
  queues each request, and executes editor work from the editor main loop.
- No MCP server — by design, not for lack of one. Godot's MCP-addon ecosystem is
  active, but MCP pays for breadth in tokens: many tool schemas plus verbose JSON
  sit in the agent's context every turn. Hera delivers comparable editor reach as
  a compact-JSON CLI, so any agent that can run a shell command can use it — not
  only MCP clients.

---

## 2. Godot-specific constraints

| # | Godot reality | Design consequence |
|---|---------------|--------------------|
| 1 | GDScript addons are standard Godot addons under `res://addons/<name>/`. | Distribution is just copying `addons/hera_agent_godot/`; no .NET SDK or generated project files. The addon lives at the repo root so the Asset Library installs it correctly. |
| 2 | Editor scripts use `@tool` and run inside the editor. | The plugin entrypoint is `hera_agent_plugin.gd`, extending `EditorPlugin`. |
| 3 | Editor and scene-tree mutation should run on the editor main loop. | Network handling enqueues work; `_process` drains queued requests and calls tools. |
| 4 | Godot's core concepts are Node, Scene, Resource, Signal, and NodePath. | Commands are named `scene`, `node`, `run`, `eval`, and `output`; no Unity vocabulary. |
| 5 | GDScript has native `Expression` support and the best editor integration. | `eval` follows Godot's GDScript path instead of trying to compile another language. |

---

## 3. Repository layout

```text
hera-agent-godot/
├── addons/
│   └── hera_agent_godot/        # the distributable addon (ships to users)
│       ├── plugin.cfg
│       ├── hera_agent_plugin.gd
│       ├── LICENSE, README.md
│       ├── core/
│       ├── server/
│       └── tools/               # status, run, scene, node, signal, resource, …
├── project.godot               # dev host project (root, so it loads the addon)
├── scenes/                     # dev fixtures (run/save/screenshot target)
├── main.go
├── go.mod
├── cmd/                        # Go CLI commands
├── internal/                   # client / discovery / protocol
├── docs/
└── .gitattributes              # export-ignore keeps the AssetLib zip addon-only
```

The Godot dev project lives at the repo **root** (`project.godot` + `addons/` +
`scenes/`) so it loads the addon during development *and* so the Asset Library —
which installs the repo archive preserving paths — drops `addons/hera_agent_godot/`
straight into a user's project. `.gitattributes` `export-ignore` strips the CLI,
docs, CI, and dev project from that archive, leaving only the addon content. The
Asset Library ZIP should include `addons/hera_agent_godot/LICENSE`; it does not
need a duplicate `LICENSE` at the ZIP download root.

---

## 4. Request lifecycle

WorkQueue owns one session-local gate. `begin` assigns a private execution ID
(or the admitted operation ID) plus editor session; `complete` releases only that
owner. The plugin uses the same async dispatch seam for every tool. Batch passes
an internally bound dispatcher to its children, so they retain parent ownership
without granting client-supplied params bypass authority. Session retirement
clears the gate, retires receipts and prevents further batch children. Delayed
completion cannot respond through a replacement session's server.

Only status and buffered editor logs bypass dispatch ownership; operation
status/cancel/duplicates are answered by admission. All other reads are
conservative because loading resources/scripts can run code. An in-flight tool
keeps ownership while awaiting runtime replies or filesystem work. The scan
worker can clear `is_scanning` before the editor installs its result; pending
processing plus completion events close that gap. Idle scan/import state and
resource availability are checked together, not a fixed delay or one signal.
`is_importing` is feature-detected; see COMMANDS.md for older-engine limitations.

Explicit `operation` requests enter the existing WorkQueue before dispatch.
The queue validates the ID, session, deadline, request digest and supported
mutation, then reserves a bounded receipt. Status/cancel and duplicates are
answered without calling tools. The plugin marks a fresh receipt running before
the setter/async bridge and completes it before responding over HTTP. Losing the
connection therefore does not erase the result or cause a retry.

The shared `core/operation_records.gd` ledger also guards runtime set/call file
deliveries, with the runtime's random session identity checked before admission.
The editor propagates its ID and retains the runtime receipt when available;
lost runtime replies remain outcome unknown, never an automatic resend. Queue
cancellation cannot undo an already-started tool. Both ledgers bound count,
serialized bytes, evidence, execution deadlines and retention; IDs embed immutable
session/deadline fields so dropped history cannot become valid new execution.
These records are not persisted. See the operation contract in COMMANDS.md.

```text
1. CLI: hera run --scene res://Main.tscn --wait
2. CLI parses args and builds Request{ tool:"run", params:{...} }
3. discovery scans ~/.hera-agent-godot/instances/ and picks a live editor
4. client posts JSON to http://127.0.0.1:<port>/rpc
5. addon server reads JSON and enqueues a work item
6. hera_agent_plugin.gd drains the queue in _process
7. ToolRegistry resolves the tool and runs it through EditorInterface / SceneTree
8. addon queues Response{ ok, data/error }; poll sends bounded partial chunks
9. CLI prints compact output
```

---

## 5. Component responsibilities

### Go CLI

| Component | Responsibility |
|-----------|----------------|
| `cmd/*` | Parse command flags, build requests, run local helper commands (`instances`, `smoke`), and format responses. |
| `internal/discovery` | Scan `~/.hera-agent-godot/instances/` and return fresh editor instances, keeping expired heartbeat files separate so a stalled editor is not reported as missing. |
| `internal/client` | POST one request to one editor instance with timeout and retry. |
| `internal/protocol` | Request / response JSON contract. |

### Godot addon

| Component | Responsibility |
|-----------|----------------|
| `hera_agent_plugin.gd` | `@tool` `EditorPlugin`; owns server, queue, heartbeat, registry, and tiny built-in file/project helper tools. |
| `server/http_server.gd` | Local HTTP listener bound to `127.0.0.1`, rejecting remote/browser-origin calls. |
| `server/work_queue.gd` | Main-loop handoff for pending HTTP requests. |
| `server/heartbeat.gd` | Writes `~/.hera-agent-godot/instances/<pid>.json`. |
| `core/tool_registry.gd` | Explicit tool name to handler mapping. |
| `core/tool_response.gd` | Compact `{ ok, data/error }` response helpers. |
| `tools/*_tool.gd` | One handler per capability: status, run, scene, node, signal, resource, eval, guidance, output, diagnostics, screenshot, batch, and game bridge. |
| `runtime/game_inspector.gd` | Runtime autoload used by `game tree`, `game ui tree`, `game ui audit`, `game instances`, `game screenshot`, `game click`, `game input`, `game clock`, `game node get`, `game node set`, `game node call`, and `game assert` while a play session is running. It writes per-process heartbeats and request/response files so stale game processes cannot answer current requests; `game --pid N` selects one fresh process explicitly. Heartbeats include `user_data_dir` and live viewport sizes because Godot's `user://` is per user-data directory, not per process. The autoload uses `PROCESS_MODE_ALWAYS` and wall-clock heartbeats so `game clock --pause` / `Engine.time_scale` do not freeze the inspector. |
| `runtime/game_inspector_export_guard.gd` | Temporarily removes Hera's owned runtime autoload while Godot collects export dependencies and project settings, then restores it for editor play. |
| `runtime/game_ui_auditor.gd` / `runtime/game_ui_audit_checks.gd` | Bounded runtime UI traversal, verdict assembly, and generic Godot `Control` defect checks. Rules derive from live rectangles, clipping, input/focus behavior, minimum sizes, mouse filtering, and sibling geometry. |
| `runtime/game_value_codec.gd` | Runtime value serialization and argument/property coercion shared by live `game node get/set/call`. |
| `runtime/game_image_analyzer.gd` | Generic runtime screenshot metrics for low-token visual QA (`nonblank`, dimensions, sampled color count, brightness, per-edge content ratios, asymmetric clipping, and low-detail hints). |
| `runtime/game_assertions.gd` | Generic runtime property assertion comparisons for `game assert` and scenario QA. |
| `runtime/game_input_sequence.gd` | Bounded action replay on `physics_frame`, buffered input flushing, and held-action cleanup. Clock steps use a selected-phase SceneTreeTimer to pause after node callbacks. |

---

## 6. Discovery & instance files

- Directory: `~/.hera-agent-godot/instances/`.
- One file per running editor: `<pid>.json`.
- Schema:

```json
{
  "pid": 12345,
  "port": 8770,
  "project_path": "/abs/path/to/project",
  "godot_version": "4.7.stable",
  "editor_session_id": "00000000000000000000000000000001",
  "scene": "res://Main.tscn",
  "ts": 1750636800
}
```

The CLI treats an instance as live only if `now - ts` is within the freshness
window. Expired files stay visible as `stale` on `hera instances` so a process
that stopped publishing is distinct from a missing advertisement. They are not
targeted unless a later heartbeat becomes fresh again.

`editor_session_id` is a random 128-bit hex identifier created with the status
tool for each addon startup. Status and heartbeat share it; disabling/restarting
the addon creates a new identity even within the same editor PID. Discovery
preserves it and omits it for legacy heartbeats. It is evidence, not a mutation
guard or authentication credential.

Status also reports the running engine's `godot_commit` and a small
`capabilities` map. Keys ending in `_api` report direct method/class/signal
presence as `supported` or `unsupported`; presence does not prove a working
debugger connection, logger collector, completed import, or rendered frame.
`csharp` checks `CSharpScript`; `dap`, `script_symbol_lookup`, and `dotnet_sdk`
remain `unverified` because status does not perform those behavioral probes.
There is no version-to-capability table and no API dump in responses.

`node_set_guard: supported` advertises the bounded conditional `node set`
implementation, not merely an engine API. The node tool receives the same
session ID as status and heartbeat. Teardown clears the node tool's session
before retiring the registry, invalidating a setter still holding that tool.
CLI preflight retains one selected client,
including for guarded batch entries. Guarded writes use an internal
`set_guarded` action so an old addon replacing the endpoint after preflight
cannot silently treat them as ordinary writes. On the editor main thread, the shared
node setter compares current scene/node identity and typed property value
before undo registration, without an intervening await. Snapshot serialization
roundtrips through the property codec; equality compares typed Variants, never
display strings. Post-set reads check the original target again. Custom
setters are not isolated or rolled back, and node set never saves to disk.
See [COMMANDS](COMMANDS.md#conditional-node-property-changes) for type limits.

The addon republishes the file by staging it under a temp name and swapping it
in with `DirAccess.rename_absolute`. That swap is atomic on POSIX but **not on
Windows**, where Godot's `DirAccess::rename` removes an existing destination
before `MoveFileW` — so `<pid>.json` is briefly absent on every heartbeat. A CLI
scan landing in that window would otherwise report "no live Godot editor found"
while an editor is running, so discovery rescans with four growing delays when
the first pass comes up empty.

HTTP receive and response-write phases each have a five-second deadline.
Responses advance by at most 64 KiB per connection per poll; a slow reader
cannot block the editor loop. Queued asynchronous work keeps its own tool
deadline. Responses arriving after client cleanup are discarded.

Runtime requests are written to a unique temporary file, closed, then renamed
to their final JSON path. The runtime parses a complete request before removing
it for dispatch. Process targeting and response IDs remain mandatory; the CLI
does not automatically retry mutations after a transport failure.

UI summaries, clicks, and viewport-boundary checks share transformed local
Control geometry. Bounds are viewport-local axis-aligned rectangles; semantic
clicks reject controls in another Viewport instead of injecting into the wrong
viewport.

---

> **File-backed `diagnostics` and `output` remain the default.**
> Both read `debug/file_logging/log_path` (`user://logs/godot.log`), and Godot
> never writes that file while running as the editor. From `main/main.cpp`:
>
> ```cpp
> (!log_file.is_empty() || (!project_manager && !editor && GLOBAL_GET("debug/file_logging/enable_file_logging")))
> ```
>
> The `!editor` guard means the `RotatedFileLogger` is not installed at all in
> an editor session — so editor-console messages (plugin errors, `push_warning`,
> parse errors) never reach these tools no matter how `enable_file_logging` is
> set. Confirmed on 4.7: a headless editor with a deliberately broken autoload
> printed `SCRIPT ERROR: Parse Error` to the console while its `user://logs`
> directory was never even created, before or after a clean exit. Game and
> project runs *do* write there, which is what these tools actually cover.
>
> **Workaround.** The first branch of that condition is an escape hatch: launch
> the editor with `--log-file <path>` and the guard is bypassed, so editor
> output is captured separately, including startup before plugin registration.
> Hera does not automatically read this separate path or relaunch the editor.

`--source editor` uses `core/editor_log.gd`, shared by output and diagnostics
and registered once per plugin lifetime. Status and heartbeat use that same
`editor_session_id`. Class/API checks precede compilation of a fixed in-memory
Logger adapter source. No common script statically inherits or types `Logger`
or `ScriptBacktrace`; the existing 4.2 all-addon parse gate still includes the
collector file. API absence yields `unsupported`; compilation/registration or
initial callback failure yields `unverified`. A received registration message
is required before `editor_log_cursor` becomes `supported`.

Callbacks store bounded message/severity/location data into a 1024-event ring
under a mutex. They never inspect editor objects, do file I/O, capture script
variables, or emit logs. Reads snapshot the ring while locked, then filter and
serialize on the editor thread. Removal retires the registration lease under
the storage mutex before clearing data and unregistering. Callbacks hold an
immutable weak sink and lease; a callback already in flight becomes a no-op
after retirement, including after restart. Re-enable creates a new session.
The cursor is the last observed sequence in that session. Invalid or lost
cursors return an explicit expiration plus the oldest available restart point.
This gives an observation interval without claiming causality or recovering
pre-registration/external-process evidence. See [the response contract](COMMANDS.md#editor-log-evidence).

The native callback/threading constraints are documented in the maintained
[Godot Logger class](https://github.com/godotengine/godot/blob/master/doc/classes/Logger.xml).
Local behavior verification covers the Windows 4.7.2 engine recorded in the
baseline; a capability probe is not a cross-platform certification.

## 7. Security boundaries

- Listener binds only to `127.0.0.1`.
- Response serialization escapes JSON-forbidden control characters, including
  ANSI ESC from editor progress logs, while preserving decoded message text.
- Browser-origin requests are rejected.
- Opt-in shared-token auth: when `~/.hera-agent-godot/token` (or
  `HERA_AGENT_GODOT_TOKEN`) is set, `/rpc` requires a matching `X-Hera-Token`
  header (401 otherwise). See [SECURITY.md](./SECURITY.md) for the full
  threat model.
- Instance files live under the user's home directory and contain no secrets.
- Dangerous operations are out of scope for v0 and must become explicit named
  tools if ever added.

---

## 8. Deliberate non-goals

- No MCP server — a deliberate bet on a low-token, shell-native CLI (see §1), not
  an MCP gap: comparable editor reach at a fraction of the per-turn tokens.
- No Godot 3.x support. 4.7 is the fully-QA'd baseline; 4.2 is the verified
  4.x floor (see [SUPPORT_MATRIX.md](./SUPPORT_MATRIX.md)).
- No C#/.NET addon requirement.
- No reflection-based tool auto-discovery.

See [ROADMAP.md](./ROADMAP.md) for the phased plan, [COMMANDS.md](./COMMANDS.md)
for the command surface as it lands, and
[GODOT_EDITOR_ANALYSIS.md](./GODOT_EDITOR_ANALYSIS.md) for the source/API/debug
analysis workflow used instead of Unity-style binary-first inspection.
