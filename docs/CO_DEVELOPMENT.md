# Co-developing hera-agent-godot

This page is for agents and humans **changing Hera itself** (Go CLI, GDScript
addon, docs, distribution). Agents that only *drive* a live Godot editor should
read [AGENTS.md](../AGENTS.md) and [COMMANDS.md](COMMANDS.md) instead.

Claude Code and Codex collaborate on this repository. Rules for both agents:

- The other agent reconstructs context **only from the repo**: git history and
  docs. Make small commits with descriptive English messages; never leave
  meaningful state only in chat.
- State that lives **outside this repo** — Godot Asset Store submissions and
  the Homebrew tap repo ([NotNull92/homebrew-hera](https://github.com/NotNull92/homebrew-hera))
  — must be recorded under `docs/` (release notes in `docs/releases/` or the
  ROADMAP) whenever it changes.
- `main` is shared: never force-push or rewrite pushed history; rebase
  local-only work.
- The same gates apply regardless of agent: `go build/vet/test` + `gofmt`
  for Go; Godot `--check-only` for GDScript; sync README (EN/KO),
  `docs/COMMANDS.md`, and `docs/ROADMAP.md` when the surface changes, and
  regenerate the contract goldens when `docs/CONTRACT.md` behavior changes.
- Outward-facing actions (store uploads, PRs to third-party repos, version
  bumps/releases) need the user's explicit go-ahead; Asset Store form
  submission is done by the user personally.
- Machine-specific facts (Godot binary path, isolated-smoke pattern, toolchain
  limits, external-form gotchas) live in [DEV_MACHINE.md](DEV_MACHINE.md)
  — read it before local Godot smokes, third-party PRs, or publishes, and
  update it there instead of re-discovering.
- **Ported capabilities must be fully naturalized.** When an idea, architecture,
  or workflow is adapted from an outside tool, what ships is a Hera-native
  capability: named for the Godot/Hera construct it operates on, and justified
  from engine behaviour rather than from "the other tool does it this way".
  Leave no external tool name, no "ported from X" framing, no side-by-side
  comparison tables, and no borrowed taxonomy labels anywhere in the repo —
  docs, skills, and code alike. Re-derive each rule from the Godot fact that
  forces it; if a rule cannot be justified that way, it does not belong.
  Attribution cuts the other way for **copied material**: anything actually
  vendored or licensed keeps its upstream provenance and licence, and published
  standards stay cited (as WCAG is in the `ui-theme-qa` corpus). Never strip a
  credit while keeping copied expression — rewrite the expression genuinely
  instead. Prefer deriving values from Godot's own defaults over vendoring
  someone else's data, so the question does not arise.

## Canonical Godot sources for documentation and review

When verifying or reviewing Godot engine behavior, APIs, CLI flags, version
compatibility, or official documentation, consult the maintained upstream
repositories first:

- Godot engine: [github.com/godotengine/godot](https://github.com/godotengine/godot)
- Official Godot documentation: [github.com/godotengine/godot-docs](https://github.com/godotengine/godot-docs)

Local repository documentation remains authoritative for Hera-specific
contracts and policies. Use the upstream sources above to settle Godot facts;
do not substitute stale recollection or unofficial summaries when they apply.
