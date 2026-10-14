# Godot support matrix

The [2026-09-14 local baseline](reports/2026-09-14-engine-capability-baseline.md)
verified the installed Windows 4.7.2 .NET executable, all 52 addon parse checks,
17 standalone regressions, and isolated live status/smoke. It does not expand
the support tiers below or re-verify 4.2–4.6 on this machine.

The M1 editor collector was exercised locally on that Windows 4.7.2 engine:
real log/error callbacks, four logging threads, cursors/overflow, teardown,
and CLI evidence reads. Runtime API checks plus an observed registration
callback gate `editor_log_cursor`; no engine-version allowlist is used.
The fixed Logger source is compiled only after API checks, keeping unsupported
types out of common static dependencies. Local 4.2 parse/load execution for
this change remains unverified; the existing CI 4.2 all-addon gate is retained.

> Live `status` spot-check **2026-07-13** on official Windows stables (then
> `v0.7.0`). GDScript `--check-only` on **4.2-stable** and **4.7-stable** still
> runs in CI on every commit, including the `v1.2.0` tree.

Hera is developed and fully QA'd against **Godot 4.7**. The spot-check below
verifies how far back the addon actually works: every addon script passes the
GDScript static gate, the plugin loads in a headless editor, and the CLI
answers `status` end-to-end over the live HTTP bridge.

| Godot (stable) | `--check-only` all addon scripts | Plugin loads + CLI `status` answers | Full command surface QA'd |
|---|---|---|---|
| 4.2 | ✅ | ✅ | — |
| 4.3 | ✅ | ✅ | — |
| 4.4 | ✅ | ✅ | — |
| 4.5 | ✅ | ✅ | — |
| 4.6 | ✅ | ✅ | — |
| 4.7 | ✅ (CI, every commit) | ✅ (development baseline) | ✅ |

## Configured headless runtime tier

The [headless CI recipe](HEADLESS_CI.md) configures a pinned, **Godot 4.7-only**
nonvisual lifecycle: the enabled addon starts in a headless editor, publishes a
fresh heartbeat, answers CLI requests, passes `smoke --skip-game`, and runs the
deterministic runtime-logic scenario. It does not claim live headless runtime
support for 4.2–4.6, all Hera commands, screenshots, visual UI, renderer output,
or window/input behavior.

Remote GitHub Actions verification passed on 2026-07-13 at commit
[`5c0ba65`](https://github.com/NotNull92/hera-agent-godot/commit/5c0ba6562961a6a11ab581d0f4eab440d34ce008).
The [successful run](https://github.com/NotNull92/hera-agent-godot/actions/runs/29256396824)
covered the Godot 4.7 nonvisual editor-to-game lifecycle and its
requirement-covered runtime-logic scenario. See
[HEADLESS_CI.md](HEADLESS_CI.md) for the exact lifecycle and exclusions.

**What this means**

- **Verified minimum: 4.2.** The addon parses on 4.2 in CI. It loaded and
  answered `status` on 4.2–4.6 in the 2026-07-13 spot-check. v1.1.0/v1.2.0
  commands (`node reparent`, `game clock`, joypad events, Logger-backed editor
  logs when the API exists) use APIs present in 4.2 or degrade via capability
  states.
  Nothing in the addon requires a 4.7-only class (the heaviest dependency is
  the `EditorInterface` singleton, available since 4.2).
- **Recommended: 4.7+.** Only 4.7 gets the full live QA treatment (runtime
  game control, input injection, screenshot analysis, `game qa` scenarios)
  every release. On 4.2–4.6, commands beyond the spot-checked path are
  expected to work but are not routinely exercised.
- CI runs the GDScript `--check-only` gate on **both ends of the matrix**
  (oldest verified: 4.2-stable; newest: 4.7-stable), so a change that breaks
  the 4.2 floor fails the build.

**Method** (repeatable): for each version, download the official
`Godot_v<V>-stable_win64.exe.zip`, copy `addons/hera_agent_godot/` into a
fresh minimal project, run `--headless --path <proj> --check-only --script`
over every addon `.gd`, then enable the plugin and boot
`--headless --editor`; wait for the heartbeat under
`~/.hera-agent-godot/instances/` and run `hera-godot --instance <pid> status`.
Expected noise on never-imported projects: font import errors from the panel's
bundled `.woff2` (harmless; real projects import assets before the plugin is
enabled).

## v1.2.0 export guard and Godot 4.2 (2026-10-14)

The 4.2 standalone `--check-only` process does not set the editor hint.
Its [ClassDB::can_instantiate](https://github.com/godotengine/godot/blob/4.2-stable/core/object/class_db.cpp#L401-L419)
returns false for editor API classes outside editor mode, and the
[GDScript analyzer](https://github.com/godotengine/godot/blob/4.2-stable/modules/gdscript/gdscript_analyzer.cpp#L4943-L4959)
rejects a statically resolved constructor. Both
[4.2](https://github.com/godotengine/godot/blob/4.2-stable/editor/register_editor_types.cpp)
and [4.7](https://github.com/godotengine/godot/blob/4.7-stable/editor/register_editor_types.cpp)
register `EditorExportPlugin` as a constructible class. The unmodified 4.2
editor loaded Hera successfully; the standalone failure was not an editor
load failure. Hera now loads the export-guard script at editor startup and
constructs it there, where editor classes are available.

The guard is retained on 4.2: export callbacks still suspend Hera's owned
runtime inspector autoload and restore it for editor play. There is no 4.2
fallback that includes the inspector in exported project settings, and no
security or isolation downgrade. Do not run the inspector in exported games.
