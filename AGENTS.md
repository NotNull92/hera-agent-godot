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
hera status                                  # project / version / active scene
hera scene tree                              # node tree of the edited scene
hera scene list                              # open scenes + current
hera scene open res://Path.tscn              # open a scene
hera scene save                              # save the edited scene
hera scene create res://Path.tscn [--root Node2D] [--force] [--open]
hera scene save-as res://Path.tscn [--force]
hera script create res://scripts/foo.gd [--extends Node2D] [--class-name Foo] [--force]
hera project mkdir res://scripts
hera node find [query] [--type Class]        # find nodes
hera node get <path>                         # dump a node's properties
hera node add <type> [--parent p] [--name n] # add a node (undoable)
hera node set <path> --prop p --value v      # set a property (undoable)
hera node remove <path>                      # remove a node (undoable)
hera node attach-script <path> <res://script.gd> # attach a script (undoable)
hera node detach-script <path>               # clear a node script (undoable)
hera signal list <node>                      # signals a node exposes + connections
hera signal connect <from> <sig> <to> <method>     # wire a signal (undoable)
hera signal disconnect <from> <sig> <to> <method>  # unwire (undoable)
hera resource get <res://...>                # dump a resource's properties
hera game tree                               # running game node tree
hera game instances                          # running game process heartbeats
hera game screenshot [--path p] [--analyze]  # capture/analyze running game viewport
hera game node get <path> [--prop p|--props a,b] # running game node properties
hera game node set <path> --prop p --value v # set a running game property (not undoable)
hera game node call <path> <method> [--arg v] # call a running game method (not undoable)
hera game assert <path> <prop> <op> [value]  # assert runtime property for QA
hera game qa --file scenario.json            # run generic QA scenario
hera run [--scene r] [--current] [--wait]    # play; hera stop [--wait]
hera eval "<expression>"                     # evaluate one GDScript expression
hera output [--type log|error|warning|all] [--lines N]
hera diagnostics [--lines N]                 # summarize project log errors/warnings
hera screenshot [--path p] [--width N] [--height N] [--runtime] [--analyze] # render edited scene or runtime viewport
hera batch [--file f] [--continue]           # run a JSON array of {tool, params}
hera instances                               # list live Hera-enabled editors
hera smoke [--run-game|--skip-game]          # quick live editor smoke check
```

Global flags go **before** the command: `--json` (pretty-print), `--ids` (print
only node paths, for `scene tree` / `node find`), `--instance <pid>` (explicitly
target a pid shown by `status`). Default output is compact JSON.

## Conventions & safety

- **Output is compact by default** to stay low-token. Use `--ids` to get just
  node paths when scanning, `--json` only when you need the full structure.
- **Mutations are undoable where Godot exposes editor undo.**
  `node add/set/remove`, `node attach-script/detach-script`, and
  `signal connect/disconnect` register with the editor's undo history, so the
  user can Ctrl+Z those changes.
- **Run one live editor per project.** Hera is designed for a single active
  Godot editor. Mutation-capable commands (`node add/set/remove`,
  `node attach-script/detach-script`, `signal connect/disconnect`,
  `scene open/save/create/save-as`, `script create`, `project mkdir`, `eval`,
  `game node set/call`, `smoke --run-game`, and `batch`) enforce that by
  refusing to run when several editors are live unless `--instance <pid>` is
  passed explicitly.
- **`eval` is powerful.** It runs one GDScript expression (not statements) with
  the edited scene root as base, so `get_node("X").something()` works — and can
  have side effects. It is **not** registered with undo. Prefer `node set` for
  property changes.
- **`game node set/call` is runtime-only.** It changes the running game process,
  is not registered with undo, and is lost when the play session stops.
- **Runtime game requests are process-isolated.** If stale Godot game processes
  are still alive, `game instances` shows them and mutation/read requests refuse
  ambiguous targets instead of accepting an old response.
- **Prefer low-token QA reads.** Use `game node get --prop/--props`,
  `game assert`, `screenshot --runtime --analyze`, and `game qa --file` before
  dumping full node properties during automated QA.
- **File and scene creation are persistent.** `script create`, `project mkdir`,
  `scene create`, and `scene save-as` write project files; use `--force` only
  when overwriting is intended.
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
