# README walkthrough — recording script

This folder holds the asset behind the README demo GIF: an agent building a
playable scene, running it, **and running live QA on the running game** —
entirely from the shell, with compact JSON responses that stay tiny in an
agent's context.

- **Committed asset:** [`player.gd`](./player.gd) — a `CharacterBody2D` that
  auto-drives across the arena and bounces at the viewport edges (so a running
  session shows live motion `hera game node get` reads back) **and** steers on
  the built-in `ui_left` / `ui_right` actions (so `hera game input` can prove
  input QA: inject an action, then read the state back to confirm the game
  responded).
- **Built live:** `Playground.tscn` is assembled by the commands below during
  recording (it is `.gitignore`d, not committed) so the walkthrough starts
  from nothing.

## Before recording

1. **One live editor**, this project open (mutations require exactly one).
2. **Restart the editor**, then **wait until `hera status` returns** before you
   run anything else. A just-restarted editor is still loading; firing commands
   into it races the load and can produce duplicate / auto-named nodes. One
   clean `hera status` response means it is ready.
3. **Game Feel Mode off** (Hera panel → *Game Feel Mode(Beta)* toggle) so
   `node add` responses stay lean — with it on, `node add` also carries a
   game-feel `agent_hint` (~300 B instead of ~50 B).
4. Put a terminal and the Godot window **side by side** (split screen). The
   left/terminal side shows the commands and compact JSON; the right/editor
   side shows the Scene dock filling in, then the running game window.

## The walkthrough (run top to bottom)

Response sizes are exact bytes from a live Godot 4.7 run; `≈tok` is bytes ÷ 4.

### 1 — Build the arena and player (from an empty scene)

| Command | bytes |
|---------|------|
| `hera scene create res://scenes/demo/Playground.tscn --root Node2D --open` | 70 |
| `hera node add ColorRect --name Stage` *(background, added first = behind)* | 51 |
| `hera node set Stage --prop size --value "Vector2(1152, 648)"` | 56 |
| `hera node set Stage --prop color --value "Color(0.12, 0.12, 0.16, 1)"` | 65 |
| `hera node add ColorRect --name WallL` | 51 |
| `hera node set WallL --prop size --value "Vector2(24, 648)"` | 54 |
| `hera node set WallL --prop color --value "Color(0.9, 0.3, 0.5, 1)"` | 62 |
| `hera node add ColorRect --name WallR` | 51 |
| `hera node set WallR --prop position --value "Vector2(1128, 0)"` | 58 |
| `hera node set WallR --prop size --value "Vector2(24, 648)"` | 54 |
| `hera node set WallR --prop color --value "Color(0.9, 0.3, 0.5, 1)"` | 62 |
| `hera node add Label --name HUD` | 43 |
| `hera node set HUD --prop position --value "Vector2(40, 28)"` | 55 |
| `hera node set HUD --prop text --value "HERA LIVE QA"` | 51 |
| `hera node add CharacterBody2D --name Player` | 59 |
| `hera node set Player --prop position --value "Vector2(120, 300)"` | 60 |
| `hera node add ColorRect --parent Player --name Body` | 56 |
| `hera node set Player/Body --prop size --value "Vector2(64, 64)"` | 59 |
| `hera node set Player/Body --prop color --value "Color(0.3, 0.8, 1, 1)"` | 68 |
| `hera node attach-script Player res://scenes/demo/player.gd` | 174 |
| `hera scene tree` *(verify: 7 nodes)* | 443 |
| `hera scene save` | 45 |

Node order matters: `Stage` is added **first** so it draws behind everything;
`Player` last so its box draws on top. No `z_index` needed.

### 2 — Run and read the live game

| Command | Response | bytes |
|---------|----------|------|
| `hera run --current --wait` | `{"playing":true,…}` | 60 |
| `hera game tree` | live tree: `root → HeraGameInspector → Playground → Stage/WallL/WallR/HUD/Player → Body` | 695 |
| `hera game node get Player --prop position` *(repeat 2×)* | `(263.0, 300.0)` → `(321.7, 300.0)` — moving right | 121 |

### 3 — Live QA on the running game

| Command | What it proves | bytes |
|---------|----------------|------|
| `hera game qa diagnose` | Read-only health check: 0 errors, tree ok, UI ok, screenshot non-blank → **`ok:true, issues:[]`** | 417 |
| `hera game screenshot --analyze --path "user://hera_qa_frame.png"` | Captures the running frame + metrics: `nonblank:true`, `unique_colors:5`, `edge_content_detected:true` | 549 |
| `hera game node set Player --prop velocity --value "Vector2(220, 0)"` | Force a known state: moving right | 75 |
| `hera game node get Player --props position,velocity` | Confirm `velocity:(220, 0)` | 147 |
| `hera game input action ui_left --press` | Inject the `ui_left` action into the running game | 88 |
| `hera game node get Player --props position,velocity` | **`velocity:(-220, 0)`** — the game responded to the input | 148 |
| `hera game input action ui_left --release` | Release the action | 90 |
| `hera game input-log --limit 3` | Input diagnostic log shows the `ui_left` press/release with `source:"hera"` | 319 |

The **inject-then-read-back** pair is the centerpiece: it forces a known
velocity, injects an input, and reads the velocity back reversed — a real,
deterministic functional QA assertion, not a screenshot guess.

### 4 — Control the live game and stop

| Command | Response | bytes |
|---------|----------|------|
| `hera game node set Player/Body --prop color --value "Color(1, 0.4, 0.2, 1)"` | box recolors instantly in the running window | 85 |
| `hera stop --wait` | `{"playing":false,"scene":""}` | 28 |

**Whole session ≈ 1,170 output tokens** (~4,690 bytes across 36 commands) —
build an arena and player, run it, inspect live state, run functional QA
(health check + input-driven assertion + input log), recolor a running node,
and stop. Compact JSON is the default (one line per command) and the agent
needs no tool-schema preload because it already speaks shell.

## Overlay beats (B story — low token)

Response sizes are exact bytes from a live Godot 4.7 run.

- **~1,170 tokens for the entire build-run-QA session.** Sum the byte column
  on screen, or show the total at the end.
- **Compact by default.** The same `scene tree` is smaller compact than with
  `--json` (pretty); `--ids` drops it to just node paths.
- **Rich diagnostics stay small.** The heaviest responses are `game tree`
  (695 B), `game screenshot --analyze` (549 B) and `game qa diagnose` (417 B)
  — full health/QA readouts, still a few hundred bytes each.

## Visual beats (A story — the round-trip)

- **Section 1:** the editor's Scene dock fills in node-by-node as each command
  runs — the clearest "the shell is driving the editor" moment.
- **Section 2:** the game window is open, the box is sliding between the walls,
  the repeated `position` reads return **different numbers** — proof it is a
  live running game, not a static inspection.
- **Section 3 (QA):** `game qa diagnose` returns green; the `velocity` flips
  from `(220, 0)` to `(-220, 0)` the moment `ui_left` is injected — the agent
  drives input and verifies the game reacted. Bring up the editor's Debugger
  panel here if you want the debug view on screen alongside the QA commands.
- **Section 4:** the box in the running window **changes color instantly** when
  the command runs — the agent reaching into a live game.

## After recording (cleanup)

`Playground.tscn` is git-ignored, so nothing is left to commit. To reset for a
re-take: `hera stop --wait`, `hera scene open res://scenes/Main.tscn`, then
delete `scenes/demo/Playground.tscn`.
