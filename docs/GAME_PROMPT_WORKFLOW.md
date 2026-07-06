# Prompt-Driven Game Workflow

Use this workflow when implementing a game prompt with Hera. The goal is to
turn a free-form prompt into a small requirement ledger, build against the live
Godot editor, then prove each requirement through runtime QA.

## 1. Extract Requirements First

Before editing, translate the prompt into stable requirement IDs. Keep them
short and observable.

Example:

```json
[
  "path-enemies-spawn",
  "build-spot-places-tower",
  "tower-shoots-nearby-enemies",
  "money-lives-wave-ui",
  "restart-resets-state",
  "viewport-no-clipping"
]
```

If `guidance ui` reports `game_feel_ui_mode:true`, include Game Feel as concrete
requirements instead of one vague line:

```json
[
  "hover-selection-feedback",
  "button-press-feedback",
  "state-change-motion",
  "runtime-screenshot-no-clipping"
]
```

## 2. Design For QA Hooks While Building

Timer-driven, AI-driven, hidden-state, or physics-heavy games should expose
deterministic helpers on the scene root.

Recommended names:

```gdscript
func qa_state() -> Dictionary
func qa_pause(paused: bool) -> void
func qa_step(seconds: float = 0.0) -> void
func qa_restart() -> void
```

Add domain-specific helpers only when they remove real flake:

```gdscript
func qa_place_tower(index: int) -> void
func qa_spawn_hazard_at(position: Vector2) -> void
func qa_reveal(cell: int) -> void
```

After running the game, use:

```sh
./hera game qa discover
```

Then call only discovered helpers with `game node call`.

## 3. Prefer Narrow Runtime Reads

Start broad once, then narrow every repeated read.

```sh
./hera game ui tree --type Button --fields name,path,text,disabled
./hera game ui tree --text Restart --fields name,path,text,rect,disabled
./hera game node get /root/Main --prop score
./hera game screenshot --analyze
```

Use full `game ui tree` only when the layout is still unknown.

## 4. Prove Requirements With `game qa`

Use object-format QA scenarios for implementation checks. Every requirement
from step 1 should appear in at least one step's `covers`.

```json
{
  "requirements": [
    "start-wave-button",
    "build-spot-places-tower",
    "money-lives-wave-ui",
    "runtime-screenshot-no-clipping"
  ],
  "steps": [
    {"tool": "run", "current": true, "wait": true},
    {"tool": "game.qa.discover", "covers": ["qa-hooks-present"]},
    {"tool": "game.click", "text": "Start Wave", "covers": ["start-wave-button"]},
    {"tool": "game.click", "x": 126, "y": 274, "covers": ["build-spot-places-tower"]},
    {
      "tool": "game.ui.tree",
      "covers": ["money-lives-wave-ui"]
    },
    {
      "tool": "screenshot.runtime",
      "covers": ["runtime-screenshot-no-clipping"]
    },
    {"tool": "diagnostics", "max_errors": 0}
  ]
}
```

If any requirement has no covering step, Hera fails before running the scenario.
If a covering step fails at runtime, that requirement stays missing in the final
summary.

## 5. Report Evidence, Not Intent

Record only observable evidence in `docs/reports/v0.6.0.md`:

- Which requirement IDs were covered.
- Which Hera command produced the evidence.
- Which issue remains open, if any.
- Whether the scene was rolled back after the test cycle.

Avoid final claims like "implemented all features" unless the requirement-covered
QA scenario passed and diagnostics were clean.
