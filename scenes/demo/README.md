# README walkthrough — recording script

This folder holds the asset behind the README demo GIF: an agent building a
playable scene, running it, and reading/controlling the live game **entirely
from the shell**, with compact JSON responses that stay tiny in an agent's
context.

- **Committed asset:** [`player.gd`](./player.gd) — a 15-line
  `CharacterBody2D` script that drives itself across the viewport and bounces
  at the edges, so a running session shows live motion `hera game node get`
  can read back.
- **Built live:** `Playground.tscn` is assembled by the commands below during
  recording (it is `.gitignore`d, not committed) so the walkthrough starts
  from nothing.

## Before recording

1. **One live editor**, this project open (mutations require exactly one).
2. **Restart the editor** first so stale scene tabs are gone and `player.gd`
   is freshly imported — the walkthrough's `attach-script` step then needs no
   filesystem scan.
2b. **Game Feel Mode off** (Hera panel → *Game Feel Mode(Beta)* toggle) so
   `node add` responses stay lean — with it on, step 2 also carries a
   game-feel `agent_hint` (~303 B instead of 59 B).
3. Put a terminal and the Godot window **side by side** (split screen). The
   left/terminal side shows the commands and compact JSON; the right/editor
   side shows the Scene dock filling in, then the running game window.
4. `hera` on `PATH` (or run the built binary). Confirm with `hera status`
   returning this project before you hit record.

## The walkthrough (run top to bottom, ~steady pace)

Each line is one command; the response is one compact line. The `≈tok` column
is the response's output token cost (bytes ÷ 4), for the low-token overlay.

| # | Command | Response (compact JSON) | bytes | ≈tok |
|---|---------|--------------------------|------|-----|
| 1 | `hera scene create res://scenes/demo/Playground.tscn --root Node2D --open` | `{"created":"…/Playground.tscn","opened":true,"root":"Node2D"}` | 70 | 18 |
| 2 | `hera node add CharacterBody2D --name Player` | `{"added":"Player","name":"Player","type":"CharacterBody2D"}` | 59 | 15 |
| 3 | `hera node set Player --prop position --value "Vector2(120, 200)"` | `{"path":"Player","prop":"position","value":"(120.0, 200.0)"}` | 60 | 15 |
| 4 | `hera node add ColorRect --parent Player --name Body` | `{"added":"Player/Body","name":"Body","type":"ColorRect"}` | 56 | 14 |
| 5 | `hera node set Player/Body --prop size --value "Vector2(48, 48)"` | `{"path":"Player/Body","prop":"size","value":"(48.0, 48.0)"}` | 59 | 15 |
| 6 | `hera node set Player/Body --prop color --value "Color(0.3, 0.8, 1, 1)"` | `{"path":"Player/Body","prop":"color","value":"(0.3, 0.8, 1.0, 1.0)"}` | 68 | 17 |
| 7 | `hera node attach-script Player res://scenes/demo/player.gd` | `{"base_type":"CharacterBody2D","path":"Player","script":"…/player.gd","script_diagnostics":{…}}` | 174 | 44 |
| 8 | `hera scene tree` | `{"count":3,"nodes":[Playground → Player → Body],…}` | 248 | 62 |
| 9 | `hera scene save` | `{"saved":"res://scenes/demo/Playground.tscn"}` | 45 | 11 |
| 10 | `hera run --current --wait` | `{"playing":true,"scene":"…/Playground.tscn"}` | 60 | 15 |
| 11 | `hera game tree` | live tree: `root → HeraGameInspector → Playground → Player → Body` | 431 | 108 |
| 12 | `hera game node get Player --props position,velocity` | `{…"position":"(325.3, 200.0)","velocity":"(-220.0, 0.0)"…}` | 148 | 37 |
| 13 | `hera game node get Player --prop position` *(repeat 2–3×)* | `{…"position":"(263.0, 200.0)"…}` → `(226.3…)` → `(178.6…)` | 118 | 30 |
| 14 | `hera game node set Player/Body --prop color --value "Color(1, 0.4, 0.2, 1)"` | `{"path":"…/Body","prop":"color","value":"(1.0, 0.4, 0.2, 1.0)"}` | 85 | 21 |
| 15 | `hera stop --wait` | `{"playing":false,"scene":""}` | 28 | 7 |

**Whole round-trip ≈ 427 output tokens** (~1,709 bytes) — build a scene, wire
a script, run the game, read live state, and recolor a running node, all in
~427 tokens of tool output. Compact JSON is the default (one line per command)
and the agent needs no tool-schema preload because it already speaks shell.

## Overlay beats (B story — low token)

Numbers measured on Godot 4.7; response sizes are exact bytes from a live run.

- **~427 tokens for the entire session.** Sum the `≈tok` column on screen as
  it runs, or show the total at the end.
- **Compact by default.** The same `scene tree` is **248 B compact vs 370 B**
  with `--json` (pretty) — ~33% smaller, and compact is the default.
- **`--ids` when only paths matter.** `hera --ids scene tree` returns just
  `.` / `Player` / `Player/Body` — **20 bytes** for the whole tree (~92%
  smaller than the JSON). Optional to show as a one-line aside.

## Visual beats (A story — the round-trip)

- **Steps 1–7:** the editor's Scene dock fills in node-by-node as each command
  runs — the clearest "the shell is driving the editor" moment.
- **Steps 11–13:** the game window is open and the box is sliding; the repeated
  `position` reads return **different, decreasing/increasing numbers** — proof
  it is a live running game, not a static inspection.
- **Step 14:** the box in the running window **changes color instantly** when
  the command runs — the agent reaching into a live game.

## After recording (cleanup)

`Playground.tscn` is git-ignored, so nothing is left to commit. To reset for a
re-take: `hera scene open res://scenes/Main.tscn`, then delete
`scenes/demo/Playground.tscn`.
