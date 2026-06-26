# CLAUDE.md

This project ships a CLI (`hera-agent-godot`) that drives a live Godot 4.x
editor. Full guidance for using it — setup, commands, conventions, and the
"verify your work" workflow — is in **[AGENTS.md](AGENTS.md)**. Read that first.

Quick reminders:

- Act on the real editor and **check the result**; don't guess scene state.
- Output is compact JSON by default; `--ids` for paths, `--json` to pretty-print.
- `node add/set/remove`, `node attach-script/detach-script`, and
  `signal connect/disconnect` are undoable. Mutation-capable commands (`scene
  open/save/create/save-as`, `script create`, `project mkdir`, `eval`,
  `game node set/call`, `smoke --run-game`, `batch`, and node writes) require
  exactly one live editor.
- After editing, verify with `node get` / `scene tree` / `output --type error`.

Repo conventions: Go CLI in `cmd/` + `internal/`; the Godot addon (GDScript) in
`addons/hera_agent_godot/`, with the dev host project (`project.godot`,
`scenes/`) at the repo root. Run `go build/vet/test` and `gofmt` for Go;
validate addon scripts with `godot --headless --path . --check-only --script <res://...>`.
Before writing or editing GDScript, read `docs/GDSCRIPT_AGENT_GUIDE.md`; it
captures Godot-specific syntax rules for ternaries, type inference, `Variant`,
typed containers, and owner-qualified engine constants.
