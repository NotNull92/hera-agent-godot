# Godot support matrix

> Spot-checked **2026-07-13** with the current addon (`v0.7.0` baseline) on
> Windows, using official stable builds. Tracked in ROADMAP Phase 7.

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

- **Verified minimum: 4.2.** The addon parses, loads, serves the HTTP bridge,
  and reports `status` on 4.2–4.6. Nothing in the addon requires a 4.7-only
  API (the heaviest dependency is the `EditorInterface` singleton, available
  since 4.2).
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
`~/.hera-agent-godot/instances/` and run `hera --instance <pid> status`.
Expected noise on never-imported projects: font import errors from the panel's
bundled `.woff2` (harmless; real projects import assets before the plugin is
enabled).
