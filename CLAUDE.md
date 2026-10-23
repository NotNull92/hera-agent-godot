# CLAUDE.md

This project ships a CLI (`hera-godot`; `hera-agent-godot` is a transitional alias)
that drives a live Godot 4.x editor. Read the driver card **[AGENTS.md](AGENTS.md)**
first; the full operating rules — setup, commands, conventions, and the
"verify your work" workflow — are in **[docs/AGENT_GUIDE.md](docs/AGENT_GUIDE.md)**,
linked section by section from the card.

Quick reminders:

- Act on the real editor and **check the result**; don't guess scene state.
- Output is compact JSON by default; `--ids` for paths, `--json` to pretty-print.
- `node add/set/remove/reparent`, `node attach-script/detach-script`, and
  `signal connect/disconnect` are undoable. Mutation-capable commands (`scene
  open/save/create/save-as`, `script create`, `project mkdir`, `eval`,
  `game node set/call`, `game clock` (except a snapshot), `game input`,
  `smoke --run-game`, `batch`, and node writes) require
  exactly one live editor.
- After editing, verify with `node get` / `scene tree` / `output --type error`.

Collaboration: Claude Code and Codex **co-develop this repo** (the CLI,
addon, docs, and distribution). The other agent sees only git history and repo
docs — not your chat context. Follow [docs/CO_DEVELOPMENT.md](docs/CO_DEVELOPMENT.md):
small descriptive commits, record out-of-repo state (Asset Store, the
homebrew-hera tap) in `docs/`, and never rewrite pushed history.

Porting: when an idea or architecture is adapted from an outside tool,
**fully naturalize it**. Ship a Hera-native capability — named for the
Godot/Hera construct it operates on, justified from engine behaviour, with no
external tool name, "ported from X" framing, comparison tables, or borrowed
taxonomy labels left in the repo. Material that is genuinely *copied* (vendored
data, licensed source) keeps its upstream provenance instead; never strip a
credit while keeping copied expression. See [docs/CO_DEVELOPMENT.md](docs/CO_DEVELOPMENT.md).

Repo conventions: Go CLI in `cmd/` + `internal/`; the Godot addon (GDScript) in
`addons/hera_agent_godot/`, with the dev host project (`project.godot`,
`scenes/`) at the repo root. Run `go build/vet/test` and `gofmt` for Go;
validate addon scripts with `godot --headless --path . --check-only --script <res://...>`.
For GDScript, follow the low-token quick gate in `AGENTS.md`. The full
`docs/GDSCRIPT_AGENT_GUIDE.md` remains authoritative, but do not reload it
mechanically for routine edits; open it when the quick gate does not cover the
change, diagnostics fail, the guide changed, or syntax/API is uncertain.

## Canonical Godot sources

When documentation or a review needs to verify Godot engine behavior, APIs,
CLI flags, version compatibility, or official guidance, consult these upstream
repositories first:

- Godot engine: [github.com/godotengine/godot](https://github.com/godotengine/godot)
- Official Godot documentation: [github.com/godotengine/godot-docs](https://github.com/godotengine/godot-docs)

This repository's docs remain authoritative for Hera-specific contracts and
policies. Use the upstream repositories to settle Godot facts rather than stale
recollection or unofficial summaries.

## Handing work to Codex (this machine)

"Codex에 전달해" means: send the instruction to the **Codex pane in the herdr
workspace**, not a new `codex exec` process. Find the pane with
`herdr pane list` (agent `codex`, cwd this repo, same `HERDR_WORKSPACE_ID`),
then `herdr pane run <PANE_ID> '<text>'`; the Codex TUI may leave the text in
its input box, so follow with `herdr pane send-keys <PANE_ID> enter` and confirm
with `herdr pane read <PANE_ID>` that the prompt was submitted. Write the durable brief under `docs/handoff/`
first (local-only, gitignored) and point the pane at that file. To give Codex
a fresh conversation (e.g. its context is nearly used up), send `/new` the same
way before the brief.
