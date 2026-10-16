# Tower defense demo rehearsal — 2026-10-16

Verdict: **PASS, all four criteria**. One timed implementation of the exact
fenced text in `demo/td-csharp/PROMPT.md`; no completed game was supplied to the
agent before the clock. Base commit: `aa7ac97` on `main` (parent `59f5def`).
The base is empty again; the populated scene and C# game are not committed.

## Acceptance evidence

| Criterion | Verdict | Evidence |
| --- | --- | --- |
| At least two successful C# command types | PASS | C# `eval` at +0.7 s, +165.7 s, +201.9 s; `script validate res://TdGame.cs` at +49.4 s and +119.3 s, both valid with no diagnostics. `script run` was not needed. |
| Hera-run game, enemies moving and tower firing | PASS | `run --current --wait` twice within this one implementation/fix cycle; 38-step QA scenario passes all 15 requirements. Paired motion captures show the same leading enemy at x=180.86055 and x=210.37738, y=336; capture A contains a live projectile, Shots=1, Kills=0, three enemies. |
| Editor survives rebuilds without restart | PASS | GUI editor PID 19900, session `80554817a67809b61a941fbdc0251375`, remains unchanged from preflight through both game builds, play sessions and final successful C# eval. No `csharp_unload_failed`; temporary assembly mismatch clears after automatic reload. |
| Agent time at most eight minutes | PASS | 220.417 seconds (3m 40.417s), 2026-10-16 06:10:58.553–06:14:38.970 UTC / 15:10:58.553–15:14:38.970 KST, including source writing, builds, checks, fixes, visual review and short run summary. Even including the subsequent stop cleanup it is under 226 seconds. |

## Environment and preparation (outside the timed run)

- Official Godot 4.7.2 .NET, commit `ed1daf0bf001b61586d9930840f2f1394092c079`,
  native GUI executable on the development PC; Compatibility/OpenGL on RTX 3060.
- Published `hera-godot@2.0.0` installed from npm in an isolated prefix;
  its downloaded `vendor/hera.exe` reports `v2.0.0`. This is not `go run` or
  the development PATH shim. npm's win32-x64 archive SHA256:
  `7ff49b5aa0b5df71c5e22d3fac75a998263bccc1203dad618c13f51121eb2709`.
- Copied root addon version 2.0.0; released `csharp-host install` installs
  helper v2.0.0 into the private profile. No second addon copy is committed.
- Target net8.0, Godot.NET.Sdk/4.7.2, nullable enabled; solution included.
  Empty-project preparation builds passed with zero compiler warnings/errors.
- Five ImageGen sprites generated before the clock, exported nearest-neighbor
  to 64×64; three sprites retain transparent backgrounds. Provenance and
  generation prompt subjects are in the demo README. Default Godot font.
- `HOME`, `USERPROFILE`, `APPDATA`, `LOCALAPPDATA`, XDG paths and
  `DOTNET_CLI_HOME` point under the disposable work directory. Existing NuGet
  package cache reused. Runtime registration confirms isolated `user_data_dir`:
  `.../profile/appdata/Godot/app_userdata/Hera TD Demo`.
- Game Feel Mode and Game Feel UI Mode both true, confirmed by status/guidance.
- The first **preparation-only** editor (PID 980) exposed a missing SDK 8:
  runtime 8.0.31 existed but only SDK 7/10 were installed. Its startup reported
  SDK/MSBuild errors. Installed official SDK 8.0.426 into the disposable prefix,
  prepended it to that environment's PATH/DOTNET_ROOT, rebuilt and launched
  the clean rehearsal editor PID 19900 **before starting the clock**.
  The install script needed explicit `-Architecture x64` on this host.
  No machine-wide SDK, PATH, user editor or user project was changed.

## Timeline and corrections

| Relative time | Observation/action |
| --- | --- |
| 0–2 s | Read frozen prompt/driver card; UI guidance and tweening reference; C# eval inspects edited root. |
| 2–55 s | Write one C# Node2D game; C# disk validation and normal build pass; Hera filesystem scan. |
| 63–67 s | Loaded assembly inspected; attach through Hera, save with evidence, run through Hera. Runtime 35944, viewport 960×640. |
| 76–106 s | Real mouse placement spends 40 gold; wave click, shots/kills/rewards observed; screenshots inspected; UI audit clean. |
| 114–127 s | Stop game; improve instruction wording, stable HUD name, clamp life count at zero. Second validation/build passes, no compiler warnings/errors. Immediate eval returns `compile_failed` because disk DLL is newer than loaded assembly. No retry of a mutation, no editor restart. |
| 147–154 s | Second play process 28548; 38 QA steps / 15 requirements pass. Wave, placement, occupied/path rejection, no overspend, combat, health/death, money, leaks, terminal HUD and restart checked. |
| 166–168 s | C# eval succeeds after normal assembly reload; same editor/session. Read Game Feel ethics/checklist. |
| 188–192 s | Restart through visible button; place one tower; start wave. Pause only for stable captures, resume 0.3 seconds, pause again; enemy position changes and first capture has a projectile. |
| 201–204 s | UI audit again has zero errors/warnings. Complete editor log has zero errors/warnings/dropped events. C# eval succeeds again. Runtime file log is unavailable, not claimed clean. |
| 220.417 s | Short rehearsal summary delivered; clock ends. |
| Afterwards | Stop through Hera, close only owned editor PID 19900, preserve evidence, restore Main/project settings, remove run outputs with scoped `git clean`, verify empty base, commit base on main. |

The first tweening lookup succeeded but the temporary Python console wrapper
could not print an em dash with CP949. Switched only that evidence runner's
stdout/stderr to UTF-8; the recorded CLI response and exit status were intact.
The unavailable runtime file log returned exit 0 plus `available:false`; this
is an evidence limitation, not a clean-log result. The available editor collector
reported `complete:true`, `dropped_count:0`, error_count=0, warning_count=0;
startup `editor.log` for PID 19900 also has no errors. Captures and state reads
are separate observations; the pause brackets make the motion comparisons stable.

## Game and visual QA

[QA scenario](2026-10-16-demo-td-qa.json): 38 ordered steps, all 15 top-level
requirements covered; screenshot steps require evidence. Loss is induced by
`QaLeak` with Lives=1, rather than waiting through a whole losing session.
This is targeted rule testing; combat and placement use real game inputs/time.

[Combat](assets/demo-td-combat.png) shows four placed towers, enemies, health
bars, reward text and HUD. [Motion A](assets/demo-td-motion-a.png) shows a tower
projectile in flight; [Motion B](assets/demo-td-motion-b.png) shows the enemies
further along the path. [Loss](assets/demo-td-loss.png) shows lives=0, disabled
GATE LOST wave button, loss instructions and available restart.
All four PNGs are actual 960×640 runtime captures with `size_matches_project:true`,
nonblank/detail checks passing and `possible_clipping:false`; visually reviewed.
The full runtime screen is a small fixed map; no offscreen camera or resizing
claims. Bounded tower/enemy scale pulses, projectile motion, hover tile/range
feedback and immediate HUD changes provide proportional feedback. No strong
flashes or shake, no audio, monetization, progression persistence or asset packs.

The generated game source was removed from the base after the run. Its final
SHA256 was `31554f175e549bb8d8369a0b1fbf427864c45e5c2a9549c0eff00a7dc21d55c7`;
the second C# validation reports exactly that disk hash. Local evidence retains
a copy outside the repository. Frozen PROMPT.md SHA256 remains
`49b948dda5943e528ad975a9b682d427e44eac57d7228b0ef2267ef3c37fc68b`.
No AGENTS.md fix was needed: the agent naturally selected eval and validation.

## Command ledger

Every controlled CLI invocation is recorded below, including preparation and
cleanup. Numbers refer to [raw command responses](2026-10-16-demo-td-commands.jsonl),
which preserve argument arrays, timestamps, exit codes, durations and stdout/stderr
UTF-8 byte counts. Sizes are process output sizes, **not HTTP wire sizes** or
capture file sizes. `+t` is seconds from the rehearsal clock; negative times are
preparation. Each command uses the published npm CLI's binary. The 38 children of
`game qa` are specified individually in the linked scenario; they are one CLI
invocation, and their result/evidence objects are retained in its raw response.

Additional preparation invocations made inside `prepare.ps1` / the npm shim,
before the controlled ledger: `--version` three times (7 stdout bytes each:
`v2.0.0` plus newline), `csharp-host install` twice (same private helper path,
same successful v2.0.0 JSON). These output sizes below are reconstructed from
the observed complete text; the installation/build terminal output itself was
not separately saved. Neither helper install is part of the timed run.

Preparation helper output: 158 bytes per install; stderr empty.

| # | +t (s) | Exact arguments after CLI | Exit | stdout B | stderr B |
| --- | --- | --- | --- | --- | --- |
| 1 | -163.889 | `--instance 980 status --capabilities` | 0 | 1061 | 0 |
| 2 | -148.967 | `--instance 980 eval --lang gdscript "EditorInterface.get_editor_settings().set_setting(\"hera_agent_godot/game_feel_mode\", true)"` | 0 | 80 | 0 |
| 3 | -148.755 | `--instance 980 eval --lang gdscript "EditorInterface.get_editor_settings().set_setting(\"hera_agent_godot/ui_juicy_mode\", true)"` | 0 | 80 | 0 |
| 4 | -148.545 | `--instance 980 scene open res://Main.tscn` | 0 | 65 | 0 |
| 5 | -148.340 | `--instance 980 eval SceneRoot.Name.ToString()` | 0 | 89 | 0 |
| 6 | -21.667 | `--instance 19900 status --capabilities` | 0 | 1119 | 0 |
| 7 | -21.336 | `--instance 19900 guidance game-feel` | 0 | 3655 | 0 |
| 8 | -21.089 | `--instance 19900 guidance ui` | 0 | 1265 | 0 |
| 9 | 0.216 | `--instance 19900 guidance ui` | 0 | 1265 | 0 |
| 10 | 0.415 | `--instance 19900 game_feel tweening_easing` | 0 | 2467 | 0 |
| 11 | 0.664 | `--instance 19900 eval SceneRoot.GetClass()` | 0 | 93 | 0 |
| 12 | 49.429 | `--instance 19900 script validate res://TdGame.cs` | 0 | 580 | 0 |
| 13 | 55.240 | `--instance 19900 project scan` | 0 | 33 | 0 |
| 14 | 62.833 | `--instance 19900 script inspect res://TdGame.cs` | 0 | 1376 | 0 |
| 15 | 62.999 | `--instance 19900 node attach-script . res://TdGame.cs` | 0 | 372 | 0 |
| 16 | 63.206 | `--instance 19900 scene save --evidence` | 0 | 609 | 0 |
| 17 | 63.591 | `--instance 19900 run --current --wait` | 0 | 43 | 0 |
| 18 | 67.152 | `--instance 19900 game instances` | 0 | 375 | 0 |
| 19 | 75.751 | `--instance 19900 game qa discover` | 0 | 138 | 0 |
| 20 | 76.009 | `--instance 19900 game screenshot --path res://initial.png --analyze` | 0 | 698 | 0 |
| 21 | 76.276 | `--instance 19900 game click --x 160 --y 272` | 0 | 61 | 0 |
| 22 | 76.563 | `--instance 19900 game node call /root/Main QaState` | 0 | 423 | 0 |
| 23 | 76.785 | `--instance 19900 game click --text "START WAVE"` | 0 | 126 | 0 |
| 24 | 85.018 | `--instance 19900 game node call /root/Main QaState` | 0 | 451 | 0 |
| 25 | 85.221 | `--instance 19900 game screenshot --path res://combat.png --analyze` | 0 | 698 | 0 |
| 26 | 85.486 | `--instance 19900 diagnostics --source editor --lines 10` | 0 | 315 | 0 |
| 27 | 105.550 | `--instance 19900 game ui audit` | 0 | 117 | 0 |
| 28 | 105.797 | `--instance 19900 game ui tree --fields name,path,text,disabled` | 0 | 664 | 0 |
| 29 | 114.170 | `--instance 19900 stop --wait` | 0 | 29 | 0 |
| 30 | 119.279 | `--instance 19900 script validate res://TdGame.cs` | 0 | 580 | 0 |
| 31 | 123.920 | `--instance 19900 eval SceneRoot.Name.ToString()` | 1 | 231 | 100 |
| 32 | 124.228 | `--instance 19900 run --current --wait` | 0 | 43 | 0 |
| 33 | 147.048 | `--instance 19900 game qa --file rehearsal-qa.json` | 0 | 4983 | 0 |
| 34 | 165.659 | `--instance 19900 eval SceneRoot.GetClass()` | 0 | 93 | 0 |
| 35 | 166.732 | `--instance 19900 status` | 0 | 315 | 0 |
| 36 | 166.917 | `--instance 19900 game_feel ethics_checklist` | 0 | 3029 | 0 |
| 37 | 167.109 | `--instance 19900 game_feel checklist_all` | 0 | 1704 | 0 |
| 38 | 188.284 | `--instance 19900 game click --text RESTART` | 0 | 109 | 0 |
| 39 | 188.365 | `--instance 19900 game click --x 352 --y 272` | 0 | 61 | 0 |
| 40 | 188.462 | `--instance 19900 game click --text "START WAVE"` | 0 | 115 | 0 |
| 41 | 188.549 | `--instance 19900 game node call /root/Main QaState` | 0 | 440 | 0 |
| 42 | 188.682 | `--instance 19900 game node call /root/Main QaState` | 0 | 440 | 0 |
| 43 | 188.817 | `--instance 19900 game node call /root/Main QaState` | 0 | 440 | 0 |
| 44 | 188.948 | `--instance 19900 game node call /root/Main QaState` | 0 | 440 | 0 |
| 45 | 189.078 | `--instance 19900 game node call /root/Main QaState` | 0 | 439 | 0 |
| 46 | 189.209 | `--instance 19900 game node call /root/Main QaState` | 0 | 440 | 0 |
| 47 | 189.344 | `--instance 19900 game node call /root/Main QaState` | 0 | 439 | 0 |
| 48 | 189.473 | `--instance 19900 game node call /root/Main QaState` | 0 | 441 | 0 |
| 49 | 189.609 | `--instance 19900 game node call /root/Main QaState` | 0 | 440 | 0 |
| 50 | 189.734 | `--instance 19900 game node call /root/Main QaState` | 0 | 440 | 0 |
| 51 | 189.914 | `--instance 19900 game node call /root/Main QaState` | 0 | 440 | 0 |
| 52 | 190.044 | `--instance 19900 game node call /root/Main QaState` | 0 | 439 | 0 |
| 53 | 190.176 | `--instance 19900 game node call /root/Main QaState` | 0 | 439 | 0 |
| 54 | 190.308 | `--instance 19900 game node call /root/Main QaState` | 0 | 440 | 0 |
| 55 | 190.389 | `--instance 19900 game clock --pause` | 0 | 75 | 0 |
| 56 | 190.471 | `--instance 19900 game node call /root/Main QaState` | 0 | 440 | 0 |
| 57 | 190.547 | `--instance 19900 game screenshot --path res://motion-a.png --analyze --evidence` | 0 | 1201 | 0 |
| 58 | 190.693 | `--instance 19900 game clock --resume` | 0 | 76 | 0 |
| 59 | 191.067 | `--instance 19900 game clock --pause` | 0 | 75 | 0 |
| 60 | 191.148 | `--instance 19900 game node call /root/Main QaState` | 0 | 440 | 0 |
| 61 | 191.229 | `--instance 19900 game screenshot --path res://motion-b.png --analyze --evidence` | 0 | 1200 | 0 |
| 62 | 201.269 | `--instance 19900 game ui audit` | 0 | 117 | 0 |
| 63 | 201.571 | `--instance 19900 diagnostics --source editor --lines 20` | 0 | 315 | 0 |
| 64 | 201.755 | `--instance 19900 output --type error` | 0 | 599 | 0 |
| 65 | 201.945 | `--instance 19900 eval SceneRoot.GetChildCount()` | 0 | 78 | 0 |
| 66 | 203.026 | `--instance 19900 status --capabilities` | 0 | 1119 | 0 |
| 67 | 220.562 | `--instance 19900 stop --wait` | 0 | 29 | 0 |

## Reset and verification

Base commit `aa7ac97` contains only the empty project, solution, five sprites,
prepare script, README and isolation/ignore files. The pre-existing frozen prompt
remains unchanged. Main has no script; no `.cs`, copied addon, `.godot`, import
sidecars or screenshots remain under the demo. The editor-generated autoload UID
was removed from the base; the copied addon registers its own inspector on launch.

Checked sprite dimensions/alpha, project SDK/TFM/nullable, PowerShell AST syntax,
empty Main, frozen prompt hash, scoped cleanup preview and `git diff --check`.
Actual preparation (including helper install), two game builds, two Hera C#
validations, runtime input/state/QA and visual inspection supplied the behavioral
checks. No Go/GDScript product code changed, so their unrelated test suites were
not rerun. No changes to version strings, tags, releases, stores, PROMPT.md or
pre-existing untracked PLAN/REVIEW/HTML/research files; no push performed.

## Remaining recording steps

Install/use SDK 8 on the recording environment; prepare the base with the explicit
published CLI path. Open the base, enable both Game Feel settings, open Main and
check the version panel. Record the 30-second manual intro, then submit only the
frozen prompt text and start the timer. Expect a brief asynchronous assembly reload
after builds; wait for readiness rather than restarting. Repeat the scoped reset
after recording if desired. This rehearsal is evidence of this single Windows run,
not a promise that every autonomous future run takes the same time or has the same
layout. Runtime file-log evidence and recording/export were not verified.
