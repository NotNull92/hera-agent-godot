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

## Stability tiers

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
hera [--json|--ids] [--instance <pid>] [--timeout <ms>] <command> [args]
```

- Global flags come **before** the command. `--instance` and `--timeout`
  accept both `--flag N` and `--flag=N`.
- `--timeout <ms>` bounds **each HTTP request** (default 5000 ms); it does not
  bound a whole command — `--wait` polls send many requests. A timed-out
  request is a runtime failure (exit `1`).
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
  ``game: no game is running; start one with `hera run --current --wait` ``).
- `version` is the one exception: it prints a bare version string, not JSON.

### Output modes

| Mode | Behavior |
|------|----------|
| default | Compact JSON, one line. |
| `--json` | Same `data`, pretty-printed with 2-space indent. |
| `--ids` | For responses carrying a `nodes` array (`scene tree`, `node find`): node paths only, one per line. Anything else falls back to compact JSON. |

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
| `1` | Runtime failure or failed check: no live editor, tool returned an error, mutation guard refused, or a verdict command reported not-OK. | `node get /nonexistent`, `game tree` with no game running, `game qa diagnose` with issues |
| `2` | Usage error: unknown command, missing/invalid flag argument, invalid `--instance` pid or `--timeout` value, malformed scenario file arguments. | `hera bogus`, `run --scene` (missing value), `--instance abc`, `--timeout abc` |

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
- `smoke` — progress/summary output, exit `0`/`1` by overall result.

## Per-command contract

Tier markings are the Phase 7 proposal. "Key fields" lists top-level `data`
fields; entries marked ✓ were captured live from a Godot 4.7 editor.
Stable-command stdout is additionally pinned byte-for-byte by the golden
contract tests (see [Contract tests](#contract-tests)).

### Core & discovery

| Command | Tier | Key fields |
|---------|------|-----------|
| `status` | stable | ✓ `pid`, `project_name`, `project_path`, `godot_version`, `scene`. (`game_feel_mode`, `game_feel_ui_mode` are experimental fields inside a stable response.) |
| `instances` | stable | ✓ `count`, `instances[]` of `{pid, port, project_path, godot_version, scene, ts}` |
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
| `node get` | stable | ✓ `name`, `path`, `type`, `properties{}` (stringified values) |
| `signal list` | stable | ✓ `node`, `count`, `truncated`, `signals[]` of `{name, args[], connections[]}` (+ `external_connections` when editor-internal targets exist) |
| `screenshot diff` | experimental | ✓ `before`, `after`, `width`, `height`, `threshold`, `total_pixels`, `changed_pixels`, `changed_ratio`, `max_delta`, `identical`, and `changed_bounds{x,y,width,height}` when anything changed. Computed locally; `max_delta` is reported even when it is under the threshold |
| `theme get` / `theme set` | experimental | ✓ (`get`) `path`, `types[]`, `items{<type>{colors{}, constants{}, font_sizes{}}}`; (`set`) `path`, `type`, `applied{}`, `undoable:false`. Colour values are printed to 6 decimals so a value round-trips what the caller wrote rather than float32 noise |
| `project info` | stable | ✓ `name`, `path`, `current_scene`, `files{all, scene, script, resource, asset, shader, other}`, `godot{...}` |
| `project list-files` | stable | file list with compact type tags |
| `classdb info/methods/properties/signals/constants/enums/inherits` | stable | ✓ (`info`) `class`, `parent`, `can_instantiate`, `is_node`, `is_resource` |
| `resource get` | stable | class, name, editor-visible properties |
| `resource uid` | stable | ✓ `path`, `uid`, `uid_path`, `sidecar`, `sidecar_exists` |
| `resource list` | stable | resource entries with class + path |
| `output` | stable | ✓ `available`, `log_path`, `type`, `total`, `lines[]` |
| `diagnostics` | stable | ✓ `available`, `clean`, `file_logging_enabled`, `log_path`, `total_lines`, `error_count`, `errors[]`, `warning_count`, `warnings[]`. `available` is false whenever the log cannot be read, and `clean` is false there too since cleanliness cannot be asserted without a readable log. `file_logging_enabled` is the *effective* value (`get_setting_with_override`), because file logging defaults to true on desktop through the `.pc` feature tag while the untagged default is false |
| `script current` / `script inspect` | experimental | compact script metadata (class name, extends, functions, signals, exports, line count) |

### Editor mutations

All enforce the single-editor guard unless `--instance` is passed; guard
refusal is exit `1`.

| Command | Tier | Notes |
|---------|------|-------|
| `node add` / `node set` / `node remove` | stable | undoable; `node add` may include an experimental `agent_hint` field when Game Feel Mode is on |
| `signal connect` / `signal disconnect` | stable | undoable, `CONNECT_PERSIST` |
| `scene open` / `scene save` | stable | |
| `eval` | stable | stringified expression result |
| `batch` | stable | sequential results array; `--continue` keeps going past failures |
| `node instance` / `node set-resource` / `node attach-script` / `node detach-script` | experimental | attach-script responses include dependency diagnostics whose shape may evolve |
| `scene create` / `scene save-as` / `scene reload` | experimental | persistent filesystem changes |
| `editor select` / `editor clear-selection` | experimental | editor-state mutation only |
| `script open` / `script create` | experimental | |
| `project mkdir` / `project scan` / `project reimport` / `project set-main-scene` | experimental | persistent project changes |
| `resource set` / `resource create` / `resource resave` / `resource update-uids` / `resource export-mesh-library` | experimental | persistent filesystem changes |
| `screenshot` | stable (base) | ✓-adjacent base fields `path`, `width`, `height`; the `--analyze` metrics block is **experimental** |

### Runtime (game) surface

Requires a play session plus the `HeraGameInspector` autoload; not undoable.

| Command | Tier | Key fields |
|---------|------|-----------|
| `game tree` | stable | `scene` + compact node list |
| `game node get/set/call` | stable | `get` mirrors `node get` (stringified values); `call` returns a stringified result |
| `game assert` | stable | ✓ pass: `{prop, op, actual, expected}`; fail: stderr + exit 1 |
| `game instances` | experimental | `instances[]` with pid, scene, heartbeat age |
| `game ui tree` | experimental | `Control` entries; fields selectable via `--fields` |
| `game click` / `game input` / `game input-log` | experimental | input injection + diagnostic log (v0.7 surface) |
| `game screenshot` | experimental | capture path; `--analyze` metrics evolve with QA guidance |
| `game qa discover` | experimental | callable `qa_*` helpers |
| `game qa diagnose` | experimental | ✓ `ok`, `checks[]` of `{name, ok, ...}`, `issues[]` |
| `game qa --file` | experimental | `ok`, `steps`, `results[]`, `requirements*` (verdict semantics above) |

### Guidance & content

| Command | Tier | Notes |
|---------|------|-------|
| `guidance ui` / `guidance game-feel` | experimental | ✓ (`ui`) `mode`, `setting`, `instruction`, `checklist[]`. Guidance **text is content, not contract** — it changes freely; only the envelope fields are candidates for stabilization. |
| `game_feel [topic]` | experimental | topic index / topic body; knowledge-base content evolves freely |
| `smoke` | experimental | check-by-check progress; exit code is the contract, output shape is not |

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
