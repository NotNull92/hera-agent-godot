# Migrating to the 2.1 CLI response shape

These breaking omissions are implemented for the planned 2.1 release. This is
an explicit exception to the stable-field major-version policy; it does not
announce a release. Upgrade consumers before deploying this CLI.

Compact output, `--json` pretty output, and each `batch` child use the same
shape. `--ids` remains paths only. Full addon RPC responses and internal CLI
capability/session reads are unchanged; an older CLI can still use the addon.
There is no new full-output flag. User property values, structured call results,
and evidence objects are not recursively trimmed.

| Field | When now omitted | How to get it |
|-------|------------------|---------------|
| `truncated` | When false | Treat absence as false; true is still printed. |
| `count` | When equal to the length of an adjacent response array | Use the array length; a differing total is retained. |
| Node `name` | When equal to the final segment of `path` | Use that segment; root `path: "."` retains its name. Applies to tree/find/UI entries and node gets. |
| Structured failure `message` | CLI engine/debug structured failures | Read `error` and `detail`; the same diagnostic still goes to stderr. Exit codes are unchanged. |
| File-log `log_path` | When `available: true` | Use the full `output`/`diagnostics` RPC data if needed. Unavailable/unreadable logs retain the path. |
| Log `source` | When `"file"` | Treat absence as `"file"`. Editor-source metadata is retained. |
| `game input` request echoes | When a returned input field equals its request value | Retain your request. `pid`, `log_count`, normalized values that differ, and other computed fields remain. |
| Status `editor_session_id` | Both ordinary and capability output | Guard flows use the complete `expected` from `node get --prop P --snapshot`; operation IDs can use the selected live `instances[].editor_session_id`; raw status RPC also retains it. |
| Status `project_path` | Both ordinary and capability output | `project info` returns `path`; `editor state` returns `project_path`. |

`status --capabilities` retains its capability block. Never manufacture a guard
snapshot from a printed status response. Retain the complete `expected` object.

Two optional combined commands accompany this change: `node add` and
`node instance` accept repeatable `--set prop=value` and return re-read `values`;
`game assert ... --within-ms N` retries each runtime frame and returns
`waited_ms` (plus `timed_out: true` on timeout). QA assertions accept `within_ms`.
Value flags accept `--flag=` exactly as `--flag ""`, subject to each flag's
normal validation. See [COMMANDS.md](COMMANDS.md) for bounds and failure behavior.
