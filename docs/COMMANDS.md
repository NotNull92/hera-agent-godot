# Commands

> Status: implemented command surface. Output is compact by default to stay
> low-token.

Each command maps 1:1 to an addon tool and sends a single JSON request to the
selected editor instance.

## Linked evidence (opt-in)

`script validate res://file.gd --evidence` adds `source: disk`, selected engine
path and editor-reported version/commit, pre/post SHA-256, start/finish UTC times,
and `changed_during_validation`. Missing engine/hash evidence or a changed file
makes `valid: false` and exits 1, even if the child exited 0. The ordinary 64-KiB
output cap, timeout and engine exit code remain. Each hash reads at most 64 MiB.
`editor_buffer`, `loaded_script`, and `running_code` each report unavailable;
external IDE buffers are not observed. Equal hashes are two observations, not an
atomic snapshot: changes between reads, changes restored before the second read,
and changed dependencies can escape detection. Version comes from the selected
editor context; the child banner remains in output, and replacing the engine
executable concurrently is not prevented. No automatic reload occurs.

`screenshot --evidence`, `screenshot --runtime --evidence`, and
`game screenshot --evidence` add source, editor/runtime sessions, PID, scene,
wall-clock capture time, observed process/physics/draw frame counters, actual PNG
size and viewport/window/project geometry. Editor source is an offscreen
`editor_preview`, not the running game. `--runtime-session ID` rejects a different
runtime incarnation. `--operation-id session:unix_ms:nonce` adds only a validated
format correlation label with `association: caller_supplied`; it does not look up
a receipt or prove causality. These options imply `--evidence`. Runtime images
are never upscaled. Frame counters/time are sampled at readback, not an atomic
state-and-image snapshot. A rendered-frame boundary does not imply animation has
settled. Headless rendering and a missing frame after 1000 ms return explicit
unavailable evidence, including while paused or at time scale zero. Ordinary
captures without `--evidence` keep their previous error strings; they do not
use the `evidence_unavailable` prefix. Runtime screenshots wait for a rendered
frame only when evidence is requested.

`scene save --evidence` and `resource set ... --evidence` report `effect`,
`save_call`, `persistence` and separate before/after disk observations. Optional
`--expected-sha256 HASH` checks the current file before dispatching the save or
resource mutation; external/manual writers are not locked out after that check.
CLI capability negotiation and distinct internal evidence actions prevent an old
addon from ignoring the save precondition. Failed commands can print their
evidence on stdout with exit 1 and an error on stderr; defaults stay compact.

These protections also apply to non-nested `batch` children: `evidence: true`
requests capability negotiation before the batch is sent, and `expected_sha256`
on scene save/resource set implies evidence. Protected saves use distinct internal
actions even after preflight. A failed or missing evidence child makes the CLI
exit 1. A success-shaped child without available evidence becomes `ok: false`,
`error: evidence_unavailable`, with its `data` retained. This response check occurs
after execution; it cannot undo earlier children or change the server's `stopped`
observation. Batches without evidence keep their existing output and exit behavior.

Resource property changes report `effect: applied` even if saving fails. A fresh
resource load without the root cache verifies selected non-object/non-container
properties; observed mismatch means `persistence: failed`, even when the engine
save call returned success. Unsupported verification is unavailable, never a
pass. `saved` applies only to those selected properties, not all dependencies.
Scene save reports prior memory effect and full persistence as `unknown`: its
save-call result and file hash do not prove that all scene memory reached disk.
For scene saves, `evidence.available: true` means the file observation is readable;
`memory_disk_equivalence.available: false` still means full persistence is unverified.
No rollback or storage-device durability is claimed. A scene with no path reports
failed persistence and `save_call: not_called`.

Existing QA steps `screenshot.runtime`, `game.node.get` and `game.assert` accept
`"evidence": true`; returned evidence is retained and unavailable evidence fails
the step and its `covers` coverage. Capture and state assertions carry separate
timestamps. Cover interaction requirements with the relevant `game.assert` after
input: a passing visual step cannot override a failing state assertion. Missing
InputMap actions still fail; Hera does not create actions or alter game logic.
`hera-godot status --capabilities` advertises this bounded addon surface as `capabilities.linked_evidence`.

## Editor mutation gate

Each editor session permits one mutation or sensitive read at a time. Conflicting
requests immediately fail `editor_busy` without waiting or running. The gate spans
async work and scan completion, independent of the HTTP connection or client
timeout. `status`, `operation status|cancel` and `output`/`diagnostics --source editor`
remain available; cancellation still affects only work that has not started.
Scene, resource, script, runtime and file-log reads are conservatively gated.
Batch children explicitly share their parent's ownership; nested batches remain
invalid. A failed child never grants ownership to an unrelated request.

Scan completion combines engine scan state, completion events and the end of
pending filesystem processing. Reimport checks indexed paths and resulting import
validity/resource availability. Consumers still validate the particular resource
or script they need; idle scanning alone is not successful compilation or C#
assembly readiness. `hera-godot status --capabilities` reports the gate as
`capabilities.editor_mutation_gate`;
`import_busy_api` separately reports `EditorFileSystem.is_importing` availability.
On engines without that API (including 4.2), Hera-owned import lifetimes and
observed scans are covered, but unrelated editor imports cannot always be observed.

Ownership ends after execution settles, or on session teardown; rejected/queued
cancellation does not release another running owner. No effects are rolled back.
A lost runtime response remains unknown under the existing runtime receipt
contract, not proof that an application method finished. Independent project
sessions share no gate. The gate coordinates Hera requests, not manual editor
actions, external file writers or work an application schedules after returning.

## Operation receipts (experimental)

`operation submit <id> --request '<JSON>'` explicitly wraps a mutation in an
in-process receipt. `operation status <id>` reads it; `operation cancel <id>`
cancels only accepted, queued work. Default stdout omits `retention` and echoed
`evidence.response`; `--verbose` prints the full receipt. Exit 0 still means the
receipt was obtained, not that the mutation succeeded. These commands use the
selected editor and require `--instance` when several editors are live. Legacy
commands are unchanged.

The caller constructs and retains `id` as `EDITOR_SESSION:UNIX_MS:NONCE`, using
`status.editor_session_id`, an absolute execution deadline no more than 60 seconds
ahead, and a unique ASCII alphanumeric/underscore/hyphen nonce. Example PowerShell:

```powershell
$session = (hera-godot status | ConvertFrom-Json).editor_session_id
$deadline = [DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds() + 30000
$operationId = "${session}:${deadline}:change1"
# $request contains one supported {tool,params} object described below.
hera-godot operation submit $operationId --request $request
hera-godot operation status $operationId
hera-godot operation cancel $operationId
```

Supported request objects (maximum 16 KiB UTF-8):

- `{"tool":"node","params":{"action":"set","path":".","prop":"visible","value":"false","expected":{...},"verify":true}}`.
  Use the complete `expected` from `node get --prop visible --snapshot`.
  The addon normalizes `set` to the guarded action before admission.
- `{"tool":"game","params":{"action":"set","pid":123,"runtime_session_id":"...","path":"/root/Main","prop":"speed","value":"5"}}`.
- `{"tool":"game","params":{"action":"call","pid":123,"runtime_session_id":"...","path":"/root/Main","method":"increment","args":[1]}}`.
  Read PID and random runtime session identity from `game instances`.

Other tools/actions fail `capability_unavailable` before execution. The CLI sends
a SHA-256 of the raw request for boundary validation; the addon computes its own
digest from the normalized input and session. Same ID and normalized request
returns current/preserved state without dispatch. Different input with a retained
ID fails `operation_id_conflict`. Deadlines are checked at admission and again
before dispatch, including the runtime file consumer. A retained completed receipt
is readable after the execution deadline.

Receipts separate `lifecycle` (`accepted|running|completed|rejected|cancelled|outcome_unknown`),
`effect` (`not_applied|applied|unknown`), `verification`, `persistence`, stable
`error_code`, `target`, `input_digest`, `evidence`, and `cancellable`.
Successful method dispatch reports `applied`; it does not prove the method's
business outcome. Arbitrary calls have `persistence:unknown`; sets do not request
a disk save. Setter side effects and application saves are not tracked. Guarded
verification failures preserve `effect:applied` and report failed/unavailable
verification. Unclassified execution errors and lost runtime responses become
`outcome_unknown`. Failures that never published a runtime request or invoked
the setter/call are `rejected` with `effect:not_applied` and a stable code
(`target_unavailable`, `invalid_operation`, `editor_busy`, …), so a new ID is
allowed. Inspect receipt fields even when CLI exit is zero: that exit
means the receipt was obtained, not that the mutation succeeded.

Cancellation wins only before dispatch (`cancelled:true`, `effect:not_applied`).
Running/completed operations return `cancelled:false`, preserving their effect;
no rollback or interruption is promised. An HTTP timeout does not cancel work.
The CLI never automatically retries a failed mutation. Query its ID, or explicitly
resubmit the same ID and input. A retained unknown outcome is never re-executed.

Each editor and runtime keeps at most 128 records and 4 MiB of serialized records.
Full response evidence is capped at 16 KiB; clipping retains the effect and ID and
reports `evidence.complete:false` with a dropped-result count. Capacity rejects
new operations instead of evicting valid IDs. Records expire 60 seconds after the
immutable ID deadline; an expired ID cannot become fresh execution. Missing history
reports unknown effect, with retention-expired/session-mismatch codes where known.
Retention metadata exposes expired/dropped counts and `durable:false`. Clocks are
anchored to monotonic elapsed time during each process lifetime.

Plugin restart loses editor history and changes its session; old-session submissions
fail `session_mismatch`. Runtime restart changes `runtime_session_id` and rejects
old targets. The runtime also deduplicates repeated file deliveries during its
lifetime. Atomic request publication and partial-file reads remain unchanged.
There is no durable or distributed exactly-once guarantee or automatic recovery
after a process dies before its effect is recorded. Do not give an uncertain
mutation a new ID merely to force it to run.

## Command reference

| Command | Tool | Status | Description |
|---------|------|--------|-------------|
| `status [--capabilities]` | `status` | ☑ | Show the connected editor: project path, Godot version, active scene, `editor_session_id`, `csharp_supported` (editor build, not SDK), and Game Feel modes. `--capabilities` adds experimental `godot_commit` and the full API `capabilities` map (`supported`/`unsupported`/`unverified`). |
| `run [--scene <res://...>] [--current] [--wait]` | `run` | ☑ | Play the main scene (default), the current scene (`--current`), or a specific scene (`--scene`). `--wait` polls until the matching runtime scene is inspectable. |
| `stop [--wait]` | `run` | ☑ | Stop the running scene. `--wait` polls until stopped. |
| `output [--type log\|error\|warning\|all] [--lines N] [--source file\|editor] [--since cursor]` | `output` | ☑ | Default: tail the project's configured file log. Opt-in editor source: bounded session evidence with cursors and severity/location metadata; see below. Unreadable evidence reports `available:false`. |
| `diagnostics [--lines N] [--source file\|editor] [--since cursor]` | `diagnostics` | ☑ | Default: summarize the project's configured file log and set `source: file`. That `clean` flag is only the file log, not the editor Output panel. Opt-in editor source: counts over the retained observed interval. Unavailable evidence is never reported clean. |
| `scene tree` | `scene` | ☑ | Print the edited scene's node tree (compact: path/type/name). |
| `scene list` | `scene` | ☑ | List open scenes and the current one. |
| `scene open <res://...>` | `scene` | ☑ | Request opening a scene in the editor. |
| `scene reload [res://...]` | `scene` | ☑ | Reload the current or named open scene from disk, useful after external `.tscn` edits before saving through the editor. |
| `scene save` | `scene` | ☑ | Save the edited scene. |
| `scene create <res://...> [--root <type>] [--force] [--open]` | `scene` | ☑ | Create a new `.tscn` with an instantiable node root; refuses overwrite unless `--force` is passed. |
| `scene save-as <res://...> [--force]` | `scene` | ☑ | Save the edited scene to a new `.tscn`; refuses overwrite unless `--force` is passed. |
| `editor state` | `editor` | ☑ | Show editor context: current scene, open scenes, main scene, play state, selected nodes, and current script. |
| `editor selected` | `editor` | ☑ | Return the current editor node selection with scene-relative paths when possible. |
| `editor select <node> [--add]` | `editor` | ☑ | Select a node in the edited scene; clears the previous selection unless `--add` is passed. Editor-state mutation only. |
| `editor clear-selection` | `editor` | ☑ | Clear the editor node selection. Editor-state mutation only. |
| `script current` | `script` | ☑ | Inspect the Godot-focused script resource: `.gd` source metadata or `.cs` loaded-assembly metadata. External IDE focus is not observable. |
| `script inspect <res://script.gd\|res://Script.cs>` | `script` | ☑ | Return low-token metadata: `.gd` is source-based; `.cs` uses loaded-assembly metadata, which may be unavailable or stale (see language notes below). |
| `script open <res://script.gd\|res://Script.cs> [--line N] [--column N]` | `script` | ☑ | Open a script through Godot and its external-editor configuration, optionally at a 1-based line/column. C# requires Godot .NET. Editor-state mutation only. |
| `script create <res://script.gd\|res://Script.cs> [--lang gdscript\|csharp] [--extends <Class>] [--class-name <Name>] [--force] [--tool] [--ready] [--process] [--physics-process] [--input] [--unhandled-input] [--signal <name> ...] [--export <name:type[=value]> ...]` | `script` | ☑ | Create a script and refresh the editor filesystem. The extension selects the language; `--lang` must match. Flags generate language-appropriate tool annotations, lifecycle stubs, signals, and exports. C# requires Godot .NET and a filename-matching class; build/reload before attachment. |
| `project info` | `project` | ☑ | Show project name, root path, Godot version, current scene, and file counts by type. |
| `project list-files [--type all\|scene\|script\|resource\|asset\|shader] [--pattern <p>] [--limit N]` | `project` | ☑ | List project files from `res://`, with compact type tags and optional filtering. |
| `project scan` | `project` | ☑ | Request a Godot editor resource filesystem scan so newly written files are visible to editor tools. Editor filesystem mutation only. |
| `project reimport <res://file> ...` | `project` | ☑ | Ask Godot to reimport one or more safe `res://` project files through `EditorFileSystem.reimport_files`. Persistent import metadata/cache change. |
| `project mkdir <res://dir>` | `project` | ☑ | Create a project directory under `res://` and refresh the editor filesystem. |
| `project set-main-scene <res://scene.tscn>` | `project` | ☑ | Set and persist `application/run/main_scene` through the targeted editor; its in-memory setting and subsequent default `run` agree immediately. |
| `node find [query] [--type <Class>]` | `node` | ☑ | Find nodes by name substring and/or class. |
| `node get <path> [--prop <name>\|--props <a,b>] [--snapshot]` | `node` | ☑ | Dump a node's editor-visible properties, or selected properties for low-token editor inspection. `--snapshot` requires one `--prop` and adds a typed conditional-mutation snapshot. |
| `node add <type> [--parent <path>] [--name <n>]` | `node` | ☑ | Add a node under a parent (undoable). When Game Feel Mode is enabled, feel-related node types return a compact `agent_hint` pointing at relevant `game_feel` topics. |
| `node instance <res://scene.tscn> [--parent <path>] [--name <n>]` | `node` | ☑ | Instance a PackedScene under a parent after validating the scene path (undoable). |
| `node set <path> --prop <name> --value <v> [--expected <JSON> [--verify]]` | `node` | ☑ | Set a node property (undoable; value coerced to the property's type). Optional typed preconditions reject stale targets before mutation. |
| `node set-resource <path> --prop <name> --resource <res://...>` | `node` | ☑ | Set an object/resource property from a Resource file, with path and type compatibility checks (undoable). |
| `node remove <path>` | `node` | ☑ | Remove a node (undoable). |
| `node reparent <path> --parent <path> [--no-keep-global-transform]` | `node` | ☑ | Move a node to a new parent (undoable). Defaults to keeping the global transform the way `Node.reparent` does. |
| `node attach-script <path> <res://script.gd\|res://Script.cs>` | `node` | ☑ | Attach a script after validating path and base type (undoable). GDScript checks obvious `preload("res://...")` dependencies. C# requires Godot .NET and a loaded class; build/reload first. C# diagnostics carry a build warning, not C# compiler diagnostics. |
| `node detach-script <path>` | `node` | ☑ | Clear a node's script (undoable). |
| `signal list <node>` | `signal` | ☑ | List the signals a node exposes (name + arg names) and scene-local connections; editor-internal targets are counted as `external_connections`. |
| `signal connect <from> <sig> <to> <method>` | `signal` | ☑ | Connect a node's signal to a method on another node (undoable; persistent, saved with the scene). |
| `signal disconnect <from> <sig> <to> <method>` | `signal` | ☑ | Remove that connection (undoable). |
| `resource get <res://...>` | `resource` | ☑ | Load a resource (`.tres`/`.res`/`.tscn`/any `res://`) and dump its class, name, and editor-visible properties. Read-only; no scene needs to be open. |
| `resource uid <res://...>` | `resource` | ☑ | Return Godot's resource UID plus the `.uid` sidecar content when present. |
| `resource list [res://dir] [--type <Class>] [--pattern <text>] [--limit N]` | `resource` | ☑ | Recursively list project resources from a safe `res://` path, optionally filtering by resource class, path substring, and result limit. |
| `resource set <res://...> --prop <name=value> ...` | `resource` | ☑ | Load a resource, coerce Godot literal strings to the target property types, set editor-visible properties, and save it back to disk. Persistent filesystem change. |
| `resource create <Class> <res://out.tres> [--force] [--prop <name=value> ...]` | `resource` | ☑ | Create an instantiable `Resource` class, optionally set editor-visible properties using Godot literal strings, and save it as `.tres`/`.res`. |
| `resource resave <res://...>` | `resource` | ☑ | Load and save a resource to refresh serialized data and UID metadata. Persistent filesystem change. |
| `resource update-uids` | `resource` | ☑ | Resave project resources/scripts that Godot can load, useful after migrations that need UID sidecars refreshed. Persistent filesystem change. |
| `resource export-mesh-library <res://scene.tscn> <res://out.tres> [--item <name> ...]` | `resource` | ☑ | Build a `MeshLibrary` from top-level scene children containing `MeshInstance3D` nodes, optionally filtered by item name. |
| `theme get <res://theme.tres> [--type <ThemeType>]` | `theme` | ☑ | Dump a `Theme`'s per-type items (colors, constants, font sizes). Read-only. A `Theme`'s data lives behind `set_color`/`set_constant`/`set_font_size` rather than properties, so `resource get` cannot reach it. |
| `theme set <res://theme.tres> --type <ThemeType> [--color <name=Color(r,g,b,a)>] [--constant <name=int>] [--font-size <name=int>]` | `theme` | ☑ | Set items on one theme type and save the resource. Persistent filesystem change and **not undoable** — there is no `EditorUndoRedoManager` step for a resource write. Create the file first with `resource create Theme <res://theme.tres>`. |
| `classdb info <Class>` | `classdb` | ☑ | Show ClassDB metadata: parent, instantiability, Node/Resource ancestry. |
| `classdb methods <Class>` | `classdb` | ☑ | List ClassDB methods with compact argument and return type summaries. |
| `classdb properties <Class>` | `classdb` | ☑ | List ClassDB properties with type, class, hint, and hint string. |
| `classdb signals <Class> [--own]` | `classdb` | ☑ | List ClassDB signals with compact argument summaries. Includes inherited signals by default; `--own` limits output to the class itself. |
| `classdb constants <Class> [--own]` | `classdb` | ☑ | List ClassDB integer constants with values and enum membership when available. Includes inherited constants by default; `--own` limits output to the class itself. |
| `classdb enums <Class> [--own]` | `classdb` | ☑ | List ClassDB enums and their integer constants. Includes inherited enums by default; `--own` limits output to the class itself. |
| `classdb inherits <Class> <BaseClass>` | `classdb` | ☑ | Check inheritance using Godot ClassDB. |
| `game [--pid <game-pid>] tree` | `game` | ☑ | Print a live game node tree. Without `--pid`, use the editor-play process; an explicit fresh PID from `game instances` also targets externally launched games. |
| `game ui tree [--path <node>] [--depth N] [--fields <a,b>] [--type <Class>] [--text <label>]` | `game` | ☑ | Print live `Control` nodes. Scope by subtree, depth, class, exact text, and returned fields (`name,path,type,visible,rect,text,disabled,pressed`) for low-token UI QA before semantic clicks. |
| `game ui audit [--path <node>] [--severity error\|warning\|all] [--rule <id>] [--strict] [--limit N]` | `game` | ☑ | Audit visible runtime `Control` nodes for generic UI defects: empty interactive rectangles, interactive controls outside the viewport or fully clipped, full-viewport mouse blockers, minimum sizes that cannot fit their parent, and overlapping interactive siblings. Errors fail the command; warnings fail only with `--strict`. `--limit` defaults to 50 and accepts 1–500 findings. |
| `game instances` | `game` | ☑ | List Hera runtime game processes seen by the editor, including pid, scene, heartbeat age, `user_data_dir`, and viewport sizes. Fresh entries are `instances[]`; expired heartbeats are `stale[]`. Processes that share `user://` set `shared_user_data`. |
| `game screenshot [--path <p>] [--analyze]` | `game` | ☑ | Capture the running game viewport to PNG at its live size (no upscale). Returns `path`, PNG `width`/`height`, window/visible/project sizes, and `size_matches_project` (PNG vs project viewport, not the visible rect). `--analyze` adds generic image/layout metrics (`nonblank`, dimensions, sampled color count, brightness, edge content by side, clipping and low-detail hints). `possible_clipping` is not a resolution match. Input coordinates use this actual viewport. |
| `game click --x N --y N` / `game click --node <path>` / `game click --text <label>` | `game` | ☑ | Send a left mouse click to the running game viewport. `--node` and `--text` target the center of a live `Control`, avoiding brittle pixel coordinates. Runtime-only and useful for surface-level QA. |
| `game input mouse --x N --y N --button left\|right\|middle --press\|--release\|--click [--modifiers shift,ctrl]` | `game` | ☑ | Inject a runtime mouse input event and record it in the input diagnostic log. Separate `--press` and `--release` calls, with `wait`, let QA prove long-click behavior. |
| `game input key --key KEY_W --press\|--release [--physical] [--modifiers shift,ctrl]` / `game input action <name> --press\|--release` / `game input text <text>` | `game` | ☑ | Inject keyboard, InputMap action, or text input events for direct gameplay QA. Key events default to `Input.parse_input_event`; `--route viewport` sends them through the viewport when focused Controls need key events. |
| `game input joypad --button A\|JOY_BUTTON_A --press\|--release [--device N]` / `game input axis --axis LEFT_X --value -1..1 [--device N]` | `game` | ☑ | Inject `InputEventJoypadButton` / `InputEventJoypadMotion` through `Input.parse_input_event`. Button and axis names match Godot's `JoyButton` / `JoyAxis` constants (prefix optional). Default device is Hera's synthetic device id; pass `--device 0` when the game reads a specific pad. |
| `game input-log [--limit N] [--clear]` | `game` | ☑ | Read the runtime input diagnostic log: click coordinates, button, press/release, short/long classification, key names, modifiers, active keys, and active mouse buttons. |
| `game clock [--pause\|--resume] [--time-scale N] [--step] [--physics]` | `game` | ☑ | Read or change the live `SceneTree.paused` flag and `Engine.time_scale`. No flags returns `{paused, time_scale, process_frames, physics_frames}`. `--step` unpauses for one `process_frame` (or `physics_frame` with `--physics`) and leaves the tree paused. `time_scale` must be greater than 0; use `--pause` to stop the clock. The runtime inspector uses `PROCESS_MODE_ALWAYS` so later `game` commands still run while paused. |
| `game node get <path> [--prop <name>\|--props <a,b>]` | `game` | ☑ | Dump a live runtime node's editor-visible properties, or selected properties for low-token QA. Selected names may use dotted paths such as `player.position` or `state.score`. Absolute paths like `/root/Main` are accepted. |
| `game node set <path> --prop <name> --value <v>` | `game` | ☑ | Set a live runtime node property. Runtime-only, not undoable, and lost when play stops. |
| `game node call <path> <method> [--arg <v> ...]` | `game` | ☑ | Call a live runtime node method and return the stringified `result` and, from v1.2.0, a structured `result_value` (or `result_value_omitted`). Runtime-only and may have side effects. |
| `game assert <path> <prop> <eq\|ne\|contains\|gt\|lt\|exists> [value]` | `game` | ☑ | Assert a live runtime node property with a compact pass/fail response. Designed for generic QA, not a specific game. |
| `game qa discover [path]` | `game` | ☑ | List callable runtime `qa_*` helpers and `Qa` followed by an uppercase letter (e.g. `QaReady`); method names are case-sensitive. Discover methods on the current scene root, or on a specific node path. Returns compact method names, argument names, default counts, and return type when known. |
| `game qa diagnose [--lines N] [--max-errors N] [--max-warnings N] [--path user://capture.png]` | local + tools | ☑ | Run a read-only, project-agnostic runtime health check. It reports editor diagnostic counts, live game-process ambiguity, runtime/UI tree truncation, and screenshot blankness, low detail, or likely clipping. Capture width/height and `size_matches_project` are reported without failing a healthy image that is simply smaller than the project viewport. It does not assume nodes, controls, rules, or QA helper names. |
| `game qa --file <scenario.json> [--continue]` | local + tools | ☑ | Run a generic JSON QA scenario made of `run`, `stop`, `wait`, `game.node.get`, `game.node.set`, `game.node.call`, `game.qa.discover`, `game.ui.tree`, `game.ui.audit`, `game.click`, `game.input`, `game.input_log`, `game.clock`, `game.assert`, `screenshot.runtime`, and `diagnostics` steps. `game.ui.audit` accepts its CLI filters in `params`; failed audit evidence is retained in the step result and does not satisfy `covers`. Known tools, required node fields, assertion operators, run actions, and numeric limits are preflighted before any requests. Diagnostics fail when unavailable or when counts are missing/invalid; omitted `max_warnings` allows warnings, while explicit `0` requires none. The file may be a legacy step array or an object with `requirements` plus `steps`; each step may declare `covers`, and missing or failed requirement coverage makes the scenario fail. |
| `guidance ui` | `guidance` | ☑ | Read the live editor's Game Feel UI Mode setting and return agent-facing UI implementation guidance. When enabled, UI work should favor snappy feedback, expressive state changes, satisfying motion, and runtime visual QA. |
| `guidance game-feel` | `guidance` | ☑ | Read the live editor's Game Feel Mode setting and return gameplay-wide feel guidance: control feel, camera, hit stop, screen shake, sound, particles, rewards, Honest Juice, accessibility, runtime QA, and compact `game_qa_patterns` for reusable prompt-game checks such as ordered QA, primary input, stable inspection, visible state sync, live viewport layout, deterministic helpers, shared route geometry, typed collections, and observable feedback evidence. |
| `game_feel [topic]` | `game_feel` | ☑ | Query the bundled Game Feel knowledge base. No topic or `list` returns the topic index; a topic such as `screen_shake`, `control_feel`, `camera`, `ui_bar`, or `ethics_checklist` returns concrete parameters and constraints. |
| `eval [--lang csharp\|gdscript] <expression>` | `eval` | ☑ | Default C# (`SceneRoot` is the edited root); explicit GDScript uses `Expression` with the scene root as base. Returns string `result` and structured `result_value` (or `result_value_omitted`); C# adds `lang:"csharp"`. See C# setup below. |
| `instances` | local | ☑ | List live Hera-enabled Godot editors discovered from `~/.hera-agent-godot/instances/`. Expired heartbeat files are listed under `stale` (with `age_sec`) instead of being dropped silently. |
| `screenshot [--path <p>] [--width N] [--height N] [--transparent] [--runtime] [--analyze]` | `screenshot` | ☑ | Render the edited scene off-screen to PNG, or capture the running game viewport with `--runtime`. `--width`/`--height` size the off-screen editor render only; runtime captures refuse those flags and do not upscale. `--analyze` is supported for runtime captures and returns generic image/layout metrics, including per-edge content ratios and possible clipping. |
| `screenshot diff <before.png> <after.png> [--threshold N]` | local | ☐ | Compare two captures and report `changed_pixels`, `changed_ratio`, `max_delta` and a `changed_bounds` box locating the change. Runs entirely locally on files already on disk — **no editor needed**. `--threshold` (default 4, per channel 0..255) absorbs anti-aliasing wobble between captures; the two frames must share dimensions. |
| `batch [--file <p>] [--continue]` | `batch` | ☑ | Run a JSON array of `{tool, params}` (stdin or `--file`) in one request, sequentially, including async tools such as `game` and `screenshot`. |
| `smoke [--run-game\|--skip-game]` | local + tools | ☑ | Run a quick live-editor smoke check. `--run-game` also plays the current scene, checks `game tree`, captures/analyzes a runtime screenshot, then stops. |
| `engine info [--project DIR]` | engine | ☐ | Report engine identity without an editor: resolved `path`/`source`, bytes, parsed `version` (`string`, `status`, `mono`, `commit`), `editor_build`, and `help_flags` (which engine flags this build documents). Resolution order is `--engine` → `HERA_GODOT_ENGINE` → `GODOT_BIN`. |
| `engine doctor [--project DIR]` | engine | ☐ | Report engine, export-template, and project checks as `checks[]` (`name`, `ok`, `detail`) plus `warnings[]`: build kind, the `<version>` template directory, `.godot` cache state, and whether the Hera addon is installed and enabled in `project.godot`. |
| `script run <res://tool.gd> [--quit-after N] [-- <args>...]` | engine | ☐ | Run a project script through the engine's headless harness. The script must extend `SceneTree` or `MainLoop`; arguments after `--` arrive via `OS.get_cmdline_user_args()`. Reports `exit_code`, bounded `output`, `duration_ms`, and the resolved engine. Non-harness scripts fail fast as `not_a_harness` because Godot 4.7.2 hangs instead of erroring on them. |
| `project import` | engine | ☐ | Run the engine's import pass (`--import`) to completion and summarize observed artifacts (`.godot` cache, `.uid` sidecars). Engine progress text is localized and is never parsed; the exit code and artifacts decide. |
| `project export <preset> <output> [--kind pack\|release\|debug\|patch] [--patches a,b] [--verbose]` | engine | ☐ | Export with the engine after a fail-fast preflight: preset existence (`available_presets`), target directory, pack extension, and a completed help probe before claiming the build is not an editor build. `pack` needs no templates; a release/debug export only warns when the usual template directory is missing, because a preset can point at custom templates. The raw engine log is **omitted by default** (`output_omitted`) since signing tools and export plugins can log credential-bearing arguments; `--verbose` includes it. Preset contents and `.godot/export_credentials.cfg` are never printed. |
| `capture movie <out.avi\|.png\|.ogv> [--frames N] [--fps N] [--resolution WxH] [--scene <p>]` | engine | ☐ | Record deterministic frames with Movie Maker (`--write-movie` + `--fixed-fps`, bounded by `--frames`). A display driver is required: `--headless` crashed Movie Maker in testing and is never passed. A `res://` output is resolved into the project for inspection. PNG sequences report `frames_written` and `frames_present` from this run's isolated directory (excluding old destination files) and the observed `frame_size`; the recording follows the project viewport, not `--resolution`. Containers report `output_bytes` only — zero frames means unmeasured, not no frames. |
| `perf sample [--headless] [--top N]` | engine | ☐ | Run the engine with `--benchmark-file` and report the costliest phases. Godot 4.7.2 ignores `--quit-after` while benchmark flags are set, so `--timeout` (default 30 s) stops the run and `bounded_by` says whether the engine exited or the timeout ended it. Exit `0` means the benchmark JSON was read, not that the game run succeeded (`engine_error` carries a non-timeout engine failure). |
| `docs dump --out <dir> [--format xml\|json] [--scripts] [--force]` | engine | ☐ | Write documentation with the engine's doctool: engine class-reference XML, `extension_api.json` including docs, or the project's own GDScript reference (`--scripts`). `--out` is required and is created when missing. The destination is resolved through links before the guards run, and scanning follows linked directories (bounded depth; an incomplete scan refuses as `output_scan_failed`), to protect existing documentation before publication: existing `*.xml` is refused as `output_not_empty`, an unscannable destination as `output_scan_failed`, and an existing `extension_api.json` as `output_exists` — each unless `--force`. JSON mode fails as `json_artifact_missing` when no usable file was produced; XML mode reports `xml_files` and fails as `docs_output_empty` when no XML was generated. |
| `classdb dump --out <dir>` | engine | ☐ | Dump the engine ClassDB to JSON. The engine always writes into `user://`, so Hera runs it with an isolated user directory and publishes `class_db_api.json` (`class_db_api_editor.json` adds editor-only classes) into `--out`. Generation and publication are reported separately: `files[]` carries `ok`/`error` per file, `failed_copies` counts failures, and any failed copy is a command failure (`class_db_copy_failed`) instead of a false `class_db_json_missing`; a nonzero engine exit still fails the command. Destinations are replaced through a temporary file, never truncated first. |
| `debug attach [--break res://f.gd:N ...] [--dap-port N] [--wait-ms N] [--project DIR]` | dap | ☐ | Attach to a **running** editor debug server (the editor must be started with `--debug-server` and `--dap-port`; the game with `--remote-debug`, see below). Records a session, sends each breakpoint with MD5/SHA256 checksums, reports what the adapter verified (`breakpoint_results[].verified`), and waits for the next stop: `stopped.frame`/`source.sha256` or `running:true`. Hera never starts or stops the editor or the game. |
| `debug session [--session ID]` | dap | ☐ | Show the recorded session (host, port, project, breakpoints, `attached_at`). `--session` defaults to the most recent attach. |
| `debug stack` / `debug variables [--frame N]` / `debug evaluate <expr> [--frame N]` | dap | ☐ | `stack` is a pure observation: it reports the current stop's frame or `running:true`, and never resumes. `variables` and `evaluate` need a paused program, so a running debuggee is resumed through DAP — the adapter arms breakpoints on resume — and the command waits for the next stop before answering; that side effect is documented in the debug section. `variables` lists locals and members (capped, with `variables_truncated`) as `{scope,name,value}`. Expressions that start with `-` are passed after `--`. |
| `debug continue \| next \| step-in \| step-out [--wait-ms N]` | dap | ☐ | Resume or step, then report the next stop (`stopped.frame`/`source.sha256`) or `running:true`. `step-in`/`step-out` map to DAP `stepIn`/`stepOut`. |
| `debug detach [--session ID]` | dap | ☐ | Disconnect and drop the session record. The editor and game keep running. |

> **Note (`run`):** use `project set-main-scene <res://scene.tscn>` when changing
> the main scene from Hera. Newly added scenes can still require a filesystem
> refresh or project reload before the editor resolves them as PackedScenes.
> After direct `.tscn` file edits, use `stop --wait`, `scene reload [res://...]`,
> then `scene save` so the live editor and disk file have a single synchronized
> writer.

> **Note (mutations):** `node add/instance/set/set-resource/remove/reparent`, `node attach-script/detach-script`,
> `scene open/reload/save/create/save-as`, `editor select/clear-selection`, `script open/create`, `resource set/create`, `project mkdir/scan/reimport`,
> `project set-main-scene`,
> `resource resave/update-uids/export-mesh-library`, and `signal connect/disconnect`
> are mutation commands and enforce the single-editor guard. Node and signal
> mutations register with the editor's undo history where Godot exposes UndoRedo;
> file, import, resource, scene, and project setting changes are persistent
> filesystem/project changes. `signal connect` uses `CONNECT_PERSIST`, so the wiring is saved with
> the scene like the editor's "Connect a Signal" dialog. `eval` runs a single
> expression in C# by default (`SceneRoot`), or via the GDScript `Expression`
> class with `--lang gdscript` and the edited scene root as base. Both can have side
> effects and are not registered with UndoRedo. `game node set/call`, `game click`, `game input`, `game clock` (except a snapshot with no flags), and `game input-log --clear` target the
> running game process, so it is not undoable and its effects disappear when play
> stops. Hera assumes one live editor per project; mutation commands enforce that
> precondition unless `--instance <pid>` is passed explicitly.

## C# eval, validation and run

This development branch defaults `eval` to C# (Godot 4.7+ .NET); standard editors
fail as `csharp_unavailable` and suggest `--lang gdscript`. Explicit language wins
against `HERA_GODOT_EVAL_LANG`; the same rule applies to batch eval children.
Build and reload the project with the Hera native loader, then build the separate
[compiler helper](CSHARP_SUPPORT.md#compiler-helper). Global `--csharp-host PATH`
overrides `HERA_GODOT_CSHARP_HOST`; adjacent/private-home fallback paths are in the
C# guide. Missing helper/project errors do not silently select GDScript.

```sh
hera-godot --csharp-host dist/csharp/Hera.CSharpHost.dll eval '((Node2D)SceneRoot).Position'
hera-godot eval --lang gdscript 'get_child_count()'
hera-godot script validate res://Player.cs
hera-godot --engine /path/to/Godot.NET script run res://Tool.cs -- one two
```

`SceneRoot` is the edited root. C# success adds `lang:"csharp"` and reuses the
existing result encoding. Arbitrary expressions can block the editor, retain
an ALC, and mutate without undo; timeout does not cancel editor execution.
Non-Tool C# scripts can be placeholders rather than user-class instances.

Validation redirects restore/build output and intermediates, requires exactly
one root csproj with no ProjectReference items, and rejects an excluded Compile
item. It reports requested-file diagnostics and `other_files_errors`; `valid`
requires a successful complete build and unchanged requested disk bytes.
MSBuild targets/generators execute code; output redirection is not a sandbox.
Global `--timeout MS` overrides its 60-second combined build deadline; separate
editor HTTP defaults to 5 seconds. Create/open/attach/run never build for you.
C# run requires a prebuilt Debug DLL and directly runs a SceneTree/MainLoop `.cs`
harness. It does not certify source/assembly freshness.

C# debug breakpoints fail before connection with `csharp_debug_unsupported`
(exit 2), `guidance` and `docs_url`; see [IDE instructions](CSHARP_SUPPORT.md#c-debugging--c-디버깅).
All C# failure codes and partial result fields are listed in
[the development contract](CONTRACT.md#v2-development-changes-not-a-released-version).

C# eval 기본값은 프로젝트 언어와 무관합니다. 표준 에디터는 자동 대체하지 않으며
GDScript 식은 `--lang gdscript`로 선택합니다. C# 검증은 격리 출력 경로의 빌드이며
실행·undo·샌드박스 보장이 아닙니다. C# 디버깅은 IDE에서 수행하세요.

## Fresh GDScript validation

C# `script validate res://Script.cs` uses a 60-second default deadline for SDK
discovery, input query, restore and build together. Explicit `--timeout MS`
overrides it; the separate editor HTTP request still defaults to five seconds.

`hera-godot script validate res://scripts/player.gd` selects a live editor (use
`--instance` when several are open), obtains its engine/project paths, and runs
that engine with `--headless --check-only --script` in a separate process.
It checks the file on disk, including relative dependencies, rather than the
editor's cached script or unsaved text. It neither starts the scene nor builds C#.
Script/dependency loading can execute static initializers: this is not a sandbox.

The JSON result contains `path`, `valid`, `exit_code`, `output`,
`output_truncated`, and `timed_out`. Engine output is capped at 64 KiB.
Invalid scripts or timeouts exit 1; invalid CLI arguments exit 2. The engine
process has a five-second default deadline; leading `--timeout MS` overrides
both the HTTP timeout and this process deadline. Engine output includes native
file/line details when available; success does not certify absence of warnings.
This is a CLI operation; the internal `script/validate-context` RPC only
resolves the paths and does not validate code when called through `batch`.

## Conditional node property changes

Read `hera-godot node get <path> --prop <name> --snapshot` to obtain an `expected`
object, then pass that object as `node set --expected <JSON>`. Ordinary reads
keep their existing output. The complete object has exactly six string fields:

```json
{"editor_session_id":"session-from-editor","scene":"res://Main.tscn","node_instance_id":"9876543210","prop":"visible","type":"bool","value":"true"}
```

`--verify` requires `--expected`. The CLI negotiates
`capabilities.node_set_guard: supported` (`hera-godot status --capabilities`) on the same discovered
connection before snapshot or guarded requests. This also covers guarded
entries in `batch`, before any entry runs. Missing/unsupported/unverified
capabilities fail with `capability_unavailable`; no unguarded retry occurs.
An internal guarded action also makes old addons reject a write if the endpoint
is replaced after preflight.
Multiple live editors still require explicit `--instance` for mutations.

The addon compares session, current scene path, node instance, property name,
declared type, and typed current value immediately before registering undo or
calling the setter. `session_mismatch` and `state_conflict` reject the change
without setter, undo, or disk-save effects. Instance IDs are decimal strings,
valid only in the session, so replacing a node at the same path conflicts.
Empty scene paths describe unsaved scenes; node identity still distinguishes
them. No scene hash or persistent node identity is implied.

Supported types: `bool`, `int`, `float`, `String`, `StringName`, `NodePath`,
`Vector2`, `Vector2i`, `Vector3`, `Vector3i`, `Rect2`, `Rect2i`, and `Color`.
Snapshot values use lossless Godot Variant text, except string-like values
which remain literal strings. A snapshot that cannot roundtrip exactly fails
with `capability_unavailable`. Other types (including collections and resources)
are deliberately unsupported. Existing displayed property strings may be
truncated; use the complete `expected.value`, not that display, for guards.

Success returns the original `path`, `prop`, `value` plus `editor_session_id`,
`scene`, `node_instance_id`, and `verification` (`passed` or `not_requested`).
The same target is re-read after setting. With `--verify`, a setter that
clamps/rejects the requested typed value fails with `verification_failed`;
loss of the original session/scene/target fails with `verification_unavailable`.
Both failures report that mutation was applied and is not rolled back.
Custom setters/other plugins can have side effects; this is not a transaction.
No save is performed. `batch` retains its sequential, nontransactional semantics.

## Physics-frame input sequences

`hera-godot game input sequence --file events.json` reads a JSON array:

```json
[
  {"frame": 0, "action": "ui_accept", "pressed": true},
  {"frame": 3, "action": "ui_accept", "pressed": false}
]
```

Frame 0 is the next physics frame. This example holds `ui_accept` for three
physics callbacks. Events must be ordered by frame (equal frames are allowed),
with 1–128 entries, frame offsets 0–120, existing InputMap action names, and
boolean `pressed` fields. The CLI file is bounded to 64 KiB. The tree must be
unpaused, `Engine.time_scale` must be 1, and the physics rate must be at least
60 ticks per second. Already-held actions are rejected before any injection.

The runtime validates the whole sequence first, flushes input at each scheduled
physics frame, and releases sequence-held actions after the last event frame.
It cancels and releases input on clock changes or a 2.5-second wall-clock
deadline. Other input/clock mutations fail while a sequence or clock step is
active. Runtime node calls and external human input can still affect the game;
this does not make arbitrary game logic deterministic.

Success returns `kind: "sequence"`, `pid`, `frames` (last offset plus one),
and `events[]` with requested fields plus the observed `physics_frame` counter.
For requirement-based QA, use `game.input` with
`params: {"kind":"sequence","events":[...]}`, followed by `game.assert`
steps whose `covers` name the gameplay requirements. `game --pid N` selects a
specific runtime just as for other input commands.

`game clock --step` now pauses until the selected frame starts, permits its
node callbacks to complete, and pauses again. Only the selected callback phase
is counted; this is not a debugger step and does not stop `PROCESS_MODE_ALWAYS`
nodes. Reparent undo restores the original node name after a name collision,
as well as the original parent, sibling index, and owner.

## Script languages

The `.gd` or `.cs` extension is required and selects the language; Hera never
switches it based on project detection. Optional `--lang gdscript|csharp` must
agree. See [C# support](CSHARP_SUPPORT.md) for setup.

C# templates derive the `partial` class name from the filename; an explicit
`--class-name` must match. Lifecycle flags generate C# overrides (`_Ready()`,
`_Process(double delta)`, and the corresponding input/physics callbacks).
`--signal Hit` generates `[Signal] public delegate void HitEventHandler();`.
`--export Speed:float=3.5f` uses a C# type and initializer expression, not Godot
Variant text. `--tool` generates `[Tool]`.

C# creation returns `language: "csharp"`, `build_required: true`, and
`build_warning`. Hera does not generate `.csproj`/solution files or run builds.
Build and reload the assembly before attachment; an unloaded class is rejected.
Successful C# attachment adds `language` and `build_warning` to
`script_diagnostics`; its existing preload arrays are not C# diagnostics.

C# inspect/current returns `metadata_source: "assembly"`, `assembly_loaded`,
`csharp_supported`, and `build_warning`. `class_name` derives from the filename;
`base_type`/`extends` describe the engine-native base when available. Functions,
signals, and exports are empty before the assembly is loaded. Loaded metadata
can still be stale relative to source. Inspect is available on standard Godot
with unavailable assembly metadata; create/open/attach require Godot .NET.
`script current` sees only Godot's focused script resource, not an external IDE.
Existing GDScript responses remain unchanged.

## Runtime process selection

Place `--pid <game-pid>` immediately after `game` to target any game or QA
subcommand, including `qa discover`, `qa diagnose`, and scenario game steps.
This PID comes from `game instances`; global `--instance <editor-pid>` still
selects the editor that transports the request. Positive integers are accepted.
A missing, expired, malformed, zero, or negative PID fails without selecting a
different runtime. Unqualified requests retain editor-play scene matching and
refuse multiple matching processes. Explicitly targeted responses add
`game_pid` and `game_scene`, so callers can verify the selected process.

The `HeraGameInspector` autoload is an editor-play aid. Hera removes its own
autoload before Godot collects export dependencies and serializes exported
project settings, then restores it after export. Exported games therefore do
not reference the inspector; the export preset still decides whether unrelated
addon files are packaged. Disabling the plugin also removes a persisted
Hera-owned autoload; a same-named autoload pointing elsewhere is preserved.

## Global flags

Global flags go **before** the command (e.g. `hera-godot --ids node find`,
`hera-godot --instance 2840 node add Node2D`).

| Flag | Status | Meaning |
|------|--------|---------|
| `--json` | ☑ | Pretty-print the response Data. |
| `--ids` | ☑ | Print only paths from `nodes[]` or `controls[]` (`scene tree`, `node find`, `game ui tree`); compact JSON otherwise. |
| (default) | ☑ | Compact JSON — minimal tokens. |
| `--instance <pid>` | ☑ | Explicitly target an editor by pid (from `status`); also satisfies the single-editor mutation guard. Accepts `--instance N` or `--instance=N`. |
| `--timeout <ms>` | ☑ | Per-request HTTP timeout in milliseconds (default 5000); also separately bounds the engine child process for `script validate`. It does not bound a whole polling command (`--wait` sends many requests). Accepts `--timeout N` or `--timeout=N`. |
| `--engine <path>` | ☐ | Godot engine executable for commands that do not need a live editor (`engine`, `script run`, `project import/export`, `capture movie`, `perf sample`, `docs dump`, `classdb dump`). Accepts `--engine PATH` or `--engine=PATH`; the same flag is accepted after those commands. Resolution order: flag → `HERA_GODOT_ENGINE` → `GODOT_BIN`. Missing or non-file paths fail before any engine process starts. |

## Engine-backed commands (no live editor)

These commands run the real Godot engine as a bounded child process, so they work
in CI or on a machine where no editor is open. They share one contract:

- **Engine resolution.** `--engine <path>` (global, or after the command) wins,
  then `HERA_GODOT_ENGINE`, then `GODOT_BIN`. A missing or non-file path fails on
  stderr with exit `1` before any process starts. On Windows a `*_console.exe`
  path is resolved to the sibling engine executable when it exists, because the
  console wrapper starts the real engine as a child process. Nothing here
  requires a live editor, an addon, or a heartbeat.
- **Bounding.** `--timeout <ms>` bounds the engine process, and one context is
  shared by the metadata probes and the workload of a single command.
  Per-command defaults: `engine info`/`doctor` 10 s, `script run` 60 s,
  `perf sample` 30 s, `capture movie` 180 s, `project import`,
  `project export`, `docs dump` and `classdb dump` 300 s. An already-expired
  context launches nothing, and pipe draining is bounded (`WaitDelay`).
- **Process ownership.** A timeout terminates work through an owned identity, not
  a recycled numeric id. Windows resolves a `*_console.exe` wrapper to the real
  engine executable, assigns the child to a Job Object (kill-on-close) right
  after launch and keeps a process handle, so the tree is reachable even when the
  started process already exited; Unix signals only the started process through
  Go's own bookkeeping, so **Unix descendants are not terminated** (that is
  deliberate: a numeric group kill after reaping could hit an unrelated group).
  `termination_error` reports incomplete cleanup — a failed job termination, a
  missing job, or a start that could only be stopped alone. Windows children are
  launched suspended, so a descendant cannot be spawned before the job assignment.
  The bound applies only **after** the command context is cancelled: a healthy run
  is never cut short, while waiting for a process that termination could not stop
  is bounded and reported (`abandoned: true`, `...the process exit was not observed
  within the wait bound`) instead of
  hanging the caller. Refusal, cancellation and natural-exit cleanup use one
  worker owning termination and close. The caller waits at most 5 s for exit after
  refusal/cancellation, then at most 5 s for cleanup. Otherwise-successful runs with unfinished
  cleanup fail `engine_cleanup_incomplete` (timeout/abandonment take precedence);
  `cleanup_pending:true` reports any
  unfinished worker. Native calls cannot be interrupted but no termination/close
  call runs on the returning caller. Launch, ownership setup and resume remain
  synchronous. A stuck worker retains its guard until CLI teardown;
  closing it while the native call uses it would violate handle ownership. Unix
  launches are not suspended, and descendants are never enrolled in an owned tree:
  this is a persistent limitation, not merely a pre-assignment window.
  The test-only delay/factory seams and CLI globals require serialized in-process
  command execution; they are not a concurrent library API.
- **Failure vocabulary.** Any of `engine_timeout`, `engine_cleanup_incomplete`, `engine_abandoned` (the process
  exit was never observed, so nothing about its work is proven), `engine_exit_nonzero`,
  `output_incomplete` (the process exited but something kept its pipes open, so
  captured output may be truncated), or a launch error fails the command; the
  JSON result carries `error` and a one-line `error: <code>` diagnostic goes to
  stderr. `perf sample` is the documented exception: it exits `0` when the
  benchmark JSON parsed, and reports a nonzero engine exit as `engine_error`.
- **Reading results.** Engine stdout/stderr is captured, bounded to 64 KiB, and
  reported as `output` with `output_truncated`. That text is localized and
  ANSI-colored, so **status is never parsed from it**. Success comes from the
  exit code plus observed artifacts (files, sizes, counts), and from
  machine-readable artifacts such as `--benchmark-file` when the engine writes
  one: artifacts that are missing after a clean exit fail the command
  (`no_frame_produced`, `no_artifact_produced`, `docs_output_empty`,
  `docs_scan_failed`, `json_artifact_missing`). Capture and docs generate into an
  empty `.hera-run-*` directory inside the destination (inside the output file's
  parent for capture). Existing destination files cannot satisfy verification,
  even when a new artifact has identical bytes/timestamps. Destination metadata
  comparisons and `json_artifact_stale` are no longer used. An unlistable PNG
  sequence fails `sequence_scan_failed` and omits all frame counts; an unavailable
  container stat fails `artifact_stat_failed` and omits existence/byte fields.
  `artifact_source:"isolated_run_directory"` describes provenance, not protection
  against another process deliberately modifying that staging directory. Counts,
  sizes and `output_exists` describe generated artifacts; output paths name intended
  destinations. Only `published:true` confirms publication and staging removal.
  Failure retains `staging_dir` with `published:false`; do not delete it while an
  abandoned engine may still use it. `files_published` counts completed file
  publications. Publication is per-file, not transactional: failures can leave a
  partially published set. `artifact_stage_failed` and `artifact_publish_failed`
  fail the command. Capture replaces matching names including WAV sidecars; docs
  requires `--force` to replace existing output. Neither removes older unmatched
  files, so shorter recordings can leave older destination frames. Concurrent
  publishers can replace the same final file. Publication refuses non-regular
  generated files and confines destination traversal with `os.Root`.
  Export remains existence-only; `perf sample` judges its benchmark artifact.
  `exit_code` is `-1` when no numeric exit status is available — a pre-launch
  refusal, an abandoned process, or a process reaped after a signal on Unix — so
  `-1` is never a successful exit. `project export` omits the raw log unless
  `--verbose` is passed, because signing tools and export plugins can log
  credential-bearing arguments.
  This describes the engine status: `perf sample` can still exit the CLI with 0
  when benchmark JSON is readable, even with engine `exit_code: -1`.
- **Filesystem work.** Destination scans, stat calls and publication are synchronous
  filesystem operations outside the engine-process timeout. Large or slow destinations
  can delay the CLI; XML is walked for destination protection and generated-artifact counts.
  The depth cap limits recursion, not file count or elapsed time.
- **Failures.** A structured pre-flight failure prints its JSON with
  `ok:false`/`error` on stdout, a one-line `error: <code>` diagnostic on stderr,
  and exits `1`. Missing and unreadable inputs are reported apart
  (`export_presets_missing` vs `export_presets_unreadable`,
  `output_directory_missing` vs `output_directory_unreadable`).
- **Project scope.** `--project <dir>` selects the project; the working directory
  is used when it holds a `project.godot`. There is no implicit search.
- **Writes.** `project import`, `project export`, `capture movie`, `docs dump`,
  and `classdb dump` create files. Hera writes the named destinations and their per-run staging directories,
  and `classdb dump` replaces a destination through a temporary file instead of
  truncating it first. The **engine**, however, is a separate writer: `--import`
  and export regenerate `.godot/**` and `.uid` sidecars, while `docs dump` regenerates in staging and publishes matching output files;
  replacing existing documentation requires `--force`.
- **Secrets.** `export_presets.cfg` and `.godot/export_credentials.cfg` can hold
  keys and passwords. Hera reads preset **names** only, never prints option
  values or platform signing flags, and omits export logs unless `--verbose`.

Verified constraints of Godot 4.7.2 that shape these commands (see
[GODOT_EDITOR_ANALYSIS.md](./GODOT_EDITOR_ANALYSIS.md) §2 for the evidence):

| Constraint | Consequence |
|---|---|
| `--headless` uses the dummy renderer (null viewport textures, 64×64 visible rect) | capture and screenshots need a real display driver |
| `--headless` + `--write-movie` crashed (signal 11) | `capture movie` never passes `--headless` |
| `--benchmark`/`--benchmark-file` ignore `--quit-after` | `perf sample` is bounded by the timeout and reports `bounded_by` |
| `--script` hangs on non-`MainLoop` scripts instead of erroring | `script run` pre-flights the script's base type (`not_a_harness`) |
| `--doctool` needs an existing directory; `--gdscript-docs` needs a project-relative path | `docs dump` creates `--out` and passes `res://` for `--scripts` |
| Movie Maker sizes the recording from the project viewport | PNG sequences report the observed `frame_size` instead of echoing `--resolution` |

## Debug sessions (GDScript DAP)

`debug` attaches to a running editor's debug server; it is a DAP client, not a
launcher. Start the two ends yourself (or from a script):

```
godot --headless --path <project> --editor --debug-server tcp://127.0.0.1:6007 --dap-port 6006
godot --path <project> --remote-debug tcp://127.0.0.1:6007      # the game, windowed
hera-godot debug attach --project <project> --dap-port 6006 --break res://main.gd:12
```

Verified behavior behind these commands (Godot 4.7.2, see
[GODOT_EDITOR_ANALYSIS.md](./GODOT_EDITOR_ANALYSIS.md) §2):

- **`attach` works, `launch` does not.** The adapter answered `initialize`,
  `attach`, `setBreakpoints` and `configurationDone`, and a real breakpoint then
  stopped the program with its source checksums; a `launch` request produced no
  response in headless or windowed editors.
- **The adapter arms breakpoints on resume.** A stop after `setBreakpoints`
  appears once the program continues through DAP, so attach observes the current
  state first (`already_paused` or `running`), `stack` never resumes, and only
  `variables`/`evaluate` resume a running debuggee to its next stop because they
  cannot read a running program. A stop carries `attribution: "unproven: …"`: the
  quiet window before an action and the adapter's own resume event reduce, but do
  not prove, that a stop belongs to this request. The one exception is an
  `already_paused` attach observation, which Hera did not cause and which therefore
  reports `attribution: "not_applicable: …"`. A resume the adapter
  never confirms is a failure, not a result. Reconciliation uses a floor of
  20 ms, so `--wait-ms` values below that still reconcile before the action.
- **Checksums have provenance.** `setBreakpoints` carries the script's MD5 and
  SHA256 **as local disk hashes** (`checksums_source: local_disk`), and a stop's
  `source.sha256` is **adapter-reported** (`checksums_source:
  adapter_reported`, `sha256_available` says whether it arrived). Neither proves
  which revision the running game loaded: editing the file after launch can make
  disk and executing code differ, and Hera performs no revision comparison.
- **Adapter verdicts are checked.** A response with `success:false`, a response
  addressed to another command, a rejected adapter message or a partial frame all
  fail the command instead of being reported as a result.
- **Sessions are endpoint recipes.** Each command reconnects to the recorded
  host/port, re-attaches and re-applies the recorded breakpoints
  (`~/.hera-agent-godot/debug/<id>.json`, `latest` points at the newest). A session
  record is not proof that the original editor, game or paused state still
  exists: reconnecting addresses whatever now answers on that endpoint, and an
  explicit `--host`/`--dap-port` override is reported through
  `limits.host_override` / `limits.port_override`. Session ids are validated, a record that does
  not describe itself correctly is refused, and the store never falls back to a
  shared temporary directory.
- **Hera owns no debuggee.** The editor and game are started and stopped by you;
  `debug detach` only closes the connection.
- **Known DAP limits.** A response whose `command` field is empty is accepted (only
  a different non-empty name is rejected); the `initialized` event is consumed
  when present but not awaited before breakpoint configuration. This is the current
  experimental Godot adapter handshake, not a promise of generic DAP compatibility.
  Concurrent attaches publish `latest` with last-publication-wins semantics, without
  a single-writer lock. Detach deletes only its session record and never touches
  `latest`, so it cannot erase a newer attach. If `latest` names a detached record,
  default-session commands fail until another attach; use `--session ID` for another
  retained record. Endpoint continuity cannot be proven;
  and Godot's adapter provides no causal request boundary, so a stop's
  `attribution` is reported as unproven rather than guaranteed (an `already_paused`
  observation, which Hera did not cause, reports `not_applicable` instead).
  Stack and resume/step discover `threads`. Use `--thread ID` on attach, stack,
  variables, evaluate, continue or step commands to select per invocation (not
  persisted in the session). Otherwise a buffered stopped event selects its thread,
  or a sole reported thread is used. Ambiguity fails with available IDs before a
  resume. Post-resume stacks follow the stopped event's thread; stacks and stops
  expose `thread_id`. The adapter may resume all threads despite selecting one.
  Generic-adapter compatibility remains unpromised. The one `variables` recovery retry is read-only: it never
  resumes again, and fails if the debuggee is no longer paused.

## Editor log evidence

`--source file` is the unchanged default. `--source editor` requires
`capabilities.editor_log_cursor: supported` (`hera-godot status --capabilities`); the CLI checks before
sending log parameters, so legacy addons cannot silently return file evidence.
API absence is `unsupported`; registration or callback verification failure is
`unverified`. Both produce `available:false`, `clean:false`, and
`reason:evidence_unavailable`, without zero counts. File-read failures also
carry this reason, while retaining their existing response fields.

The editor collector retains 1024 events. `--lines` accepts 1–1024 for this
source (defaults: output 100, diagnostics 20). Output returns the latest matching
`entries` with `sequence`, `severity` (`log`, `warning`, `error`), `message`,
`location` (`function`, `file`, `line`), native `error_type` (-1 for plain
messages), and `truncated`. Messages retain at most 4096 characters; location
strings retain 512 each. Script and shader failures count as errors; ordinary
stderr messages also count as errors. No backtrace variables are collected.

Both commands return `source`, `editor_session_id`, `cursor`, `restart_cursor`,
`dropped_count`, `complete`, and `available`. Cursors are opaque: save a returned
cursor, perform work, then pass `--since <cursor>` to read later events. They
delimit observations, not causality. `dropped_count` is cumulative overwritten
events in this session, even when the requested interval is complete.
Output's `total` counts matches before the tail limit, and `omitted_count`
reports matches excluded by that limit. The returned cursor advances past all
observed events, including filtered/omitted events. Use `--lines 1024` to read
every retained match. Diagnostics counts all matching retained events and limits
only its `errors`/`warnings` samples. `clean` is false if history is incomplete.

An old-session, malformed, future, or overwritten cursor returns
`reason:cursor_expired`, `available:false`, `clean:false`, and a
`restart_cursor` immediately before the oldest retained event. Explicitly use
that restart cursor to begin a new interval after acknowledging the gap; Hera
does not silently reset it. Re-enabling the plugin creates a new session.

Coverage starts when the collector registers in this editor process. Earlier
startup logs, external editor/game processes, disabled engine output streams,
and un-emitted analyzer warnings are outside that evidence. Keep startup
`--log-file <path>` collection when needed; Hera does not relaunch the editor
or automatically read that separate file. Raw RPC/batch callers must check the
capability themselves before using an older addon. QA scenario diagnostics
remain on their existing file source.

See [CONTRACT.md](./CONTRACT.md) for the output contract (exit codes, error
shapes, stability tiers), [ARCHITECTURE.md](./ARCHITECTURE.md) for the request
lifecycle, and [ROADMAP.md](./ROADMAP.md) for delivery order.

### Mutation and transport boundaries

`node remove` rejects the resolved scene root, including aliases such as `Branch/..`; removal and reparent undo restore each descendant's original owner. Resource and theme property batches validate all entries before applying setters. Native JSON resource values must match the property type; integral, finite, in-range JSON numbers are accepted for integer properties. Use typed Godot variant text for packed arrays. Disk save failures are still reported after application and do not provide transactional rollback.

Runtime Control bounds and click centers include canvas transforms. Node-targeted clicks reject Controls in another Viewport. Runtime requests are published by temporary-file rename, and method calls retain the original target path even when the method frees the node. HTTP responses use bounded partial writes and a separate five-second write deadline.

QA `run` steps with `wait:true` wait for the requested play action, or for stopped editor/runtime state when `action:"stop"`. `action:"state"` remains a snapshot even when `wait:true`. The dedicated `stop` step uses the same stop-wait behavior.

The v1.2.0 installed command is `hera-godot` (`hera-agent-godot` remains an alias). New installs do not register `hera`; see [upgrade steps](MIGRATING_TO_V1.md#upgrade) before updating scripts or removing old binaries.

## Structured results from call and eval

```sh
hera-godot game node call /root/Main qa_state
# {"method":"qa_state","path":"/root/Main","result":"{ \"hp\": 3 }","result_value":{"hp":3},"type":"Dictionary"}
hera-godot eval --lang gdscript 'Vector2(1, 2)'
# {"result":"(1.0, 2.0)","result_value":"Vector2(1, 2)","type":"Vector2","undoable":false}
```

Read `result_value` directly; do not parse the legacy `result` string as JSON.
`result_value_omitted` reports `depth_limit`, `item_limit`, `size_limit`, or
`key_collision`. If both fields are absent, the producing editor/runtime is
older. `null` and `false` values still count as present. The text in `result`
can differ by engine version. See the experimental
[encoding and compatibility rules](CONTRACT.md#structured-call-and-eval-results-experimental).

## Install the compiler helper / 컴파일러 헬퍼 설치

```sh
hera-godot csharp-host install --version v2.0.0
```

Release builds default to their own tag; development builds require `--version`.
The explicit command downloads `hera-csharp-host.zip` and `checksums.txt` from
GitHub, verifies SHA256, and replaces `~/.hera-agent-godot/csharp/` after staging.
Existing helper files are replaced as a directory; close active helper processes
before upgrading. Failures before replacement preserve the old installation.
It needs no live editor and does not change project files. Default download
budget is 120 seconds; global `--timeout MS` overrides it. Success is compact
JSON (`installed`, `version`, `path`); usage errors exit 2, installation failures
exit 1 with `csharp_host_install_failed` on stderr. Checksums provide integrity
against the downloaded release list, not an independent signature.

npm, Scoop and Homebrew install only the CLI. Install the helper separately;
`eval` never downloads it automatically. .NET 8 or a compatible newer runtime
is required to execute the framework-dependent helper.

패키지 매니저는 CLI만 설치합니다. 위 명령으로 헬퍼를 별도 설치하세요. SHA256을
검증한 후 사용자 홈의 헬퍼 폴더를 교체하며 사용자 프로젝트는 변경하지 않습니다.
eval은 자동 다운로드하지 않습니다. 헬퍼 실행에는 .NET 8 이상 호환 런타임이 필요합니다.
