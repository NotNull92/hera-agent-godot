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

```text
1. CLI: hera run --scene res://Main.tscn --wait
2. CLI parses args and builds Request{ tool:"run", params:{...} }
3. discovery scans ~/.hera-agent-godot/instances/ and picks a live editor
4. client posts JSON to http://127.0.0.1:<port>/rpc
5. addon server reads JSON and enqueues a work item
6. hera_agent_plugin.gd drains the queue in _process
7. ToolRegistry resolves the tool and runs it through EditorInterface / SceneTree
8. addon writes Response{ ok, data/error }
9. CLI prints compact output
```

---

## 5. Component responsibilities

### Go CLI

| Component | Responsibility |
|-----------|----------------|
| `cmd/*` | Parse command flags, build requests, run local helper commands (`instances`, `smoke`), and format responses. |
| `internal/discovery` | Scan `~/.hera-agent-godot/instances/` and return fresh editor instances. |
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
| `runtime/game_inspector.gd` | Runtime autoload used by `game tree`, `game ui tree`, `game instances`, `game screenshot`, `game click`, `game node get`, `game node set`, `game node call`, and `game assert` while a play session is running. It writes per-process heartbeats and request/response files so stale game processes cannot answer current requests; semantic clicks can target live `Control` nodes by path or text instead of raw viewport coordinates. |
| `runtime/game_value_codec.gd` | Runtime value serialization and argument/property coercion shared by live `game node get/set/call`. |
| `runtime/game_image_analyzer.gd` | Generic runtime screenshot metrics for low-token visual QA (`nonblank`, dimensions, sampled color count, brightness, per-edge content ratios, asymmetric clipping, and low-detail hints). |
| `runtime/game_assertions.gd` | Generic runtime property assertion comparisons for `game assert` and scenario QA. |

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
  "scene": "res://Main.tscn",
  "ts": 1750636800
}
```

The CLI treats an instance as live only if `now - ts` is within the freshness
window. Stale files from crashed editors are ignored and may be cleaned
opportunistically.

The addon republishes the file by staging it under a temp name and swapping it
in with `DirAccess.rename_absolute`. That swap is atomic on POSIX but **not on
Windows**, where Godot's `DirAccess::rename` removes an existing destination
before `MoveFileW` — so `<pid>.json` is briefly absent on every heartbeat. A CLI
scan landing in that window would otherwise report "no live Godot editor found"
while an editor is running, so discovery rescans once after a short delay when
the first pass comes up empty.

---

> **Known limitation — the log file is shared.** `diagnostics` and `output` read
> `debug/file_logging/log_path` (`user://logs/godot.log`). Every Godot process
> rotates that path on startup, so launching a game takes it over from the
> editor and, with `max_log_files` reached, the editor's own log is eventually
> rotated away. After a play session these tools are therefore likely reading
> the game's log rather than the editor's, and editor-console errors can be
> invisible to them. Godot exposes no "path of my current log", so there is no
> clean fix from the addon side today.

## 7. Security boundaries

- Listener binds only to `127.0.0.1`.
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
