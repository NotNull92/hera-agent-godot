# Architecture

> `hera-agent-godot` is a low-token CLI that lets an AI coding agent inspect and
> control a **live Godot 4.7+ editor** in real time.

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
 │  hera-agent-godot    │   { "tool": "...", "params": } │   addons/hera_agent_godot/    │
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
- No MCP server is used. Any agent that can run shell commands can use the CLI.

---

## 2. Godot-specific constraints

| # | Godot reality | Design consequence |
|---|---------------|--------------------|
| 1 | GDScript addons are standard Godot addons under `res://addons/<name>/`. | Distribution is just copying `godot/addons/hera_agent_godot/`; no .NET SDK or generated project files. |
| 2 | Editor scripts use `@tool` and run inside the editor. | The plugin entrypoint is `hera_agent_plugin.gd`, extending `EditorPlugin`. |
| 3 | Editor and scene-tree mutation should run on the editor main loop. | Network handling enqueues work; `_process` drains queued requests and calls tools. |
| 4 | Godot's core concepts are Node, Scene, Resource, Signal, and NodePath. | Commands are named `scene`, `node`, `run`, `eval`, and `output`; no Unity vocabulary. |
| 5 | GDScript has native `Expression` support and the best editor integration. | `eval` follows Godot's GDScript path instead of trying to compile another language. |

---

## 3. Repository layout

```text
hera-agent-godot/
├── main.go
├── go.mod
├── cmd/
│   ├── root.go
│   ├── status.go
│   ├── run.go
│   ├── scene.go
│   ├── node.go
│   ├── eval.go
│   └── output.go
├── internal/
│   ├── client/
│   ├── discovery/
│   └── protocol/
├── godot/
│   ├── project.godot
│   └── addons/
│       └── hera_agent_godot/
│           ├── plugin.cfg
│           ├── hera_agent_plugin.gd
│           ├── core/
│           ├── server/
│           └── tools/
└── docs/
    ├── ARCHITECTURE.md
    ├── COMMANDS.md
    └── ROADMAP.md
```

The `godot/` project is a small development host for the addon. Distribution is
the `godot/addons/hera_agent_godot/` folder.

---

## 4. Request lifecycle

```text
1. CLI: hera-agent-godot run --scene res://Main.tscn --wait
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
| `cmd/*` | Parse command flags, build requests, and format responses. |
| `internal/discovery` | Scan `~/.hera-agent-godot/instances/` and return fresh editor instances. |
| `internal/client` | POST one request to one editor instance with timeout and retry. |
| `internal/protocol` | Request / response JSON contract. |

### Godot addon

| Component | Responsibility |
|-----------|----------------|
| `hera_agent_plugin.gd` | `@tool` `EditorPlugin`; owns server, queue, heartbeat, and registry. |
| `server/http_server.gd` | Local HTTP listener bound to `127.0.0.1`, rejecting remote/browser-origin calls. |
| `server/work_queue.gd` | Main-loop handoff for pending HTTP requests. |
| `server/heartbeat.gd` | Writes `~/.hera-agent-godot/instances/<pid>.json`. |
| `core/tool_registry.gd` | Explicit tool name to handler mapping. |
| `core/tool_response.gd` | Compact `{ ok, data/error }` response helpers. |
| `tools/*_tool.gd` | One handler per capability: status, run, scene, node, eval, output. |

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

---

## 7. Security boundaries

- Listener binds only to `127.0.0.1`.
- Browser-origin requests are rejected.
- Instance files live under the user's home directory.
- Dangerous operations are out of scope for v0 and must become explicit named
  tools if ever added.

---

## 8. Deliberate non-goals

- No MCP server.
- No Godot 3.x or pre-4.7 support.
- No C#/.NET addon requirement.
- No reflection-based tool auto-discovery.

See [ROADMAP.md](./ROADMAP.md) for the phased plan and [COMMANDS.md](./COMMANDS.md)
for the command surface as it lands.
