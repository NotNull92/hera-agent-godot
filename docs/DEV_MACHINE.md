# Development machine & workflow notes

Durable facts about the primary development PC (Windows 11) and hard-won
workflow lessons that both co-developing agents (Claude Code and Codex) need
but that live outside the code. Update this file when a fact changes instead
of re-discovering it; date the entries that are point-in-time observations.

## Godot on this machine

- **Binary**: Godot 4.7 stable lives at
  `C:\Users\PC\Downloads\Godot_v4.7-stable_win64.exe\Godot_v4.7-stable_win64_console.exe`
  (note the nested folder with the same name). It is **not on PATH**. Always
  use the `_console.exe` variant so stdout/stderr is captured in shells.
- **Console-wrapper gotcha**: `Godot_v4.x_win64_console.exe` is a ~200KB
  wrapper that spawns the real editor as a **child process**. The Hera
  heartbeat lands under the **child pid**, not the wrapper pid returned by
  `Start-Process`. Detect an instance by snapshotting
  `~/.hera-agent-godot/instances/` before launch and waiting for a new fresh
  `<pid>.json`; kill both child and wrapper by their specific PIDs.
- The user usually has a **live editor open on this repo**. Never use a broad
  `taskkill //IM` (it would kill the user's editor) — only PID-specific
  `taskkill //PID <pid> //F` against instances you started.

## Isolated live-smoke pattern

To smoke the addon end-to-end without touching the user's editor:

1. Copy the project (`project.godot` + `addons/` + `scenes/`) to a temp dir.
   A minimal copy (bare `project.godot` with `[editor_plugins]` enabling the
   addon + `addons/hera_agent_godot/`) is enough to boot the bridge and
   answer `status` — this is how the 4.2–4.6 rows of
   [SUPPORT_MATRIX.md](./SUPPORT_MATRIX.md) were verified.
2. Run Godot with a **separate `USERPROFILE`/`HOME`** so its heartbeat writes
   to an isolated `~/.hera-agent-godot/instances/` (CLI discovery uses the
   home dir, so this fully isolates discovery).
3. Launch `--editor --headless --path <tempcopy>`, wait for the heartbeat
   JSON, and read the isolated editor's pid from the filename.
4. Drive it with `hera --instance <pid>`, running the CLI under the same
   isolated `USERPROFILE`/`HOME`.
5. Tear down with PID-specific kills only, and remove the heartbeat file
   named after the **child** pid.

The remote/CI variant of this lifecycle is [HEADLESS_CI.md](./HEADLESS_CI.md).

## Headless Godot cannot surface GDScript warnings

GDScript *analyzer warnings* (`UNUSED_PARAMETER`, `SHADOWED_VARIABLE`, …)
appear only in the GUI editor's Output/Errors panel. Headless does **not**
print them to stdout — confirmed on 4.7 via `--check-only --script`, full
`--editor --headless`, `debug/gdscript/warnings/exclude_addons=false`, and
`treat_warnings_as_errors=true`, even with a deliberately injected unused-var
control. So a "0 warnings" audit must be **static** (scan for unused
non-`_`-prefixed params, unused locals, integer division, member shadowing)
or a manual GUI check. `--check-only` *does* catch parse/type errors —
including the virtual-signature clash when a custom tool method is named
`_get`/`_set` (use `_describe`/`_set_property` instead).

## Toolchain limits on this PC

- **`go test -race` cannot run**: the antivirus blocks race-instrumented
  binaries. Run the full non-race suite locally and note the skip; race runs
  happen in CI.
- **Long Windows paths poison clones**: cloning a large third-party repo can
  hit `Filename too long` checkout failures, after which a "one-line" commit
  silently **deletes every unmaterialized file** (observed: 2,410 deletions;
  the push succeeds). For small edits to repos we don't develop on, skip the
  clone: create a branch ref via `POST /git/refs` from the target's main SHA,
  edit with `gh api -X PUT /repos/<fork>/contents/<file> --input payload.json`
  (large base64 must go via `--input`, and beware Windows Python resolves
  `/tmp` to `C:\tmp` — use cwd-relative paths), then verify with
  `GET /compare/main...branch` that the diff is exactly the intended files
  before opening the PR.
- **Env `GITHUB_TOKEN` has only `repo` scope** (the keyring `gh` account has
  more). `gh pr create` works; `gh pr edit` (GraphQL) fails on missing
  `read:org` — use REST (`gh api -X PATCH .../pulls/N`) instead.
- **`npm publish` needs the user's own terminal**: the npm account
  (`notnull92`) has 2FA, publish triggers EOTP, and npm masks the
  `npmjs.com/auth/cli/<token>` URL as `***` in every non-TTY stream including
  its own debug log. Ask the user to run `npm publish` in PowerShell (browser
  auth works there) or to pass `--otp=<code>`. Package + bump steps:
  [packaging/README.md](../packaging/README.md).

## Web-form gotchas (external listings & stores)

- **Godot Asset Store manage page** serves stale cached tab content after
  edits — hard-refresh (F5) before trusting what the form shows. Canonical
  store copy and live form structure: the current release's
  `docs/releases/v<ver>-asset-store-upload.md`.
- **awesome-claude-code** accepts submissions **only** through its web-UI
  issue form (gh-CLI submissions risk a repo ban); prefill works via
  `issues/new?template=recommend-resource.yml&<field_id>=...` query params.
- **codex-marketplace.com** accepts only a repository-root plugin or a
  direct `plugins/<name>` path as the submission target — which is why the
  Codex plugin lives at `plugins/hera-godot/` instead of under
  `integrations/`.
