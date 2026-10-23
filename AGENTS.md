# Working with hera-agent-godot (for AI agents)

Use `hera-godot` to inspect and control the real Godot 4.x editor; confirm results.
The Hera Agent Godot plugin must be enabled. If no live editor is found, ask the
user to enable it. `hera-agent-godot` is the transitional CLI alias.

The [agent guide](docs/AGENT_GUIDE.md) is authoritative for operating rules:
[when to use Hera](docs/AGENT_GUIDE.md#when-to-use-it),
[setup](docs/AGENT_GUIDE.md#setup-once),
[engine and debug commands](docs/AGENT_GUIDE.md#commands),
[conventions and safety](docs/AGENT_GUIDE.md#conventions--safety),
[evidence and verification](docs/AGENT_GUIDE.md#verify-your-work-hera), and
[review boundaries](docs/AGENT_GUIDE.md#review-correction-boundaries).
Read relevant sections before operating. For addon development follow
[CO_DEVELOPMENT](docs/CO_DEVELOPMENT.md); use [COMMANDS](docs/COMMANDS.md) for
flags and responses. Prefer official Godot engine/docs sources for engine facts.

## Command card

Global `--instance PID`, `--ids`, `--json`, `--timeout MS`, `--engine PATH` go
before the command. `--flag=` equals `--flag ""`; normal validation applies.
Output is compact JSON. Read status `game_feel_ui_mode` before UI work;
use `guidance ui` for its checklist. `--set prop=value` is repeatable.

```
hera-godot status
hera-godot --ids scene tree [path] --depth N
hera-godot scene tree --type Class --props a,b
hera-godot node get <path> --prop p     # also --props a,b or --changed
hera-godot node add Node2D --set visible=false
hera-godot node instance res://child.tscn --set visible=false
hera-godot game assert /root/Main visible eq true --within-ms 1000
hera-godot signal list <path> --connected
hera-godot script validate res://file.gd
hera-godot run --current --wait
hera-godot stop --wait
hera-godot diagnostics
hera-godot game ui tree --type Button --fields name,path,text,disabled
hera-godot screenshot --runtime --analyze
```

The 2.1 CLI shape omits redundant counts/names, false `truncated`, and status
session/path fields. Missing log `source` means file. Pretty/batch share these rules;
explicit `--fields` stay. See [migration rules](docs/MIGRATING_TO_V2_1.md).

## GDScript quick gate

- **GDScript guide authority, low-token mode.**
  [docs/GDSCRIPT_AGENT_GUIDE.md](docs/GDSCRIPT_AGENT_GUIDE.md) is the
  authoritative source and must be followed, but do not reload the whole guide
  mechanically for routine edits. Use this quick gate first, then open the full
  guide only when the change touches syntax/API not covered here, diagnostics
  fail, the guide changed, or you are uncertain:
  - Do not invent syntax; when uncertain, check official Godot docs/engine
    sources or existing code.
  - Use explicit types for function parameters/returns, dynamic API results,
    `Variant`, and untyped `Array`/`Dictionary` reads.
  - Use `:=` only when Godot can infer a concrete non-Variant type.
  - Use GDScript ternaries (`a if condition else b`), never C-style `? :`.
  - Qualify engine constants/enums/flags with their owner, e.g.
    `Control.PRESET_FULL_RECT`.
  - Prefer `and`/`or`/`not`, typed `@onready` or exported node references,
    named signal handlers for normal UI, and `@tool` only when editor-time
    execution is required and guarded.
  - After any GDScript edit, run `hera-godot script validate res://file.gd`
    on the affected disk script before calling the work done. Fall back to
    `godot --headless --path . --check-only` when needed; use Hera
    diagnostics/run/output if no engine is available, as described in the guide.

## Verify your work

After an edit, **confirm it**. QA assertions accept `within_ms` (1–30,000);
read `waited_ms`, `timed_out` and the last `actual` when available.

- After `node set`: check the re-read `value` in the response; use `node get`
  only for other properties. After `node add --set` or `node instance --set`,
  check returned `values`; creation and properties share one undo action.
  Without `--set`, confirm structure with `hera-godot --ids scene tree`.
- After structural changes: `hera-godot scene tree` (or `--ids`).
- After `run`: `hera-godot diagnostics` first; `output --type error` for raw
  project file log lines, or
  `hera-godot diagnostics --source editor` when that capability is supported. File-log
  `clean: true` is not a claim about the editor Output panel.
- For UI/visual changes: `hera-godot screenshot` for the edited scene, or
  `hera-godot screenshot --runtime` after `run` for the live game viewport.
