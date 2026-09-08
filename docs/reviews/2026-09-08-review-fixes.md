# Review correction evidence — 2026-09-08

## Scope and identity

Approved implementation of the findings in [the original review](2026-09-07-code-review.md). No MCP server, dependencies, release, user-project mutation, or installation changes.

Base commit: `d6543c421e9b830301a8e0329c7ba56eaa2a1fd8`.
This is an uncommitted working-tree review; the base SHA alone is not coverage of the changes.
Source snapshot SHA256 (sorted changed/untracked non-document files, path + NUL + bytes + NUL): `81fa9cf375ea75f4fcf6c487b08768cf5862d47083243c14987ddda8ca873410`.

## Changes and observations

- Node removal snapshots every original subtree owner and rejects resolved roots. The regression exercises undo/redo, packing, descendant owners, and root aliases. Live CLI `node remove Branch/..` failed and retained the three-node tree.
- Resource/theme property updates validate and coerce all requested values before setters. Live mixed valid/invalid resource modification failed and preserved `resource_name=original`.
- HTTP response writes remain tracked and send at most 64 KiB per connection per poll, with an absolute five-second write deadline. A nonreading 32 MiB socket regression retained editor ticks, stalled its offset, then expired. Normal UTF-8 replies, delayed async responses, and late disconnected replies are covered. The old Windows socket did not reproduce a literal blocking hang; old response tracking failed the regression.
- Missing/empty tokens preserve intentional opt-out. A configured unreadable token refuses HTTP and heartbeat startup; an exclusive Windows file lock reproduced the failure with zero heartbeats.
- Runtime IPC uses temporary-file publication and only consumes parseable dictionaries. Partial-request and self-free call tests distinguish the old behavior. Native `free` calls return the original target path safely.
- Shared viewport geometry includes canvas transforms. An isolated graphical runtime at 640×480 clicked a CanvasLayer button at (250,125); its counter became 1, UI audit returned zero issues, and a runtime screenshot reported matching project size. The headless runtime was actually 64×64, so its out-of-bounds click rejection was correct.
- QA diagnostics reject unreadable and malformed evidence; thresholds distinguish omitted warnings from explicit zero. The scenario preflight checks known tool names, required node fields, run actions, assertion operators, and numeric limits before requests; individual tools still own remaining parameter validation.
- Main-scene setting uses the existing guarded editor RPC. Removed the Go text writer and default-run disk reread. An isolated Old→New setting changed disk and in-memory state, then bare `run --wait` launched `/root/New` without restart. Invalid traversal preserved settings. The engine serialization regression includes a project file without a final newline.
- Reused project path safety, variant text hints, input name parsing, and viewport geometry. Coercion policy remains at each owning boundary.
- CI now runs all standalone behavioral tests on Godot 4.7 in separate project/user directories.

## Verification

- Go: `go build ./...`, `go vet ./...`, `go test -shuffle=on ./...` passed. `gofmt -l cmd/*.go` was empty.
- Contract goldens regenerated with `go test ./cmd -run TestContract -update`; no golden content changed.
- Godot 4.7: all 16 standalone regressions passed; all 52 addon GDScript files passed check-only. Raw logs: `C:/Users/PC/AppData/Local/Temp/hera-final-regressions.log` and `hera-final-static.log`.
- Isolated live editor/runtime checks above used Godot 4.7; `script validate` and `smoke --skip-game` also passed.
- Final independent review: all five lanes passed after the Q1 correction; initial findings and fresh delta verdicts are retained below.

## Limits

Disk-save failure rollback is not implemented; upfront validation prevents invalid batches, not all filesystem failures. Windows can refuse Godot FileAccess reads of a live logger with open error 12 even when a shell can read the path; such diagnostics now return unavailable instead of a false clean result. Runtime stderr was checked separately. Headless editor shutdown RID/ObjectDB messages remain visible and are not script/test failures. Local race tests remain unavailable under the documented antivirus restriction; CI retains race coverage. Local Godot 4.2 execution was not repeated for these edits; the static CI matrix remains 4.2/4.7.

## Initial independent review ledger

All lanes bind base commit `d6543c421e9b830301a8e0329c7ba56eaa2a1fd8` plus source snapshot `81fa9cf375ea75f4fcf6c487b08768cf5862d47083243c14987ddda8ca873410`.

| Lane | Verdict | Source |
|---|---|---|
| Goal | FAIL (Q1) | [report](2026-09-08-final-goal.md) |
| Quality | FAIL (Q1) | [report](2026-09-08-final-quality.md) |
| Security | PASS | [report](2026-09-08-final-security.md) |
| Context | PASS | [report](2026-09-08-final-context.md) |
| QA | PASS within executed scenarios | [report](2026-09-08-final-qa.md) |

Q1 exposed an additional raw JSON type bypass in resource values: a Dictionary assigned to Gradient.offsets cleared the packed array and allowed earlier properties to change. A focused regression reproduced this and fractional integer truncation before the correction. The correction validates non-string types before any setter, permits integral in-range JSON numbers for integer properties, and rejects incompatible native values. The updated resource regression passes, including the range boundary. Fresh delta review is required before aggregate approval.

## Final delta identity and review ledger

Final base commit: `d6543c421e9b830301a8e0329c7ba56eaa2a1fd8`.
Final source SHA256: `23b9ba74f80ea0571e743015e46d4a1cfbff070404d85eae42b50a8dbae3e3ad`.
Only the resource codec's non-string validation and its existing resource regression changed after the initial snapshot. Initial unchanged-file coverage plus fresh delta coverage applies to this working tree; neither fingerprint is a commit.

| Lane | Delta verdict | Source |
|---|---|---|
| Quality | PASS | [report](2026-09-08-delta-quality.md) |
| Security | PASS | [report](2026-09-08-delta-security.md) |
| Goal | PASS | [report](2026-09-08-delta-goal.md) |
| Context | PASS | [report](2026-09-08-delta-context.md) |
| QA | PASS | [report](2026-09-08-delta-qa.md) |

The live CLI delta check ran against a restarted isolated editor with the final addon. The failed batch step reported `property expects PackedFloat32Array, got Dictionary`; a subsequent resource read retained `resource_name=original` and `offsets=[0.0, 1.0]`. The outer batch exit remained zero per its existing response-envelope behavior; the nested step was `ok:false`. Raw responses remain at `C:/Users/PC/AppData/Local/Temp/hera-fixes-live.fyDjpB/raw-probe-failure.json` and `raw-probe-after.json`.

Main-scene fixture evidence is retained at `C:/Users/PC/AppData/Local/Temp/hera-main-scene-809347c6167b4ef881d42449e60c61c1/` (final project.godot, read_setting.gd, editor-new.log). The CLI sequence itself was captured in execution tool responses, not a separate transcript file. The engine persistence regression log is `C:/Users/PC/AppData/Local/Temp/hera-main-scene-check-a9019d3d94ff416f95f3059442e624a2/check.log`.

## Completion

Overall verdict: **PASS** for the approved review corrections and bounded refactoring. Initial findings remain recorded rather than overwritten. All five final lanes passed using initial unchanged-code coverage plus fresh correction coverage at the final source fingerprint.

Final Go build/vet and uncached shuffled full tests passed. The 16-test Godot suite and all 52 addon parse checks passed before the final two-file correction; the corrected resource test and codec parse/execution were then rerun successfully, including independent numeric boundary checks and a restarted-editor live CLI batch. Contract goldens were regenerated after documentation sync with no golden diff.

All coordinator-owned editors/runtimes were stopped; implementation and QA workers reported their owned processes stopped as well. The user's project/editor was not changed. No source-tree smoke fixtures, version bumps, commits, installations, or external publications were created. Private temporary fixtures and logs referenced here remain as review evidence.
