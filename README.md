<p align="center">
  <img src="docs/assets/hera_godot_logo.png" alt="hera-agent-godot logo" width="420">
</p>

# hera-agent-godot

**English** · [한국어](README.ko.md)

> Hera gives agents eyes, hands, and proof in the live Godot editor.

<p align="center">
  <img src="docs/assets/hera-agent-godot-playtest_v0.9.0.gif" alt="Hera driving a live Godot editor from the shell: build a scene, run it, and QA the running game" width="820">
</p>
<p align="center">
  <em>Assembled, ran, and QA-tested a live Godot game entirely from the shell — build the scene, run it, read the moving player, inject input and verify the game reacts, recolor a running node. The whole session is <strong>~1,170 tokens</strong> of tool output: compact JSON by default, no tool-schema preload.</em>
</p>

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

## Current release baseline: v0.9.0

`v0.9.0` is the current repository tag and addon manifest baseline. It builds on
the v0.8.0 foundation (Godot 4.2–4.7 support, output contract, opt-in
shared-token auth) and focuses on **reach**: putting Hera where agents pick their
tools, and making a live session easy to prove.

Highlights:

- **Agent-side distribution**: install the CLI via the
  [`hera-godot` npm wrapper](https://www.npmjs.com/package/hera-godot), the
  Homebrew tap, or the in-repo Scoop bucket; drop Hera into an agent with the
  Claude Code and Codex plugins and a Cursor rule that all share one
  auto-invoked, ~1k-token `live-editor` skill
  ([integrations/](integrations/), [plugins/](plugins/)).
- **Value-syntax hints in error messages**: when a `node set`, `game node set`,
  or `resource set` value can't be parsed, Hera now names the expected Godot
  variant text (e.g. `Vector2(x, y)`, a flat `PackedVector2Array(...)`, or
  `node set-resource` for object properties) instead of failing silently.
- **Steadier runtime QA**: the `game qa` lifecycle waits and timeouts were
  stabilized so the run → inspect → QA → stop flow is reliable under CI.
- **Nonvisual CI tier** you can copy: a pinned Godot 4.7 headless lifecycle
  (enabled editor → `smoke` → `game qa` runtime logic) plus a static 4.2/4.7
  script gate ([docs/HEADLESS_CI.md](docs/HEADLESS_CI.md)).
- **Live demo** near the top of this README: a full build → run → QA session
  recorded from the real editor for ~1,170 tokens of tool output.

Release notes and Asset Store packaging details:
[docs/releases/v0.9.0-asset-store-upload.md](docs/releases/v0.9.0-asset-store-upload.md).

## Nonvisual CI (configured tier)

The [nonvisual CI recipe](docs/HEADLESS_CI.md) defines a pinned, **Godot 4.7-only**
nonvisual lifecycle: static script checks stay headless; the live editor and
game run inside an isolated virtual display so the deterministic runtime-logic
scenario can execute. It excludes screenshots, visual UI, renderer output, and
window/input claims, and does not extend live runtime coverage to Godot 4.2–4.6.

Remote GitHub Actions verification passed on 2026-07-13 at commit
[`5c0ba65`](https://github.com/NotNull92/hera-agent-godot/commit/5c0ba6562961a6a11ab581d0f4eab440d34ce008).
The [successful run](https://github.com/NotNull92/hera-agent-godot/actions/runs/29256396824)
includes the nonvisual editor→game lifecycle and its requirement-covered
runtime-logic scenario.

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

The `v0.9.0` CLI/addon surface includes:
`status`, `instances`, `run`/`stop`, `scene`, `editor`, `script`, `project`,
`classdb`, `node` (read + write + resource/script wiring), `signal`, `resource`
(get/uid/list/set/create/resave/update-uids/export-mesh-library), `theme`
(get/set for `Theme` resource items), `game`
(runtime inspect + input + input-log + set/call/click + assert + QA +
screenshot), `guidance`, `game_feel`, `output`, `diagnostics`, `eval`, `screenshot`,
`batch`, and `smoke`, with
`--json`/`--ids` output modes. See
[docs/COMMANDS.md](docs/COMMANDS.md) for the command reference and
[docs/ROADMAP.md](docs/ROADMAP.md) for release history and Asset Store
packaging status.

## Install

**CLI** — via a package manager:

```powershell
# Windows (Scoop)
scoop bucket add hera-agent-godot https://github.com/NotNull92/hera-agent-godot
scoop install hera
```

```sh
# macOS / Linux (Homebrew)
brew install NotNull92/hera/hera
```

```sh
# Any platform with Node.js 18+ (npm)
npm install -g hera-godot
# or zero-install: npx hera-godot status
```

Or a one-liner that fetches the latest release binary:

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
Windows winget distribution is intentionally retired: no winget-pkgs submission
exists or is planned. See the recorded decision in
[`packaging/README.md`](packaging/README.md).

**Addon** — download `hera-agent-godot-addon.zip` from the
[latest release](https://github.com/NotNull92/hera-agent-godot/releases/latest),
unzip it into your Godot project root (creating `addons/hera_agent_godot/`), and
enable it under **Project → Project Settings → Plugins**.

## Agent integrations

Each kit gives an agent one compact Hera workflow instead of a large tool
schema. Install the CLI and enable the addon first.

- **Claude Code:** inside Claude Code, add this repository as a marketplace and
  install the plugin:

  ```text
  /plugin marketplace add NotNull92/hera-agent-godot
  /plugin install hera-godot@hera-agent-godot
  /reload-plugins
  ```

  The `live-editor` skill auto-invokes for Godot editor work; invoke it directly
  as `/hera-godot:live-editor` when desired. To test a local checkout without
  adding a marketplace, run
  `claude --plugin-dir ./integrations/claude-code/hera-godot`.
- **Codex:** inside a terminal, add this repository as a Codex plugin
  marketplace and install the plugin:

  ```text
  codex plugin marketplace add NotNull92/hera-agent-godot
  codex plugin add hera-godot@hera-agent-godot
  ```

  The bundled `live-editor` skill auto-invokes for Godot editor work. To test a
  local checkout, run `codex plugin marketplace add <checkout-dir>` and remove
  it afterwards with `codex plugin marketplace remove hera-agent-godot`.
- **Cursor:** copy
  [`integrations/cursor/hera-godot.mdc`](integrations/cursor/hera-godot.mdc)
  to `<your-project>/.cursor/rules/hera-godot.mdc`. It is an Agent Requested
  project rule, so Cursor loads it when live Godot work is relevant.
- **Other coding agents:** append
  [`integrations/AGENTS.md`](integrations/AGENTS.md) to the target project's
  `AGENTS.md`.

Each agent-facing document stays below the ~1k-token surface budget that
supports Hera's low-token design; Claude Code and Codex share the same
`live-editor` skill.

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
cmd/                      Go CLI commands (status, instances, run/stop, scene, editor, script, project, classdb, node, signal, resource, theme, game, guidance, game_feel, output, diagnostics, eval, screenshot, batch, smoke)
internal/                 client / discovery / protocol
docs/                     ARCHITECTURE, COMMANDS, ROADMAP, release notes, prompt-game guidance
integrations/             compact Claude Code, Cursor, and AGENTS.md harness kits
```

## Requirements

- Go 1.25+ (CLI)
- Godot **4.7+** standard build recommended (addon). Verified minimum is
  **4.2**: the addon loads and answers the CLI on 4.2–4.6 (spot-checked) —
  see [docs/SUPPORT_MATRIX.md](docs/SUPPORT_MATRIX.md).

## Security

The bridge binds `127.0.0.1` only and rejects browser-origin requests.
Optional shared-token auth locks it to clients that know a secret
(`~/.hera-agent-godot/token` or `HERA_AGENT_GODOT_TOKEN`). Threat model and
setup: [docs/SECURITY.md](docs/SECURITY.md).

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
