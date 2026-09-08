# Development machine & workflow notes

Durable facts about the primary development PC (Windows 11) and hard-won
workflow lessons that both co-developing agents (Claude Code and Codex) need
but that live outside the code. Update this file when a fact changes instead
of re-discovering it; date the entries that are point-in-time observations.

## Godot on this machine

- **Additional active editor (2026-09-07)**: Godot 4.7.2 .NET is installed at
  `C:\Users\PC\Desktop\Godot_v4.7.2-stable_mono_win64\Godot_v4.7.2-stable_mono_win64.exe`.
  The user's open project in this session was the sibling `testproject`, not
  this repository. Always inspect `hera status` before assuming the target.

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
   home dir, so this fully isolates discovery). The same split is what isolates
   `user://` saves: `--instance` and `game --pid` select processes, they do
   not give each runtime its own save file. On Windows the user data directory
   lives under `%APPDATA%\Godot\app_userdata\<app>`; on macOS under
   `~/Library/Application Support/Godot/app_userdata/<app>`. Confirm the path
   from `game instances[].user_data_dir` rather than assuming the OS default.
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

## Toggling the plugin does not reload changed addon scripts

Disabling and re-enabling the plugin in Project Settings > Plugins is **not**
enough to pick up edits to the addon's GDScript. `hera_agent_plugin.gd` holds
its tools through `const … = preload(…)`, so re-enabling re-runs `_enter_tree`
against script resources Godot still has cached, and the old code keeps
answering. **Quit and relaunch the editor** after changing anything under
`addons/hera_agent_godot/`.

Observed on 4.7: after merging a change to `diagnostics_tool.gd` and toggling
the plugin, the on-disk source read `get_setting_with_override(...)` while
`hera diagnostics` still returned the pre-change `file_logging_enabled:false`,
and `output` still used the old response shape.

Worth checking before you trust a live smoke: compare something the change
actually alters — a field's value or the response shape — against the source on
disk. If they disagree, the editor is still running the old build and the smoke
proves nothing.

## Editor console output needs `--log-file`

`hera diagnostics` / `hera output` cannot see the editor's console. Godot skips
installing the file logger entirely when running as the editor (`!editor` guard
in `main/main.cpp`), so `debug/file_logging` only ever captures game and project
runs. Turning that setting on does nothing for editor messages.

To actually capture them, launch the editor with an explicit log file, which
bypasses the guard:

```powershell
& '<Godot.exe>' --editor --path . --log-file "$env:TEMP\hera-editor.log"
```

Verified on 4.7: a headless editor with a deliberately broken autoload printed
`SCRIPT ERROR: Parse Error` to the console while its `user://logs` directory was
never created — before or after a clean exit, so it is the guard and not output
buffering.

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

## macOS / Monstel integration record (2026-09-07)

A separate macOS 26.6.2 machine used Godot 4.7.2 .NET at
`/Users/admin/Downloads/Godot_mono.app/Contents/MacOS/Godot` and the installed
`/Users/admin/.local/bin/hera` (`--version` reports `dev`). Android checks used
a Galaxy Z Flip7 / Android 16. These facts do not replace the Windows setup above.

[Monstel incident handoff](incidents/monstel-2026-09-07/README.md) records external
runtime targeting failures, ambiguous game processes, embedded viewport sizing,
an editor heartbeat stall, Android autoload/export errors, and shared-save risks.
H01/H02/H05 shipped as runtime PID selection and export autoload exclusion.
H03/H04/H06/H07 follow-up reports live viewport sizes without upscaling,
distinguishes expired vs missing editor heartbeats, flags shared `user://`,
and treats export-exit `EditorSettings` / ObjectDB messages as engine teardown
unless a with/without-Hera comparison shows a delta. Reproduce remaining
environment-specific questions on isolated project copies and isolated user data.

## C# support verification on macOS (2026-09-07)

- The installed .NET SDK is `10.0.201` (`dotnet` on PATH). The existing
  `Godot_v4.7.2-stable_mono_macos.universal.zip` in Downloads was extracted to
  a temporary directory for isolated smoke checks. Standard Godot 4.7.2 and
  4.2 were also checked; do not assume these temporary binaries remain.
- Godot's .NET archive includes a local NuGet feed under
  `Contents/Resources/GodotSharp/Tools/nupkgs`; use a temporary NuGet config
  to consume matching engine packages without changing the user's feeds.
- Resolve macOS project paths with `pwd -P` before building. Building the
  same fixture as `/tmp/...` produced a script path attribute containing
  `res://../../private/tmp/...`; rebuilding as `/private/tmp/...` produced
  `res://Player.cs` and made the loaded class visible to Godot. The live
  smoke runner canonicalizes its temporary project path for this reason.
- This run kept HOME unchanged, used unique temporary projects and explicit
  `--instance` PIDs, and stopped only processes it started. Both headless
  editor-spawned play and GUI play worked on 4.7.2. An earlier manual
  headless scene save emitted the dummy-renderer `texture_2d_get` null-texture
  error; the GUI play session had no runtime errors. These checks do not
  expand the cross-platform headless support contract.
- `go test -race` passed here. `gopls` is absent and its installation had
  previously been declined; Go build/vet/tests supplied the Go checks.

## Live log read sharing (2026-09-08)

An isolated Windows Godot 4.7 editor returned FileAccess open error 12 while reading its running project's logger file, even though a shell could read that same path. Verify actual FileAccess success; file existence and a shell read are not evidence that addon diagnostics can read it. Hera now reports `available:false`, `clean:false` instead of zero counts when open/read fails. Inspect separately captured runtime stdout/stderr when this occurs. A missing final newline in project.godot is safely handled by the editor's ProjectSettings.save; live setter → in-memory setting → default run was verified without restart.
