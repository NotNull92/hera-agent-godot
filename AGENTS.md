# Working with hera-agent-godot (for AI agents)

`hera-agent-godot` is a low-token CLI that lets you inspect and control a **live
Godot 4.x editor**. Use it to act on the *real* editor and check the result —
don't guess scene structure or whether a change worked from memory.

## When to use it

- You need the actual state of the open scene (node tree, a node's properties).
- You want to change the scene (add/set/remove nodes) and confirm it stuck.
- You want to run a scene, then read the log for errors.
- You want an off-screen preview render of the edited scene (`screenshot`).

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
hera node find [query] [--type Class]        # find nodes
hera node get <path>                         # dump a node's properties
hera node add <type> [--parent p] [--name n] # add a node (undoable)
hera node set <path> --prop p --value v      # set a property (undoable)
hera node remove <path>                      # remove a node (undoable)
hera signal list <node>                      # signals a node exposes + connections
hera signal connect <from> <sig> <to> <method>     # wire a signal (undoable)
hera signal disconnect <from> <sig> <to> <method>  # unwire (undoable)
hera run [--scene r] [--current] [--wait]    # play; hera stop [--wait]
hera eval "<expression>"                     # evaluate one GDScript expression
hera output [--type log|error|warning|all] [--lines N]
hera screenshot [--path p] [--width N] [--height N]  # render edited scene to PNG (GUI editor)
hera batch [--file f] [--continue]           # run a JSON array of {tool, params}
```

Global flags go **before** the command: `--json` (pretty-print), `--ids` (print
only node paths, for `scene tree` / `node find`), `--instance <pid>` (explicitly
target a pid shown by `status`). Default output is compact JSON.

## Conventions & safety

- **Output is compact by default** to stay low-token. Use `--ids` to get just
  node paths when scanning, `--json` only when you need the full structure.
- **Mutations are undoable.** `node add/set/remove` register with the editor's
  undo history, so the user can Ctrl+Z your changes.
- **Run one live editor per project.** Hera is designed for a single active
  Godot editor. Mutation commands (`node add/set/remove`, `signal
  connect/disconnect`, `scene open/save`, `eval`, and `batch`) enforce that by
  refusing to run when several editors are live unless `--instance <pid>` is
  passed explicitly.
- **`eval` is powerful.** It runs one GDScript expression (not statements) with
  the edited scene root as base, so `get_node("X").something()` works — and can
  have side effects. It is **not** registered with undo. Prefer `node set` for
  property changes.
- **`node set` value** is coerced to the property's type. Pass GDScript-literal
  syntax for complex types, e.g. `--value "Vector2(10, 20)"`.

## Verify your work (Hera)

After an edit, **confirm it** instead of assuming:

- After `node add`/`set`: `hera node get <path>` and check the value.
- After structural changes: `hera scene tree` (or `--ids`).
- After `run`: `hera output --type error` to catch runtime errors.
- For UI/visual changes: `hera screenshot` (renders the edited scene off-screen; needs a GUI editor).

Batch a change and its check together when it helps, e.g. pipe a JSON array of
`[{set...}, {get...}]` into `hera batch`.

See [docs/COMMANDS.md](docs/COMMANDS.md) and [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).
