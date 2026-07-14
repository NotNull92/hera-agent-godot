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
  "primary-action-changes-state",
  "secondary-action-or-cancel-works",
  "keyboard-input-affects-gameplay",
  "hud-reflects-runtime-state",
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
func qa_trigger_primary_action(target: int = 0) -> Dictionary
func qa_spawn_entity_at(position: Vector2) -> Dictionary
func qa_force_interaction(a: int, b: int) -> Dictionary
```

Report-derived patterns from the v0.7.0 prompt cycles:

- Treat state-changing runtime QA as an ordered transaction. Do not parallelize
  semantic clicks, `game input`, `game node call qa_*`, or other state-mutating
  runtime commands against the same live game process.
- If live runtime registration is empty while editor diagnostics are clean, run
  a direct affected-scene load before changing autoloads or adding runtime
  workarounds. Warning-as-error parse failures are the first branch to rule out.
- Keep `guidance game-feel` separate from `guidance ui`. Use gameplay feel
  guidance for combat, reward, camera, control, audio, particles, accessibility,
  and runtime feel QA; use UI guidance for `Control` layout and input feedback.
- For `Node2D` games with `CanvasLayer` HUDs, set full-screen decorative/root
  `Control` nodes to `Control.MOUSE_FILTER_IGNORE` so map clicks reach
  `_unhandled_input`. Keep only real buttons, selectors, and sidebars
  interactive.
- For framed board, grid, lane, arena, or sidebar UIs, verify parent padding,
  child rect bounds, readable text contrast, and disabled-state contrast from
  the live UI state, not only screenshot clipping.
- For tokens, markers, cards, units, hazards, rewards, and grid marks, use
  semantic bounded child visuals instead of letter-only control text. Keep the
  interactive frame stable and animate the child visual, overlay, or draw layer.
- For grid, board, card, inventory, lane, or tile layouts, derive internal
  insets from `frame_size - (cell_count * cell_size + gaps)` instead of hand
  guessing offsets. Verify first/last cell rects and overlay origin.
- For inspection handoff states, leave a stable representative scene: default
  settings restored, stale transient feedback cleared, and at least two
  state-linked Game Feel channels visible when Game Feel is part of the prompt.
- For primary play-surface plus HUD/sidebar layouts, compare sibling panel
  geometry in the runtime UI tree. Mismatched outer heights, unbounded helper
  copy, and uneven density are visual QA failures even without clipping.
- For runtime drawing, derive backgrounds from the live viewport and keep
  map/playfield/HUD rectangles inside explicit padded layout bounds. Do not set
  `get_viewport().size` from gameplay scripts unless fixed resolution is an
  explicit prompt requirement.
- For realtime or physics-driven games, expose restart/start helpers,
  deterministic step helpers, and targeted event helpers so scoring, removal,
  damage, life changes, or launch state can be verified without racing natural
  motion.
- For delayed or locked states, expose trigger and step-forward helpers, then
  assert lock flags, visible state, and disabled control counts while the delay
  is active.
- For dense or hidden-state games, prefer filtered reads such as
  `game ui tree --type Button --fields name,path,text,disabled` over full UI
  dumps during repeated QA. If one QA step creates a flag, mark, selection, or
  setup state, later helpers should avoid that precondition or reset it.
- For AI-assisted or automated-turn games with undo, document whether undo rolls
  back one atomic action or a full player-plus-system turn, and expose QA that
  proves that boundary.
- For autonomous movement games, expose a restart-paused helper, pause control,
  and one-step movement helper. Restart and pause immediately before
  deterministic inspection.
- For every generated game, identify the primary input scheme from the prompt
  and drive it through the live runtime with `game input`. Helper-only QA is not
  enough for keyboard-first, mouse-first, touch-first, or controller-first games.
- For priority-based AI, expose explicit setup helpers for the highest-priority
  branches in addition to a simple smoke check.
- For stateful toggles or mode buttons, read the current UI tree before
  semantic text clicks, reset to a known baseline, or target a stable node path.
  Do not assume the old label still exists after helper calls.
- For collision-heavy games, expose a forced-overlap helper so collision
  feedback, screen shake, particles, end-state, and Game Feel evidence can be
  proven without waiting for random movement.
- For settings controls available during pause, win, loss, draw, or game-over,
  append to or preserve the terminal-state instruction instead of replacing it
  with only the setting change.
- Programmatic state or configuration changes must update visible selectors,
  labels, and counters in the same transaction. QA should compare internal state
  against the current UI tree when helpers change difficulty, mode, tool, or
  ruleset values.
- For resource, progression, survival, or failure loops, isolate state changes
  such as spend, reward, damage, recovery, spawn, completion, and loss with
  focused helpers instead of relying only on a full natural run.
- If generated code needs Hera runtime helper entrypoints, discover the current
  add-on path or autoload from this checkout before hardcoding helper script
  paths, then run a direct affected-scene load.
- For visible traversal paths, lanes, rails, patrols, projectiles, or routes,
  derive drawing and movement from one authoritative geometry. Smooth corners
  before rendering thick paths and verify with a screenshot plus one live
  movement step.
- For Game Feel QA, expose the active target, channel list, duration,
  intensity/scope, and visible values. Do not treat a boolean or feature-list
  string as proof that the rendered target changed.
- For animated UI feedback, regenerate style/theme resources only on state
  changes. Frame loops should update bounded transforms, offsets, opacity, or
  draw values.

After running the game, use:

```sh
./hera game qa discover
```

Then call only discovered helpers with `game node call`.

## 3. Run Generic Runtime Diagnosis

After starting a game, establish the runtime baseline before writing any
project-specific assertions:

```sh
./hera game qa diagnose
```

This is deliberately genre-agnostic. It checks the editor diagnostics, whether
exactly one game process is live, whether runtime node or UI reads were
truncated, and whether the runtime capture is blank, low-detail, or likely
clipped. It does not require a HUD, button, scene-root name, `qa_*` helper, or
any particular game rule.

Use `--max-warnings N` only when warnings are part of the project's acceptance
criteria. Use `--path user://qa-baseline.png` when the capture must be retained.
Follow this baseline with a requirement-covered scenario for the prompt's own
rules; generic diagnosis cannot prove genre-specific behavior.

## 4. Prefer Narrow Runtime Reads

Start broad once, then narrow every repeated read.

```sh
./hera game ui tree --type Button --fields name,path,text,disabled
./hera game ui tree --text Restart --fields name,path,text,rect,disabled
./hera game node get /root/Main --prop score
./hera game screenshot --analyze
```

Use full `game ui tree` only when the layout is still unknown.

## 5. Prove Requirements With `game qa`

Use object-format QA scenarios for implementation checks. Every requirement
from step 1 should appear in at least one step's `covers`.

```json
{
  "requirements": [
    "primary-pointer-action",
    "secondary-pointer-diagnostics",
    "keyboard-input",
    "hud-reflects-runtime-state",
    "runtime-screenshot-no-clipping"
  ],
  "steps": [
    {"tool": "run", "current": true, "wait": true},
    {"tool": "game.qa.discover", "covers": ["qa-hooks-present"]},
    {
      "tool": "game.input",
      "params": {
        "kind": "mouse",
        "mode": "press",
        "x": 160,
        "y": 220,
        "button": "left",
        "modifiers": ["shift"]
      },
      "covers": ["primary-pointer-action", "secondary-pointer-diagnostics"]
    },
    {"tool": "wait", "duration_ms": 650, "covers": ["secondary-pointer-diagnostics"]},
    {
      "tool": "game.input",
      "params": {
        "kind": "mouse",
        "mode": "release",
        "x": 160,
        "y": 220,
        "button": "left",
        "modifiers": ["shift"]
      },
      "covers": ["primary-pointer-action", "secondary-pointer-diagnostics"]
    },
    {
      "tool": "game.input",
      "params": {"kind": "mouse", "mode": "click", "x": 240, "y": 220, "button": "right", "modifiers": ["ctrl"]},
      "covers": ["secondary-pointer-diagnostics"]
    },
    {
      "tool": "game.input",
      "params": {"kind": "key", "mode": "press", "key": "KEY_ENTER", "physical": true},
      "covers": ["keyboard-input"]
    },
    {
      "tool": "game.input",
      "params": {"kind": "key", "mode": "release", "key": "KEY_ENTER", "physical": true},
      "covers": ["keyboard-input"]
    },
    {"tool": "game.input_log", "lines": 10, "covers": ["secondary-pointer-diagnostics", "keyboard-input"]},
    {
      "tool": "game.ui.tree",
      "path": "/root/Main/HUD",
      "text": "Restart",
      "params": {"type": "Button", "fields": ["name", "path", "text", "rect", "disabled"], "depth": 4},
      "covers": ["hud-reflects-runtime-state"]
    },
    {
      "tool": "screenshot.runtime",
      "covers": ["runtime-screenshot-no-clipping"]
    },
    {"tool": "diagnostics", "max_errors": 0}
  ]
}
```

For long-click QA, do not put `duration_ms` inside a `game.input` step. Use a
`game.input` mouse `press`, a `wait` step, then a matching mouse `release`; the
runtime `game.input_log` entry records the measured `duration_ms` and
`click_kind`.

If any requirement has no covering step, Hera fails before running the scenario.
If a covering step fails at runtime, that requirement stays missing in the final
summary.

## 6. Lint Findings Before Reporting

Prompt-game reports should capture reusable implementation lessons, not
one-game design notes. Before adding a finding to the user-confirmed release
QA report, run this lint gate:

- Pass findings about generic Hera or Godot workflow risks: live editor/runtime
  ambiguity, input routing conflicts, viewport clipping patterns, deterministic
  QA helper gaps, parser/check-only failures, warning-as-error issues, unclear
  API contracts, repeated manual work, accessibility/safety conflicts, or slow
  feedback loops.
- Pass findings whose fix would help at least two different game genres or Hera
  itself.
- Fail findings tied to one game's rules, tuning, content, theme, scoring
  balance, spawn values, level design, or player strategy.
- Fail prompt-completion evidence that only says the current game worked. Keep
  that in the cycle summary, not as an implementation finding.
- If a finding names a specific game mechanic, rewrite it as the generic pattern
  it exposed. If the issue cannot survive that rewrite, skip it.

Useful lint question: "Would this still matter if the next prompt were a
different genre?" If no, do not add it as a report finding.

Examples:

- Pass: `CanvasLayer` root `Control` nodes can consume map clicks before
  `Node2D._unhandled_input()`.
- Pass: Timer-driven games need pause/step/force-event QA helpers to avoid
  racing runtime state.
- Fail: Breakout's opening serve angle is too sharp.
- Fail: A tower-defense cost, enemy count, or color palette should be tuned.

## 7. Report Evidence, Not Intent

Record only observable evidence in the user-confirmed release QA report:

- Which requirement IDs were covered.
- Which Hera command produced the evidence.
- Which issue remains open, if any.
- Whether the scene was rolled back after the test cycle.

Avoid final claims like "implemented all features" unless the requirement-covered
QA scenario passed and diagnostics were clean.
