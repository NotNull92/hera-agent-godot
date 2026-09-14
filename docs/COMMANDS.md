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
unavailable evidence, including while paused or at time scale zero.

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
`status.capabilities.linked_evidence` advertises this bounded addon surface.

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
assembly readiness. `status.capabilities.editor_mutation_gate` reports the gate;
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
cancels only accepted, queued work. These commands use the selected editor and
require `--instance` when several editors are live. Legacy commands are unchanged.

The caller constructs and retains `id` as `EDITOR_SESSION:UNIX_MS:NONCE`, using
`status.editor_session_id`, an absolute execution deadline no more than 60 seconds
ahead, and a unique ASCII alphanumeric/underscore/hyphen nonce. Example PowerShell:

```powershell
$session = (hera status | ConvertFrom-Json).editor_session_id
$deadline = [DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds() + 30000
$operationId = "${session}:${deadline}:change1"
# $request contains one supported {tool,params} object described below.
hera operation submit $operationId --request $request
hera operation status $operationId
hera operation cancel $operationId
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
`outcome_unknown`. Inspect receipt fields even when CLI exit is zero: that exit
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
| `status` | `status` | ☑ | Show the connected editor: project path, Godot version/commit, active scene, `editor_session_id`, API `capabilities`, `csharp_supported` editor capability (not SDK availability), and Game Feel modes. |
| `run [--scene <res://...>] [--current] [--wait]` | `run` | ☑ | Play the main scene (default), the current scene (`--current`), or a specific scene (`--scene`). `--wait` polls until the matching runtime scene is inspectable. |
| `stop [--wait]` | `run` | ☑ | Stop the running scene. `--wait` polls until stopped. |
| `output [--type log\|error\|warning\|all] [--lines N] [--source file\|editor] [--since cursor]` | `output` | ☑ | Default: tail the project's configured file log. Opt-in editor source: bounded session evidence with cursors and severity/location metadata; see below. Unreadable evidence reports `available:false`. |
| `diagnostics [--lines N] [--source file\|editor] [--since cursor]` | `diagnostics` | ☑ | Default: summarize the project's configured file log. Opt-in editor source: counts over the retained observed interval. Unavailable evidence is never reported clean. |
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
| `game node call <path> <method> [--arg <v> ...]` | `game` | ☑ | Call a live runtime node method and return the stringified result. Runtime-only and may have side effects. |
| `game assert <path> <prop> <eq\|ne\|contains\|gt\|lt\|exists> [value]` | `game` | ☑ | Assert a live runtime node property with a compact pass/fail response. Designed for generic QA, not a specific game. |
| `game qa discover [path]` | `game` | ☑ | List callable runtime `qa_*` helpers and `Qa` followed by an uppercase letter (e.g. `QaReady`); method names are case-sensitive. Discover methods on the current scene root, or on a specific node path. Returns compact method names, argument names, default counts, and return type when known. |
| `game qa diagnose [--lines N] [--max-errors N] [--max-warnings N] [--path user://capture.png]` | local + tools | ☑ | Run a read-only, project-agnostic runtime health check. It reports editor diagnostic counts, live game-process ambiguity, runtime/UI tree truncation, and screenshot blankness, low detail, or likely clipping. Capture width/height and `size_matches_project` are reported without failing a healthy image that is simply smaller than the project viewport. It does not assume nodes, controls, rules, or QA helper names. |
| `game qa --file <scenario.json> [--continue]` | local + tools | ☑ | Run a generic JSON QA scenario made of `run`, `stop`, `wait`, `game.node.get`, `game.node.set`, `game.node.call`, `game.qa.discover`, `game.ui.tree`, `game.ui.audit`, `game.click`, `game.input`, `game.input_log`, `game.clock`, `game.assert`, `screenshot.runtime`, and `diagnostics` steps. `game.ui.audit` accepts its CLI filters in `params`; failed audit evidence is retained in the step result and does not satisfy `covers`. Known tools, required node fields, assertion operators, run actions, and numeric limits are preflighted before any requests. Diagnostics fail when unavailable or when counts are missing/invalid; omitted `max_warnings` allows warnings, while explicit `0` requires none. The file may be a legacy step array or an object with `requirements` plus `steps`; each step may declare `covers`, and missing or failed requirement coverage makes the scenario fail. |
| `guidance ui` | `guidance` | ☑ | Read the live editor's Game Feel UI Mode setting and return agent-facing UI implementation guidance. When enabled, UI work should favor snappy feedback, expressive state changes, satisfying motion, and runtime visual QA. |
| `guidance game-feel` | `guidance` | ☑ | Read the live editor's Game Feel Mode setting and return gameplay-wide feel guidance: control feel, camera, hit stop, screen shake, sound, particles, rewards, Honest Juice, accessibility, runtime QA, and compact `game_qa_patterns` for reusable prompt-game checks such as ordered QA, primary input, stable inspection, visible state sync, live viewport layout, deterministic helpers, shared route geometry, typed collections, and observable feedback evidence. |
| `game_feel [topic]` | `game_feel` | ☑ | Query the bundled Game Feel knowledge base. No topic or `list` returns the topic index; a topic such as `screen_shake`, `control_feel`, `camera`, `ui_bar`, or `ethics_checklist` returns concrete parameters and constraints. |
| `eval <expression>` | `eval` | ☑ | Evaluate one GDScript expression in either project language (`Expression` class, scene root as base) and return the result. |
| `instances` | local | ☑ | List live Hera-enabled Godot editors discovered from `~/.hera-agent-godot/instances/`. Expired heartbeat files are listed under `stale` (with `age_sec`) instead of being dropped silently. |
| `screenshot [--path <p>] [--width N] [--height N] [--transparent] [--runtime] [--analyze]` | `screenshot` | ☑ | Render the edited scene off-screen to PNG, or capture the running game viewport with `--runtime`. `--width`/`--height` size the off-screen editor render only; runtime captures refuse those flags and do not upscale. `--analyze` is supported for runtime captures and returns generic image/layout metrics, including per-edge content ratios and possible clipping. |
| `screenshot diff <before.png> <after.png> [--threshold N]` | local | ☐ | Compare two captures and report `changed_pixels`, `changed_ratio`, `max_delta` and a `changed_bounds` box locating the change. Runs entirely locally on files already on disk — **no editor needed**. `--threshold` (default 4, per channel 0..255) absorbs anti-aliasing wobble between captures; the two frames must share dimensions. |
| `batch [--file <p>] [--continue]` | `batch` | ☑ | Run a JSON array of `{tool, params}` (stdin or `--file`) in one request, sequentially, including async tools such as `game` and `screenshot`. |
| `smoke [--run-game\|--skip-game]` | local + tools | ☑ | Run a quick live-editor smoke check. `--run-game` also plays the current scene, checks `game tree`, captures/analyzes a runtime screenshot, then stops. |

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
> expression via the `Expression` class (not full GDScript statements), with the
> edited scene root as the base instance. Expressions can call methods with side
> effects and are not registered with UndoRedo. `game node set/call`, `game click`, `game input`, `game clock` (except a snapshot with no flags), and `game input-log --clear` target the
> running game process, so it is not undoable and its effects disappear when play
> stops. Hera assumes one live editor per project; mutation commands enforce that
> precondition unless `--instance <pid>` is passed explicitly.

## Fresh GDScript validation

`hera script validate res://scripts/player.gd` selects a live editor (use
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

Read `hera node get <path> --prop <name> --snapshot` to obtain an `expected`
object, then pass that object as `node set --expected <JSON>`. Ordinary reads
keep their existing output. The complete object has exactly six string fields:

```json
{"editor_session_id":"session-from-editor","scene":"res://Main.tscn","node_instance_id":"9876543210","prop":"visible","type":"bool","value":"true"}
```

`--verify` requires `--expected`. The CLI negotiates
`status.capabilities.node_set_guard: supported` on the same discovered
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

`hera game input sequence --file events.json` reads a JSON array:

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

Global flags go **before** the command (e.g. `hera --ids node find`,
`hera --instance 2840 node add Node2D`).

| Flag | Status | Meaning |
|------|--------|---------|
| `--json` | ☑ | Pretty-print the response Data. |
| `--ids` | ☑ | Print only node paths (for `scene tree` / `node find`); compact JSON otherwise. |
| (default) | ☑ | Compact JSON — minimal tokens. |
| `--instance <pid>` | ☑ | Explicitly target an editor by pid (from `status`); also satisfies the single-editor mutation guard. Accepts `--instance N` or `--instance=N`. |
| `--timeout <ms>` | ☑ | Per-request HTTP timeout in milliseconds (default 5000); also separately bounds the engine child process for `script validate`. It does not bound a whole polling command (`--wait` sends many requests). Accepts `--timeout N` or `--timeout=N`. |

## Editor log evidence

`--source file` is the unchanged default. `--source editor` requires
`status.capabilities.editor_log_cursor: supported`; the CLI checks before
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
