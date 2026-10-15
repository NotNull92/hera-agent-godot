# Output Contract

> Status: **stable as of v1.0.0**. Field lists below were captured from a live
> Godot 4.7 editor and are protected by contract goldens. The stable markings
> are binding compatibility commitments for the v1 major line.

This document defines what consumers of the `hera` CLI — agents, scripts,
CI pipelines, wrappers — may rely on: invocation shape, output streams, exit
codes, error shapes, and per-command response fields. (`hera-agent-godot` is
the transitional alias for the same binary; the contract is identical.)

## What is public, what is internal

- **Public contract:** the CLI — its arguments, stdout/stderr, and exit codes.
- **Internal:** the localhost HTTP `/rpc` protocol between CLI and addon
  (`{"tool", "params"}` → `{"ok", "data", "error"}`), the heartbeat files under
  `~/.hera-agent-godot/instances/`, and port selection. CLI and addon ship and
  version **together**; the wire protocol may change between releases and is
  not a stable integration point. Build on the CLI, not on `/rpc`.

## Opt-in code, capture and save evidence

`--evidence` is experimental on script validation, screenshots, scene save and
resource set. See [linked evidence](COMMANDS.md#linked-evidence-opt-in) for fields,
limits and proof boundaries. Legacy successful outputs remain unchanged. Detailed
validation exits 1 for changed/unobservable source even when `exit_code` is 0.
Other detailed failures retain available JSON evidence on stdout and the error
on stderr, with exit 1. Parse errors remain exit 2. Explicit unavailability cannot
pass an evidence-enabled QA step. Scene save's `persistence: unknown` is a truthful
observation result, not full memory-to-disk verification. Screenshot operation
associations are caller-supplied labels, independent from operation receipts.
Captures without `--evidence` keep ordinary error strings; they do not use
`evidence_unavailable`.

Non-nested batch children receive the same evidence negotiation and guarded save
actions; `expected_sha256` implies evidence on scene save/resource set. Missing or
failed evidence children exit 1. A child claiming success without available
evidence is marked `ok: false` with `error: evidence_unavailable`; its data remains
on stdout. Ordinary batches retain their existing envelope and exit contract.

## Experimental operation receipts

`operation submit <session:unix_ms:nonce> --request JSON`, `operation status <id>`,
and `operation cancel <id>` print a receipt on stdout with exit 0 when obtained.
Inspect its `lifecycle`, `effect`, `verification`, `persistence`, `error_code`,
`cancellable`, and `evidence`; exit 0 alone is not mutation success. Default CLI
stdout omits `retention` and echoed `evidence.response`; `--verbose` prints them. Invalid CLI
input exits 2. Admission conflicts/expiry/capacity/unsupported actions exit 1
with `operation: <stable_code>: <detail>` on stderr and empty stdout.
Unknown or expired status is an explicit `outcome_unknown` receipt, not success
or proof of no effect. Failures that never published a runtime file or invoked
the setter/call are `rejected`/`not_applied` (for example `target_unavailable`),
not `outcome_unknown`. Cancellation additionally returns `cancelled`.

Supported actions are guarded editor node set and session-targeted runtime set/call.
`hera-godot status --capabilities` advertises this bounded surface as
`capabilities.operation_receipts`;
`game instances[].runtime_session_id` identifies the runtime incarnation.
IDs and session identities are strings; the decimal milliseconds component fixes
the deadline. Request/response evidence is bounded, in-process only, and disappears
on restart. The full lifecycle, error and retention semantics are defined in
[Commands](COMMANDS.md#operation-receipts-experimental). Legacy command output is
unchanged. Contract goldens cover status/cancel/conflict and command help.

## Stability tiers

Concurrent mutation or sensitive-read requests fail before dispatch with the
stable `editor_busy` code (exit 1, empty stdout, `<command>: editor_busy: ...`
on stderr). An admitted operation receipt instead records `lifecycle: rejected`,
`effect: not_applied`, and `error_code: editor_busy`, with receipt exit 0.
Safe status, operation status/cancel and buffered editor-log reads remain
available. A client timeout does not release a running owner's gate. See
[the gate policy](COMMANDS.md#editor-mutation-gate) for import readiness,
engine capability limits, batch ownership and the boundary of runtime completion.

| Tier | Meaning |
|------|---------|
| **stable** | Shape is frozen for `v1.0.0`. Documented fields keep their name and JSON type within a major version. New fields may be **added** at any time — consumers must ignore unknown fields. |
| **experimental** | Shape may change in any release. Changes are called out in release notes but carry no compatibility promise. |

Removing or renaming a stable field, or changing its JSON type, is a breaking
change and requires a major version bump plus a deprecation cycle.

## Versioning and deprecation policy

Hera follows semantic versioning from `v1.0.0` onward. The CLI and addon ship
as one versioned product and should be upgraded together.

- Patch releases contain compatible fixes and documentation corrections.
- Minor releases may add commands, flags, or response fields. Consumers of
  stable JSON must ignore unknown fields.
- A stable command, flag, field, JSON type, output stream, or exit-code meaning
  is deprecated in a minor release before it can be removed or incompatibly
  changed. Removal happens no earlier than the next major release.
- Experimental surfaces may change in a minor release. Their release notes must
  identify the change, but they do not carry the stable compatibility promise.
- The internal localhost HTTP protocol and heartbeat files remain internal and
  may change without a public deprecation cycle.

See [MIGRATING_TO_V1.md](MIGRATING_TO_V1.md) for the 0.x upgrade path.

## Invocation

```
hera-godot [--json|--ids] [--instance <pid>] [--timeout <ms>] [--engine <path>] <command> [args]
```

- Global flags come **before** the command. `--instance`, `--timeout`, and
  `--engine` accept both `--flag N` and `--flag=N`.
- `--timeout <ms>` bounds **each HTTP request** (default 5000 ms); it does not
  bound a whole command — `--wait` polls send many requests. A timed-out
  request is a runtime failure (exit `1`).
  For `script validate`, the same value separately bounds the native engine
  child process after context discovery, with up to one second to close pipes.
  For engine-backed commands (below) it bounds the engine child process itself.
- `--engine <path>` selects the engine executable for commands that do not need
  a live editor (also `HERA_GODOT_ENGINE`, then `GODOT_BIN`). It accepts the flag
  after those commands as well. A missing or non-file path fails on stderr with
  exit `1` before any process starts.
- Unknown commands and malformed flags/arguments never reach the editor; they
  fail fast with exit code `2`.
- With opt-in shared-token auth enabled ([SECURITY.md](./SECURITY.md)), a
  token mismatch is a runtime failure: `unauthorized: ...` on stderr, exit
  `1`. The CLI picks the token up automatically from
  `HERA_AGENT_GODOT_TOKEN` or `~/.hera-agent-godot/token`.

## Output streams

- **stdout** carries exactly one payload: the response `data` as **compact
  JSON on a single line** (default mode). Nothing else is written to stdout on
  the success path.
- **stderr** carries human-readable diagnostics on failure, one line per
  problem, in the shape `<command>: <message>` (e.g.
  `node: node not found: /nonexistent`). Error **message text is not part of
  the contract** — do not parse it; branch on the exit code instead. Messages
  may include actionable hints (e.g.
  ``game: no game is running; start one with `hera-godot run --current --wait` ``).
- `version` is the one exception: it prints a bare version string, not JSON.

### Output modes

| Mode | Behavior |
|------|----------|
| default | Compact JSON, one line. |
| `--json` | Same `data`, pretty-printed with 2-space indent. |
| `--ids` | For responses carrying a `nodes` or `controls` array (`scene tree`, `node find`, `game ui tree`): paths only, one per line. Anything else falls back to compact JSON. |

### JSON conventions

- Encoding is UTF-8. Key order is **not** part of the contract.
- Consumers must tolerate unknown fields (additive evolution is always
  allowed, even on stable commands).
- Project-relative paths use `res://`; absolute paths use forward slashes on
  all platforms (e.g. `C:/Users/...`).
- Node/resource **property values are Godot-stringified** (e.g. position
  `"(0.0, 0.0)"`, booleans `"true"`), not typed JSON — this applies to
  `node get`, `game node get`, `resource get`, and `game assert`
  actual/expected values.

## Exit codes

| Code | Meaning | Examples (verified) |
|------|---------|---------------------|
| `0` | Success. | `status`, `scene tree`, passing `game assert` |
| `1` | Runtime failure or failed check: no live editor, tool returned an error, mutation guard refused, or a verdict command reported not-OK. | `node get /nonexistent`, `game tree` with no game running, `game ui audit` with errors, `game qa diagnose` with issues |
| `2` | Usage error: unknown command, missing/invalid flag argument, invalid `--instance` pid or `--timeout` value, malformed scenario file arguments. | `hera-godot bogus`, `run --scene` (missing value), `--instance abc`, `--timeout abc` |

### Verdict commands

Commands whose job is to pass or fail mirror the verdict in the exit code, but
differ in where the detail goes:

- `game assert` — pass: verdict JSON on stdout
  (`{"prop","op","actual","expected"}`), exit `0`. Fail: the addon returns an
  error, so the CLI prints `game: assert failed: ...` to **stderr** and exits
  `1` (no stdout payload).
- `game qa --file` — summary JSON on **stdout**
  (`{"ok","steps","results"[,"requirements","requirements_covered","requirements_missing"]}`),
  exit `0`/`1` following `ok`. A requirement with no successful covering step
  makes `ok` false.
- `game qa diagnose` — report JSON on **stdout**
  (`{"ok","checks":[{"name","ok",...}],"issues":[...]}`), exit `0`/`1`
  following `ok`.
- `game ui audit` — audit JSON on **stdout**
  (`{"ok","strict","scope","controls","errors","warnings","findings","truncated"}`),
  exit `0`/`1` following `ok`. Errors always make `ok` false; warnings do so
  only with `--strict`. Operation failures remain stderr-only.
- `smoke` — progress/summary output, exit `0`/`1` by overall result.

## Per-command contract

Tier markings are the Phase 7 proposal. "Key fields" lists top-level `data`
fields; entries marked ✓ were captured live from a Godot 4.7 editor.
Stable-command stdout is additionally pinned byte-for-byte by the golden
contract tests (see [Contract tests](#contract-tests)).

### Core & discovery

| Command | Tier | Key fields |
|---------|------|-----------|
| `status` | stable | ✓ `pid`, `project_name`, `project_path`, `godot_version`, `scene`. Experimental additions always in the default CLI: `editor_session_id`, `game_feel_mode`, `game_feel_ui_mode`, and boolean `csharp_supported` (editor build capability, not SDK availability). `godot_commit` and `capabilities` (values `supported`, `unsupported`, `unverified`) are experimental detail on `status --capabilities`. See ARCHITECTURE §6 for capability scope. |
| `instances` | stable | ✓ `count`, `instances[]` of `{pid, port, project_path, godot_version, scene, ts}` with optional experimental `editor_session_id`; optional `stale[]` of the same shape plus `age_sec` when expired heartbeat files remain. Legacy heartbeats omit the session field. |
| `version` | stable | bare string (linker-injected; `dev` for source builds) |
| `run` / `stop` | stable | ✓ state shape `{playing, scene}` |

### Editor reads

| Command | Tier | Key fields |
|---------|------|-----------|
| `scene tree` | stable | ✓ `scene`, `count`, `truncated`, `nodes[]` of `{name, path, type}` |
| `scene list` | stable | ✓ `current`, `open[]` |
| `editor state` | stable | ✓ `current_scene`, `current_script{found, path}`, `main_scene`, `open_scenes[]`, `playing`, `playing_scene`, `project_name`, `project_path`, `selected[]` |
| `editor selected` | stable | selection list with scene-relative paths |
| `node find` | stable | ✓ `count`, `truncated`, `nodes[]` of `{name, path, type}` |
| `node get` | stable | ✓ `name`, `path`, `type`, `properties{}` (stringified values). Opt-in experimental `--prop P --snapshot` adds the complete typed `expected` object; ordinary reads are unchanged. |
| `signal list` | stable | ✓ `node`, `count`, `truncated`, `signals[]` of `{name, args[], connections[]}` (+ `external_connections` when editor-internal targets exist) |
| `screenshot diff` | experimental | ✓ `before`, `after`, `width`, `height`, `threshold`, `total_pixels`, `changed_pixels`, `changed_ratio`, `max_delta`, `identical`, and `changed_bounds{x,y,width,height}` when anything changed. Computed locally; `max_delta` is reported even when it is under the threshold |
| `theme get` / `theme set` | experimental | ✓ (`get`) `path`, `types[]`, `items{<type>{colors{}, constants{}, font_sizes{}}}`; (`set`) `path`, `type`, `applied{}`, `undoable:false`. Colour values are printed to 6 decimals so a value round-trips what the caller wrote rather than float32 noise |
| `project info` | stable | ✓ `name`, `path`, `current_scene`, `files{all, scene, script, resource, asset, shader, other}`, `godot{...}` |
| `project list-files` | stable | file list with compact type tags |
| `classdb info/methods/properties/signals/constants/enums/inherits` | stable | ✓ (`info`) `class`, `parent`, `can_instantiate`, `is_node`, `is_resource` |
| `resource get` | stable | class, name, editor-visible properties |
| `resource uid` | stable | ✓ `path`, `uid`, `uid_path`, `sidecar`, `sidecar_exists` |
| `resource list` | stable | resource entries with class + path |
| `output` | stable | ✓ `available`, `log_path`, `type`, `total`, `lines[]`. Additive `source` is `file` for the default log and `editor` for `--source editor`. |
| `diagnostics` | stable | ✓ `available`, `clean`, `file_logging_enabled`, `log_path`, `total_lines`, `error_count`, `errors[]`, `warning_count`, `warnings[]`. Additive `source` is `file` for the default log (so `clean` is not a claim about the editor Output panel) and `editor` for `--source editor`. `available` is false whenever the log cannot be read, and `clean` is false there too since cleanliness cannot be asserted without a readable log. `file_logging_enabled` is the *effective* value (`get_setting_with_override`), because file logging defaults to true on desktop through the `.pc` feature tag while the untagged default is false |
| `script current` / `script inspect` | experimental | compact script metadata; GDScript source response unchanged, C# loaded-assembly metadata as specified below |
| `script validate` | experimental | fresh on-disk `.gd` native engine check; JSON `{path,valid,exit_code,output,output_truncated,timed_out}`; optional `error` for launch failure; invalid/timeout exits 1, output capped at 64 KiB; leading `--timeout` also bounds the child process (default 5 s) |

### Editor log source (experimental)

The stable file-backed `output`/`diagnostics` defaults are unchanged. Their
unreadable response adds `reason:evidence_unavailable`. Opt-in `--source editor`
and `--since cursor` follow [the editor log evidence contract](COMMANDS.md#editor-log-evidence).
`--since` requires editor source. CLI negotiation checks the additive
`capabilities.editor_log_cursor` state before sending editor log requests.
Absent/unsupported/unverified capabilities yield JSON `available:false`,
`clean:false`, `reason:evidence_unavailable`, with no diagnostic counts.
These are evidence states (exit 0), not transport success claims about project
health; malformed flags exit 2 and RPC/transport failures exit 1 as before.

Editor output uses `entries[]` instead of the file source's text `lines[]`.
The documented session/cursor, severity/location, truncation, dropped-history,
and availability fields are experimental. Counts describe retained observations;
`complete:false` forbids a clean-history inference. Expired cursors carry
`restart_cursor` and never silently become a normal empty result. Timestamps
are not used to attribute events to actions.

### C# script metadata and build boundary

C# script inspection is experimental and reads Godot's loaded assembly, not a
C# source parser. Its additional fields are `language: "csharp"`,
`metadata_source: "assembly"`, `assembly_loaded` (boolean), `csharp_supported`
(boolean), and `build_warning`. `class_name` derives from the filename;
`base_type`/`extends` report the engine-native base when available. `functions`,
`signals`, and `exports` are empty when the assembly is unavailable. A loaded
assembly can be stale relative to the source; `assembly_loaded` does not prove
that the latest source was built. Standard Godot can inspect `.cs` and report
unavailable assembly metadata. `script current` reflects only Godot's focused
resource, not external IDE focus. GDScript response shapes remain unchanged.

C# create/open/attach require Godot .NET. Create uses a filename-matching partial
class, rejects a conflicting `--class-name`, and returns `build_required: true`
with a build warning. Attach rejects an unloaded C# class until build/reload;
success includes `language` and `build_warning` in `script_diagnostics`, whose
preload arrays remain GDScript-only diagnostics. Hera does not create `.csproj`/solution files or automatically build. `eval` remains a GDScript expression in
either project language. See [C# support](CSHARP_SUPPORT.md).

### Editor mutations

All enforce the single-editor guard unless `--instance` is passed; guard
refusal is exit `1`.

| Command | Tier | Notes |
|---------|------|-------|
| `node add` / `node set` / `node remove` | stable | undoable; `node add` may include an experimental `agent_hint` field when Game Feel Mode is on. `node set --expected JSON [--verify]` is an experimental opt-in guard, described below. |
| `node reparent` | experimental | undoable; uses `Node.reparent` and defaults to keeping the global transform; undo restores the original name after a sibling-name collision |
| `signal connect` / `signal disconnect` | stable | undoable, `CONNECT_PERSIST` |
| `scene open` / `scene save` | stable | |
| `eval` | stable | stringified expression `result`; experimental `result_value` / `result_value_omitted` from v1.2.0 |
| `batch` | stable | sequential results array; `--continue` keeps going past failures |
| `node instance` / `node set-resource` / `node attach-script` / `node detach-script` | experimental | attach-script responses include dependency diagnostics whose shape may evolve; C# requires .NET and a loaded class, and adds `language`/`build_warning` inside `script_diagnostics` |
| `scene create` / `scene save-as` / `scene reload` | experimental | persistent filesystem changes |
| `editor select` / `editor clear-selection` | experimental | editor-state mutation only |
| `script open` / `script create` | experimental | `.gd` or `.cs`; C# requires Godot .NET. Create selects by extension; optional `--lang` must match. C# creation adds `language`, `build_required`, `build_warning` |
| `project mkdir` / `project scan` / `project reimport` / `project set-main-scene` | experimental | persistent project changes |
| `resource set` / `resource create` / `resource resave` / `resource update-uids` / `resource export-mesh-library` | experimental | persistent filesystem changes |
| `screenshot` | stable (base) | ✓-adjacent base fields `path`, `width`, `height`; the `--analyze` metrics block is **experimental** |

#### Conditional node set

`node get --prop P --snapshot` adds `expected` with exactly six string fields:
`editor_session_id`, `scene`, `node_instance_id`, `prop`, `type`, `value`.
`node set --expected JSON [--verify]` requires all six; property must match
`--prop`. Malformed CLI input exits 2. Type limits and literal/Variant value
encoding are specified in [COMMANDS](COMMANDS.md#conditional-node-property-changes).

New CLI requests negotiate `capabilities.node_set_guard` on the selected client,
including guarded batch entries. An internal `set_guarded` action also ensures
an old addon replacing the connection after preflight rejects the mutation.
Ordinary unguarded requests and read shapes remain unchanged.

Conditional failures retain empty stdout, exit 1, and the normal `node:` stderr
label. The following error prefixes (before the next colon) are stable:
`session_mismatch`, `state_conflict`, `capability_unavailable`,
`verification_failed`, `verification_unavailable`. Text following the code is
not stable. The first three reject before setter/undo/save effects; the last
two report post-mutation failure and do not roll back. Success adds string
`editor_session_id`, `scene`, `node_instance_id`, and `verification`
(`passed` or `not_requested`) to the existing `path`, `prop`, `value` fields.
No persistence or full setter-side-effect isolation is implied. Batch results
retain their existing per-entry error and sequential execution contract.

### Runtime (game) surface

Uses the `HeraGameInspector` autoload; not undoable. The default targets the
editor-play process. `game --pid N ...` selects a fresh runtime heartbeat,
including an externally launched game. `--instance` and `--pid` identify the
editor and game respectively. Invalid, missing, or expired explicit PIDs fail
without fallback.

| Command | Tier | Key fields |
|---------|------|-----------|
| `game tree` | stable | `scene` + compact node list |
| `game node get/set/call` | stable | `get` mirrors `node get` (stringified values); `call` returns the stable stringified `result`; experimental `result_value` / `result_value_omitted` from v1.2.0 |
| `game assert` | stable | ✓ pass: `{prop, op, actual, expected}`; fail: stderr + exit 1 |
| `game instances` | experimental | `instances[]` with pid, scene, heartbeat age, `user_data_dir`, viewport sizes; optional `stale[]`; `shared_user_data` when live processes share `user://` |
| `game ui tree` | experimental | `Control` entries; fields selectable via `--fields` |
| `game ui audit` | experimental | `ok`, `strict`, `scope`, `controls`, `errors`, `warnings`, structured `findings[]`, `truncated` |
| `game click` / `game input` / `game input-log` | experimental | input injection + diagnostic log (v0.7 surface); click/input coordinates are live viewport pixels, not the project window setting. `game input` also injects joypad buttons and axes |
| `game clock` | experimental | `{paused, time_scale, process_frames, physics_frames}`; `--step` adds `stepped` (`process` or `physics`) |
| `game input sequence --file` | experimental | array of 1–128 `{frame,action,pressed}` events, ordered offsets 0–120; result `{kind,frames,events[],pid}` includes observed `physics_frame` per event; prevalidates actions, cancels on clock changes/deadline and releases held inputs |
| `game screenshot` | experimental | capture path and live PNG size; window/visible/project sizes; `size_matches_project` compares PNG pixels to the project viewport, not the visible rect. `--analyze` metrics evolve with QA guidance. Captures are not upscaled. |
| `game qa discover` | experimental | callable `qa_*` helpers or `Qa` followed by an uppercase letter (e.g. `QaReady`); exact case is preserved |
| `game qa diagnose` | experimental | ✓ `ok`, `checks[]` of `{name, ok, ...}`, `issues[]` |
| `game qa --file` | experimental | `ok`, `steps`, `results[]`, `requirements*` (verdict semantics above) |

Clock steps finish one selected node-callback phase before pausing. Input
sequences require an unpaused tree, time scale 1, and physics rate >=60 Hz;
their wall deadline is 2.5 s. Concurrent clock/input mutations are rejected.
`script validate` is CLI-owned; `script/validate-context` only resolves the
selected editor's paths. Native script loading may execute initializers;
validation is not a sandbox or a warning-free guarantee.

Explicitly targeted successful responses add experimental `game_pid` and
`game_scene` fields. `game qa diagnose` accepts multiple live processes when
its selected PID is present and reports `selected_pid` in the instance check.
The runtime inspector autoload is excluded from exported project settings and
dependencies, then restored to the editor after export.

### Guidance & content

| Command | Tier | Notes |
|---------|------|-------|
| `guidance ui` / `guidance game-feel` | experimental | ✓ (`ui`) `mode`, `setting`, `instruction`, `checklist[]`. Guidance **text is content, not contract** — it changes freely; only the envelope fields are candidates for stabilization. |
| `game_feel [topic]` | experimental | topic index / topic body; knowledge-base content evolves freely |
| `smoke` | experimental | check-by-check progress; exit code is the contract, output shape is not |

## Engine-backed commands (experimental, no live editor)

New in the unreleased v1.3 line. These run the engine binary as a bounded child
process and are **experimental**: names and fields may change in a minor
release until they are promoted.

| Command | Tier | Notes |
|---------|------|-------|
| `engine info` / `engine doctor` | experimental | Identity answers carry `engine{path,source,bytes}`, `version{string,status,mono,commit}`, and `help_available`. `editor_build` and `help_flags{}` appear **only when the help probe completed** (exit 0, non-empty, untruncated, not timed out); otherwise `help_probe_error` explains why and no build claim is made. `engine doctor` adds `checks[]{name,ok,detail}` and `warnings[]`, and exits `0` for a healthy engine even when individual checks are `false`. |
| `script run` | experimental | `exit_code`, `timed_out`, `duration_ms`, `output` (+`output_truncated`), `arguments[]`, `path`, `project_path`, `engine{path,source,version}`, `source: "engine_process"`. Exit `1` when the engine exits non-zero or the bound is hit; a pre-flight refusal is `not_a_harness` (only bases known to be incompatible are refused). |
| `project import` | experimental | `exit_code`, `duration_ms`, `output`, `cache{}`, `script_uid_files`, `engine{}`. Engine progress text is localized and not part of the contract. |
| `project export` | experimental | Success: `project_path`, `kind`, `preset`, `output_path`, `output_exists`, `output_is_dir` (target type, since Godot takes a file for ordinary binary/pack/patch exports and some plugin-driven targets write a directory), `output_bytes` (regular files), `available_presets[]`, `engine{path,source,version}` (+`export_templates_dir`/`export_template_files` when the usual directory was found), optional `warnings[]`. The raw log is omitted by default: `output_omitted:true` plus a reason; `--verbose` restores `output`/`output_truncated` at the caller's risk. Pre-flight failure: `ok:false`, `error` in `{preset_not_found, export_presets_missing, export_presets_unreadable, output_directory_missing, output_directory_unreadable, pack_extension, editor_build_required}`, with a stderr diagnostic; exit `1`. A clean exit whose target is absent fails with `no_artifact_produced`; Hera checks existence, not platform-specific completeness. Template lookup is advisory and never blocks an export. |
| `capture movie` | experimental | `output_path`, `output_exists`, `output_bytes`, `frames_requested`, `frames_written`/`frames_present` (PNG sequences), `frame_size` (PNG sequences), `fps`, `engine{}`. PNG frames are identified by change against a pre-run snapshot, so `frames_written` counts frames whose size or modification time changed and `frames_unchanged` counts the rest. The post-run scan always decides first: a sequence that cannot be listed fails with `sequence_scan_failed` and one with no frame at all fails with `no_frame_produced`, whatever the pre-run state was. With output present, an all-unchanged sequence is `frames_verification: "inconclusive"` (a same-size rewrite inside one timestamp interval is indistinguishable from no work, and a concurrent writer's change is indistinguishable from this run's), and a pre-run observation that could not be taken reports the same inconclusive field with its reason but withholds `frames_written`/`frames_unchanged`. Containers are compared as a single file: missing after a clean exit fails with `no_artifact_produced`, unchanged since before the run is `output_verification: "inconclusive"`, and a container whose stat failed before the run is simply "not seen before", so it is not reported as inconclusive. A display driver is required; `--headless` is never passed. `res://` outputs are resolved into the project before inspection. |
| `perf sample` | experimental | `benchmark{entries,total_seconds,top[]{name,seconds}}`, `bounded_by` in `{engine_exit, timeout}`, `headless`. Godot 4.7.2 ignores `--quit-after` under benchmark flags, so `timeout` is a normal outcome (default bound 30 s) and exit `0` means the JSON was read, not that the examined game run succeeded. |
| `docs dump` | experimental | `mode` in `{xml, json, scripts}`, `output_dir`, `files`+`bytes` (XML modes) or `output_file`+`output_bytes` (json), `engine{}`. A destination holding `*.xml` is refused (`output_not_empty`) and an existing `extension_api.json` is refused (`output_exists`) unless `--force`; JSON mode also fails with `json_artifact_missing` when no regular non-empty file was produced, `json_artifact_stale` when the file was not replaced, and XML mode with `docs_output_empty` when nothing was generated, `docs_scan_failed` when the destination could not be scanned, and `xml_verification: "inconclusive"` when no XML artifact changed size or modification time (a same-size rewrite inside one timestamp interval is indistinguishable from no work) or when the pre-run state could not be observed. |
| `classdb dump` | experimental | `output_dir`, `files[]{name,path,bytes,ok,error}`, `file_count`, `failed_copies`, `engine{}`. Exit `1` with `class_db_json_missing` (engine produced nothing), `class_db_copy_failed` (publication failed) or `class_db_publish_failed` (walk failed). |

Shared facts for all engine-backed commands: engine resolution is
`--engine` → `HERA_GODOT_ENGINE` → `GODOT_BIN`; `--timeout` bounds one shared
context per command; an already-expired context starts nothing; output text is
captured to 64 KiB and never parsed for status; a timeout terminates work
through an owned identity (Windows Job Object + process handle, Unix the started
process only, so Unix descendants are not signalled) and reports
`termination_error` when cleanup was incomplete. The wait for a process that
termination could not stop is bounded and reported (`abandoned`, `engine_abandoned`,
which means the wait for the process — including its pipe drain — was not observed
to finish, not that the process is known to still run) instead of hanging the
caller; a healthy run is never cut short by that bound. A
nonzero engine exit (`engine_exit_nonzero`) or a `WaitDelay` drain
(`output_incomplete`) fails the command **except** for `perf sample`, whose
documented exception is that a parsed benchmark JSON decides success.
`exit_code: -1` means no numeric exit status is available — a pre-launch refusal,
an abandoned process, or a process reaped after a signal on Unix — and is never a
successful exit. Identity commands and
pre-flight refusals are the remaining exception to the `engine` block shape: they
carry either no `engine` block or one without a `version`.

## Debug sessions (experimental)

New in the unreleased v1.3 line. `debug` is a DAP client for a running editor
debug server; it does not start or stop the editor or the game.

| Command | Tier | Notes |
|---------|------|-------|
| `debug attach` | experimental | `session_id`, `host`, `port`, `project`, `breakpoints[]`, `breakpoint_results[]{path,line,verified,unavailable,checksums_source,checksums_available}` (with `unverified_breakpoints[]` when the adapter could not bind one), `limits{host_override,port_override,identity_check}`, and either `stopped{frame,frame_count,reason,observed,resume_confirmed,attribution}` or `running:true`. `observed` is `already_paused` (whose `attribution` is `not_applicable`, since Hera asked for no resume) or `after_arming`. Exit `1` when the endpoint is unreachable, `initialize`/`attach` is rejected, observation fails, or the session record cannot be persisted (`session_store_failed`). |
| `debug session` | experimental | The recorded attach state: `session_id`, `host`, `port`, `project`, `breakpoints[]`, `attached_at`. Unknown or missing sessions fail with exit `1`. |
| `debug stack` / `debug variables` / `debug evaluate` | experimental | `stack`: `frames[]{id,name,line,source{name,path,md5,sha256,sha256_available,checksums_source}}` or `frames:[]`+`running:true`; it never resumes. `variables`: `variables[]{scope,name,value}` (capped at 50, with `variables_truncated`) — a running debuggee is resumed to its next stop first, because the adapter arms breakpoints on resume; one retry is allowed only when the adapter rejected a `variables`/`scopes` read and the first attempt's paused state was certain (no resume, or a confirmed resume), never after a transport failure or an unconfirmed resume. `evaluate`: `expression`+`result` with the same resume behavior, and expressions starting with `-` are passed after `--`. A command that cannot observe a usable stop fails with `not_paused` or `dap_request_failed` (exit 1) instead of claiming `running`. |
| `debug continue \| next \| step-in \| step-out` | experimental | Same `stopped{frame,frame_count,reason,resume_confirmed,attribution}` shape as attach, or `running:true` with `resume_confirmed:true`. Before acting, the connection is drained to a quiet window (idle window derived from `--wait-ms`, absolute budget: steady output fails the command instead of hanging) and stops observed earlier are discarded. That quiet window **does not prove** attribution, so a `stopped` event is only accepted after the adapter's own `continued`/`resumed` event and every stop carries `attribution: "unproven: …"`; a resume the adapter never confirms is a failure, not a success. |
| `debug detach` | experimental | `detached:true` and the session record is removed; the editor and game keep running. |

Session records live under `~/.hera-agent-godot/debug/`, are plain JSON, and are
not a stable integration point: build on the CLI, not on the files. Frame ids in
DAP responses are only valid for the current stop.

Known limits: a response whose `command` field is empty is accepted; the
`initialized` event is consumed when present but not awaited before breakpoint
configuration; concurrent `debug attach`/`debug detach` share one `latest`
pointer without a single-writer lock, so a detach can clear a newer session's
pointer and later default-session commands fail until the next attach; endpoint
continuity cannot be proven; and a stop's `attribution` is reported as unproven
because the adapter provides no causal request boundary, except an
`already_paused` observation (which Hera did not cause) that reports
`not_applicable`.

## Known gaps (tracked in ROADMAP Phase 7)

- Detailed field-level schemas (types, optionality) for every subcommand are
  still to be pinned; this draft freezes names of the fields listed above.

## Contract tests

`cmd/contract_golden_test.go` pins this contract in CI: it runs the real CLI
end-to-end (argv → discovery → HTTP → stdout/exit code) against a mock editor
serving fixture responses, and byte-compares stdout with golden files under
`cmd/testdata/contract/`. Stable read commands use responses captured from a
live Godot 4.7 editor; exit-code and stderr-shape semantics above are asserted
directly. After an **intentional** contract change, regenerate with
`go test ./cmd -run TestContract -update` and list the change in release notes.

See [COMMANDS.md](./COMMANDS.md) for flags and semantics,
[ARCHITECTURE.md](./ARCHITECTURE.md) for the request lifecycle, and
[ROADMAP.md](./ROADMAP.md) for the standardization arc.

## QA validation and mutation corrections (2026-09-08)

Scenario diagnostics require readable evidence and nonnegative integral counts. `available:false`, a non-boolean availability field, or absent/invalid counts cannot satisfy requirement coverage. A missing `available` field remains compatible with legacy count-bearing responses. Omitted `max_warnings` has no warning limit; explicit zero requires zero warnings. Scenario tool names, required node fields, assertion operators, run actions and numeric limits are validated before execution; remaining tool-specific parameters are validated by their owning tool.

`project set-main-scene` persists through the editor and retains `main_scene` and `project_path` output fields. Default `run` uses that editor setting immediately. Resource property validation rejects incompatible native JSON types and fractional/nonfinite/out-of-range values for integer properties; integral JSON numbers remain accepted. A rejected mixed resource/theme property batch applies no setters; this does not promise rollback after a disk save error. Node removal and reparent undo preserve original subtree ownership, and resolved root aliases are rejected. Runtime UI rectangles and click centers use viewport coordinates including canvas transforms.

Scenario `run` actions `stop` and `state` do not wait for play: `stop` with `wait:true` waits for stopped editor state and runtime heartbeats, while `state` returns its snapshot directly.

## Structured call and eval results (experimental)

From v1.2.0, successful `game node call` and `eval` responses add exactly one
of `result_value` or `result_value_omitted`. The existing `result` string stays
stable throughout v1.x; `type` still names the root Godot Variant type, not
nested element types. `path`, `method`, and `undoable` keep their meanings.

Read `result_value` directly after parsing stdout once. Test field presence:
`null` and `false` are valid values. A method that returns a JSON string still
has a string `result_value`; Hera does not parse it automatically. Return a
Dictionary or Array from QA helpers when structured data is intended.

| Godot value | Structured JSON value |
|-------------|-----------------------|
| Nil, bool, String, finite float | null, boolean, unchanged string, number |
| int in −9007199254740991…9007199254740991 | number |
| Other int | exact decimal string, avoiding float precision loss in editor/CLI JSON parsing |
| NaN, positive/negative infinity | variant-text strings `nan`, `inf`, `inf_neg` |
| StringName, NodePath | plain string |
| Array and packed arrays | recursively encoded JSON array |
| Dictionary | recursively encoded JSON object; String/StringName keys use plain text, other keys use `var_to_str` |
| Valid Object / freed Object | `str(object)` string / null; object properties are not dumped as returned values |
| Other leaves (vectors, colors, transforms, RID, Callable, Signal) | `var_to_str` string |

Limits apply to the additional value: root depth 0, maximum depth 32,
10,000 container entries across traversal, and 65,536 UTF-8 bytes in its
compact `JSON.stringify` representation. A dictionary pair counts as an entry;
container keys are also checked before variant-text formatting. If a limit is
exceeded, the entire value is omitted, with `result_value_omitted` set to
`depth_limit`, `item_limit`, or `size_limit`. Distinct keys that normalize to
the same JSON key omit the entire value with `key_collision`. Partial values
are never returned. These limits do not cap the legacy `result` string.

No capability is advertised: response fields are the signal. An older CLI
passes the new fields through unchanged. A new CLI does not synthesize them
for an older addon. For runtime calls, a new editor relaying an older runtime
still returns only legacy fields; an older editor can relay new runtime
fields. `eval` depends only on the editor addon. If both new fields are absent,
the component producing that response is older. Do not infer runtime support
from the editor's version or capabilities.

Batch eval children and QA call-step data retain the new fields. Operation
receipts retain their existing 16 KiB response-evidence limit: the additional
value can make `result_byte_limit` more likely, without changing lifecycle or
effect. Use `--verbose` to inspect retained response evidence.
