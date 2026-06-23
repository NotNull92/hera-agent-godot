# CLAUDE.md

This project ships a CLI (`hera-agent-godot`) that drives a live Godot 4.x
editor. Full guidance for using it — setup, commands, conventions, and the
"verify your work" workflow — is in **[AGENTS.md](AGENTS.md)**. Read that first.

Quick reminders:

- Act on the real editor and **check the result**; don't guess scene state.
- Output is compact JSON by default; `--ids` for paths, `--json` to pretty-print.
- `node add/set/remove` are undoable. Mutation-capable commands (`scene
  open/save`, `eval`, `batch`, and node writes) require exactly one live editor.
- After editing, verify with `node get` / `scene tree` / `output --type error`.

Repo conventions: Go CLI in `cmd/` + `internal/`; the Godot addon (GDScript) in
`godot/addons/hera_agent_godot/`. Run `go build/vet/test` and `gofmt` for Go;
validate addon scripts with `godot --headless --check-only --script <path>`.
