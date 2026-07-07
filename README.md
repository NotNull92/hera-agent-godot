<p align="center">
  <img src="docs/assets/hera_godot_logo.png" alt="hera-agent-godot logo" width="420">
</p>

# hera-agent-godot

**English** · [한국어](README.ko.md)

> Hera gives agents eyes, hands, and proof in the live Godot editor.

A **low-token CLI** that lets AI coding agents inspect and control a **live
Godot 4.7+ editor** in real time — read the output/errors, run a scene, walk and
edit the node tree, evaluate GDScript, and more. The agent acts on the *real*
editor and checks the result instead of guessing from stale training data.

**Why a CLI, not MCP?** Godot already has a healthy MCP-addon ecosystem — Hera
makes the opposite bet on purpose. MCP servers pay for breadth in tokens: dozens
to 100+ tool schemas plus verbose JSON responses sit in the agent's context
every turn. Hera delivers **MCP-grade reach over the live editor as a
compact-JSON-by-default CLI** — one command per action, minimal tokens, and it
works with anything that can run a shell command (pipes, `batch`, CI, any
agent), not just MCP clients.

The product identity is intentionally simple: **live editor truth, low-token
control, proof-first QA**. See [docs/IDENTITY.md](docs/IDENTITY.md) for the
language and design principles that keep new features aligned.

Sibling of [`hera-agent-unity`](https://github.com/NotNull92/hera-agent-unity) —
same low-token, shell-native philosophy, **designed for Godot**, not ported.

## Latest release: v0.6.0

`v0.6.0` is the current tagged release and the version prepared for the Godot
Asset Store. The addon upload was completed on **2026-07-06**.

Highlights:

- **Game Feel UI Mode (Beta)** in the editor, persisted in Godot
  `EditorSettings`, plus `hera guidance ui` so agents can read the live mode
  before UI work.
- `status` now reports `game_feel_ui_mode`, making UI-mode checks part of the
  normal low-token status path.
- Runtime UI inspection is narrower and cheaper with
  `game ui tree --path`, `--depth`, `--fields`, `--type`, and `--text`.
- QA workflows can discover runtime `qa_*` helpers with `game qa discover` and
  can fail object-format QA scenarios when declared requirements are not covered
  by executable checks.
- Editor reads are more focused with `node get --prop` and `node get --props`.

Release notes and Asset Store packaging details:
[docs/releases/v0.6.0-asset-store-upload.md](docs/releases/v0.6.0-asset-store-upload.md).

## Low-token, measured

The "MCP-grade reach, fewer tokens" claim — with numbers:

| | Hera (CLI) | Godot MCP servers (~41–155 tools) |
|---|---|---|
| Tool schemas resident **per turn** | **0** | ~4k–31k tok (grows with tool count) |
| Surface the agent loads | one doc, ~1.0k tok — cacheable & flat | full tool list, re-sent each turn |
| Per-action response | compact JSON — `status` ≈48 tok, `node get` ≈186 tok | JSON, often pretty |

Hera figures are **measured** on a live Godot 4.7 editor; the MCP column is an
**estimate** from sampled public Godot MCP tool counts (~41–155 tools) ×
~100–200 tok per tool schema. Method, caveats, and a reproducer:
**[docs/LOW_TOKEN.md](docs/LOW_TOKEN.md)**.

## Command surface

The `v0.6.0` CLI/addon surface includes:
`status`, `instances`, `run`/`stop`, `scene`, `editor`, `script`, `project`,
`classdb`, `node` (read + write), `signal`, `resource` (get/list/set/create), `game`
(runtime inspect + set/call/click + assert + QA + screenshot), `guidance`,
`output`, `diagnostics`, `eval`, `screenshot`,
`batch`, and `smoke`, with
`--json`/`--ids` output modes. See
[docs/COMMANDS.md](docs/COMMANDS.md) for the command reference and
[docs/ROADMAP.md](docs/ROADMAP.md) for release history and Asset Store
packaging status.

## Install

**CLI** — one-liner that fetches the latest release binary:

```sh
# macOS / Linux
curl -fsSL https://raw.githubusercontent.com/NotNull92/hera-agent-godot/main/install.sh | sh
```

```powershell
# Windows (PowerShell)
irm https://raw.githubusercontent.com/NotNull92/hera-agent-godot/main/install.ps1 | iex
```

Set `HERA_VERSION` to pin a tag and `HERA_BIN_DIR` to change the target dir. Or
build from source: `go build -o hera .` (Go 1.25+). Check it with `hera version`.

**Addon** — download `hera-agent-godot-addon.zip` from the
[latest release](https://github.com/NotNull92/hera-agent-godot/releases/latest),
unzip it into your Godot project root (creating `addons/hera_agent_godot/`), and
enable it under **Project → Project Settings → Plugins**.

## How it works

```
Go CLI  ──HTTP /rpc──▶  Godot editor addon (@tool EditorPlugin, GDScript)
 (cmd/, internal/)        (addons/hera_agent_godot/)
        ▲                          │
        └── scans ~/.hera-agent-godot/instances/ ◀── Heartbeat
```

- **CLI** (Go): discovers the editor, sends one compact JSON request per command.
- **Addon** (GDScript): runs a localhost HTTP server, executes each request on the
  editor main thread via `EditorInterface`.

See **[docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)** for the full design,
**[docs/COMMANDS.md](docs/COMMANDS.md)** for the command surface, and
**[docs/ROADMAP.md](docs/ROADMAP.md)** for release history.

## Repository layout

```
addons/hera_agent_godot/  the distributable Godot 4.7+ addon (GDScript)
project.godot, scenes/    dev host project — the CLI's run/save/screenshot target
cmd/                      Go CLI commands (status, instances, run/stop, scene, script, project, classdb, node, signal, resource, game, output, diagnostics, eval, screenshot, batch, smoke)
internal/                 client / discovery / protocol
docs/                     ARCHITECTURE, COMMANDS, ROADMAP, release notes
```

## Requirements

- Go 1.25+ (CLI)
- Godot **4.7+** standard build (addon)

## Sibling: hera-agent-unity

Working in Unity too? [**hera-agent-unity**](https://github.com/NotNull92/hera-agent-unity)
brings the same low-token, shell-native philosophy to the **Unity Editor** — read
console errors, run C#, enter Play Mode, manage GameObjects, build UI, and run
tests, all in compact, agent-friendly output. Across both engines, your agents
get one consistent way to drive each.

## Support

Hera is free and MIT-licensed. If it saves you time, you can support development:

[Join the Discord community](https://discord.gg/QBzEVuYwK)

[![Support on Ko-fi](https://ko-fi.com/img/githubbutton_sm.svg)](https://ko-fi.com/notnull92)

## License

MIT — see [LICENSE](LICENSE).
