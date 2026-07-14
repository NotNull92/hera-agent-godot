# hera-godot

Low-token CLI that lets AI coding agents inspect and control a **live Godot
4.x editor** over localhost HTTP: scene/node/signal edits (undoable), play
control, screenshots, diagnostics, and runtime game QA with compact JSON
output.

```sh
npm install -g hera-godot
hera status
```

Or run it without installing:

```sh
npx hera-godot status
```

The install downloads the pinned, SHA-256-verified `hera` binary for your
platform (macOS/Linux/Windows, x64/arm64) from the matching
[GitHub release](https://github.com/NotNull92/hera-agent-godot/releases).
`hera-agent-godot` is included as a transitional alias.

## The CLI needs the Godot addon

`hera` talks to the **Hera Agent Godot** editor addon. Install the addon from
the [Godot Asset Store](https://store.godotengine.org/asset/notnull92/hera-agent-godot/)
(or the release's `hera-agent-godot-addon.zip`) and enable it under
**Project Settings → Plugins**, then run `hera status`.

Docs, agent integrations (Claude Code, Codex, Cursor), and the full command
reference: [github.com/NotNull92/hera-agent-godot](https://github.com/NotNull92/hera-agent-godot).
