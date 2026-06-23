# hera-agent-godot

**English** · [한국어](README.ko.md)

> Let's go Hera, now in Godot.

A **low-token CLI** that lets AI coding agents inspect and control a **live
Godot 4.7+ editor** in real time — read the output/errors, run a scene, walk and
edit the node tree, evaluate GDScript, and more. The agent acts on the *real*
editor and checks the result instead of guessing from stale training data.

Sibling of [`hera-agent-unity`](https://github.com/NotNull92/hera-agent-unity) —
same low-token, shell-native philosophy, but **designed for Godot**, not ported.
(There is no official Godot MCP server; this is a CLI by design.)

## Status

🚧 **Phase 0 — skeleton.** Architecture and directory layout are in place; the
implementation lands per [docs/ROADMAP.md](docs/ROADMAP.md).

## How it works

```
Go CLI  ──HTTP /rpc──▶  Godot editor addon (@tool EditorPlugin, GDScript)
 (cmd/, internal/)        (godot/addons/hera_agent_godot/)
        ▲                          │
        └── scans ~/.hera-agent-godot/instances/ ◀── Heartbeat
```

- **CLI** (Go): discovers the editor, sends one compact JSON request per command.
- **Addon** (GDScript): runs a localhost HTTP server, executes each request on the
  editor main thread via `EditorInterface`.

See **[docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)** for the full design,
**[docs/COMMANDS.md](docs/COMMANDS.md)** for the command surface, and
**[docs/ROADMAP.md](docs/ROADMAP.md)** for the build plan.

## Repository layout

```
cmd/         Go CLI commands (status, run, scene, node, eval, output)
internal/    client / discovery / protocol
godot/       dev Godot 4.7+ project + the addon (godot/addons/hera_agent_godot)
docs/        ARCHITECTURE, COMMANDS, ROADMAP
```

## Requirements (target)

- Go 1.25+ (CLI)
- Godot **4.7+** standard build (addon)

## License

MIT — see [LICENSE](LICENSE).
