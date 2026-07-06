# Working with hera-agent-godot (for AI agents)

`hera-agent-godot` is a low-token CLI that lets you inspect and control a **live
Godot 4.x editor**. Use it to act on the *real* editor and check the result —
don't guess scene structure or whether a change worked from memory.

## When to use it

- You need the actual state of the open scene (node tree, a node's properties).
- You want to change the scene (add/set/remove nodes) and confirm it stuck.
- You want to run a scene, then read the log for errors.
- You want an off-screen preview render of the edited scene (`screenshot`) or a
  live game viewport capture (`screenshot --runtime` / `game screenshot`).

If the user is not running the Godot editor with the **Hera Agent** plugin
enabled, commands fail with "no live Godot editor found" — ask them to enable it.

## Setup (once)

1. Open the project in a **Godot 4.x** editor.
2. Enable **Project → Project Settings → Plugins → Hera Agent Godot**. The Output
   panel should show `[hera] ... listening on 127.0.0.1:<port>`.
3. Get the CLI: install a release binary (see the README's Install section) or
   build from source with `go build -o hera .` (from the repo root).

The CLI finds the editor automatically via `~/.hera-agent-godot/instances/`.

## Commands

```
hera status                                  # project / version / active scene / UI mode
hera scene tree                              # node tree of the edited scene
hera scene list                              # open scenes + current
hera scene open res://Path.tscn              # open a scene
hera scene reload [res://Path.tscn]          # reload current/open scene from disk
hera scene save                              # save the edited scene
hera scene create res://Path.tscn [--root Node2D] [--force] [--open]
hera scene save-as res://Path.tscn [--force]
hera editor state                            # current scene, selection, play/script state
hera editor selected                         # selected editor nodes
hera editor select <node> [--add]            # select a node in the editor
hera editor clear-selection                  # clear editor node selection
hera script current                          # inspect focused script
hera script inspect res://scripts/foo.gd     # script metadata
hera script open res://scripts/foo.gd [--line N] [--column N] # open in script editor
hera script create res://scripts/foo.gd [--extends Node2D] [--class-name Foo] [--force] [--tool] [--ready] [--process] [--physics-process] [--input] [--unhandled-input] [--signal name] [--export name:type=value]
hera project mkdir res://scripts
hera project scan                            # refresh Godot's resource filesystem
hera project reimport res://icon.svg         # reimport one or more project files
hera project set-main-scene res://Path.tscn # set ProjectSettings main scene
hera node find [query] [--type Class]        # find nodes
hera node get <path> [--prop p|--props a,b]  # dump all or selected node properties
hera node add <type> [--parent p] [--name n] # add a node (undoable)
hera node instance <res://scene.tscn> [--parent p] [--name n] # instance a PackedScene (undoable)
hera node set <path> --prop p --value v      # set a property (undoable)
hera node remove <path>                      # remove a node (undoable)
hera node attach-script <path> <res://script.gd> # attach a script (undoable; returns dependency diagnostics)
hera node detach-script <path>               # clear a node script (undoable)
hera signal list <node>                      # signals a node exposes + connections
hera signal connect <from> <sig> <to> <method>     # wire a signal (undoable)
hera signal disconnect <from> <sig> <to> <method>  # unwire (undoable)
hera resource get <res://...>                # dump a resource's properties
hera resource list [res://dir] [--type Class] [--pattern text] [--limit N] # list resources
hera resource set <res://...> --prop p=v     # set and save resource properties
hera resource create <Class> <res://out.tres> [--force] [--prop name=value]
hera game tree                               # running game node tree
hera game ui tree [--path p] [--depth N] [--fields a,b] [--type Class] [--text t] # running Control nodes, optionally scoped
hera game instances                          # running game process heartbeats
hera game screenshot [--path p] [--analyze]  # capture/analyze running game viewport
hera game click --x N --y N                  # click the running game viewport
hera game click --node /root/Main/Button     # click the center of a live Control
hera game click --text Restart               # click the center of a visible Control by text
hera game node get <path> [--prop p|--props a,b] # running game node properties
hera game node set <path> --prop p --value v # set a running game property (not undoable)
hera game node call <path> <method> [--arg v] # call a running game method (not undoable)
hera game assert <path> <prop> <op> [value]  # assert runtime property for QA
hera game qa discover [path]                 # list callable runtime qa_* helpers
hera game qa --file scenario.json            # run generic QA scenario, optionally with requirements/covers
hera run [--scene r] [--current] [--wait]    # play; hera stop [--wait]
hera eval "<expression>"                     # evaluate one GDScript expression
hera guidance ui                             # UI guidance; reads Game Feel UI Mode
hera output [--type log|error|warning|all] [--lines N]
hera diagnostics [--lines N]                 # summarize project log errors/warnings
hera screenshot [--path p] [--width N] [--height N] [--runtime] [--analyze] # render edited scene or runtime viewport
hera batch [--file f] [--continue]           # run a JSON array of {tool, params}
hera instances                               # list live Hera-enabled editors
hera smoke [--run-game|--skip-game]          # quick live editor smoke check; run-game includes runtime screenshot analysis
```

Global flags go **before** the command: `--json` (pretty-print), `--ids` (print
only node paths, for `scene tree` / `node find`), `--instance <pid>` (explicitly
target a pid shown by `status`). Default output is compact JSON.

## Conventions & safety

- **Output is compact by default** to stay low-token. Use `--ids` to get just
  node paths when scanning, `--json` only when you need the full structure.
- **UI work reads the live guidance mode first.** Before agent-driven UI work,
  run `hera guidance ui`. If it reports `game_feel_ui_mode: true`, implement
  UI around Game Feel: immediate input feedback, expressive state changes,
  satisfying bounded motion, and runtime visual QA for those effects.
- **Mutations are undoable where Godot exposes editor undo.**
  `node add/instance/set/remove`, `node attach-script/detach-script`, and
  `signal connect/disconnect` register with the editor's undo history, so the
  user can Ctrl+Z those changes.
- **Run one live editor per project.** Hera is designed for a single active
  Godot editor. Mutation-capable commands (`node add/instance/set/remove`,
  `node attach-script/detach-script`, `signal connect/disconnect`,
  `scene open/reload/save/create/save-as`, `editor select/clear-selection`, `script open/create`, `resource set/create`, `project mkdir/scan/reimport`,
  `project set-main-scene`, `eval`, `game node set/call`, `smoke --run-game`,
  and `batch`) enforce that by
  refusing to run when several editors are live unless `--instance <pid>` is
  passed explicitly.
- **`eval` is powerful.** It runs one GDScript expression (not statements) with
  the edited scene root as base, so `get_node("X").something()` works — and can
  have side effects. It is **not** registered with undo. Prefer `node set` for
  property changes.
- **GDScript guide authority, low-token mode.**
  [docs/GDSCRIPT_AGENT_GUIDE.md](docs/GDSCRIPT_AGENT_GUIDE.md) is the
  authoritative source and must be followed, but do not reload the whole guide
  mechanically for routine edits. Use this quick gate first, then open the full
  guide only when the change touches syntax/API not covered here, diagnostics
  fail, the guide changed, or you are uncertain:
  - Do not invent syntax; check official docs or existing code when uncertain.
  - Use explicit types for function parameters/returns, dynamic API results,
    `Variant`, and untyped `Array`/`Dictionary` reads.
  - Use `:=` only when Godot can infer a concrete non-Variant type.
  - Use GDScript ternaries (`a if condition else b`), never C-style `? :`.
  - Qualify engine constants/enums/flags with their owner, e.g.
    `Control.PRESET_FULL_RECT`.
  - Prefer `and`/`or`/`not`, typed `@onready` or exported node references,
    named signal handlers for normal UI, and `@tool` only when editor-time
    execution is required and guarded.
  - After any GDScript edit, run `godot --headless --path . --check-only` on the
    affected scene or script before calling the work done. If `godot` is not
    available, use Hera diagnostics/run/output as described in the guide.
- **`game node set/call` and `game click` are runtime-only.** They change the running game process,
  is not registered with undo, and is lost when the play session stops.
- **Runtime game requests are process-isolated.** If stale Godot game processes
  are still alive, `game instances` shows them and mutation/read requests refuse
  ambiguous targets instead of accepting an old response.
- **Prefer low-token QA reads.** Use `game ui tree`, `game node get --prop/--props`,
  `game assert`, `game qa discover`, `screenshot --runtime --analyze`, and
  `game qa --file` before dumping full node properties during automated QA.
  Runtime screenshot analysis
  reports per-edge content ratios and `possible_clipping` so layouts that only
  fail at the viewport boundary are easier to catch.
- **Tie QA to the user's requirements.** For prompt implementation QA, prefer a
  `game qa --file` object with top-level `requirements` and per-step `covers`
  entries so missing requested behavior fails the scenario instead of being
  buried in prose notes.
- **File, scene, resource, and project setting changes are persistent.** `script create`,
  `resource set/create`, `project mkdir/reimport`, `project set-main-scene`, `scene create`, and
  `scene save-as` write project files; use `--force` only when overwriting is
  intended.
- **Use a single writer for scene files.** Before external `.tscn` edits, stop the
  running game with `hera stop --wait`; after external edits, run
  `hera scene reload [res://Path.tscn]` before saving through the editor.
- **`node set` value** is coerced to the property's type. Pass GDScript-literal
  syntax for complex types, e.g. `--value "Vector2(10, 20)"`.

## Verify your work (Hera)

After an edit, **confirm it** instead of assuming:

- After `node add`/`set`: `hera node get <path>` and check the value.
- After structural changes: `hera scene tree` (or `--ids`).
- After `run`: `hera output --type error` to catch runtime errors.
- For UI/visual changes: `hera screenshot` for the edited scene, or
  `hera screenshot --runtime` after `run` for the live game viewport.

Batch a change and its check together when it helps, e.g. pipe a JSON array of
`[{set...}, {get...}]` into `hera batch`.

See [docs/COMMANDS.md](docs/COMMANDS.md) and [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).
For prompt-driven game implementation cycles, follow
[docs/GAME_PROMPT_WORKFLOW.md](docs/GAME_PROMPT_WORKFLOW.md).
