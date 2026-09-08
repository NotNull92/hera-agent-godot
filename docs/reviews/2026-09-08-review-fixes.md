# Review correction evidence — 2026-09-08

Shipped in `v1.1.0`. Follow-up: [push-corrections.md](2026-09-08-push-corrections.md).

Base commit: `d6543c421e9b830301a8e0329c7ba56eaa2a1fd8`.

## What landed

- Node removal snapshots every original subtree owner and rejects resolved roots.
- Resource/theme batches validate all requested values before setters.
- HTTP response writes are bounded (64 KiB per poll, 5 s deadline).
- A configured unreadable auth token refuses HTTP and heartbeat startup.
- Runtime request files are published atomically; native `free` calls keep the original path.
- Shared viewport geometry includes canvas transforms.
- QA diagnostics reject unreadable evidence; scenarios preflight before requests.
- Main-scene setting uses the editor RPC only (no duplicate Go project-file writer).
- CI runs standalone Godot 4.7 behavioral tests in isolated project/user directories.

## Limits

Disk-save failure rollback is not implemented. Windows can refuse Godot
`FileAccess` reads of a live logger (error 12) even when a shell can read the
path; diagnostics then return unavailable instead of a false clean. Headless
shutdown RID/ObjectDB messages remain engine teardown. Local race tests stay
unavailable under the antivirus restriction; CI keeps race coverage.

## Verdict

PASS for the approved corrections. Go build/vet/shuffled tests, 16 Godot
regressions, and addon `--check-only` passed. Private temp logs are not in
this tree.
