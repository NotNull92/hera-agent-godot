# Editor reliability implementation evidence — 2026-09-14

The bounded M0–M5 implementation is complete on the development branch; this is
not a release or publication. Final source commit before this documentation:
`3e38db70dba9985dbbcba79cdd13b7cd64b485bc`.

## Implemented boundaries

| Milestone | Observed contract |
|---|---|
| M0 | Engine capability states and random editor session identity shared by status and heartbeat. |
| M1 | Opt-in bounded editor logs with session cursors, unavailable states and dropped-history metadata. |
| M2 | Typed property snapshots and guarded mutation; session/scene/node identity conflicts fail before setting. |
| M3 | Explicit, bounded, in-process operation receipts; duplicate suppression, queued cancellation and unknown outcomes. |
| M4 | Import/mutation ownership across async work, batches, disconnects and retirement. |
| M5 | Disk code provenance, capture session/time/size evidence, separate memory/save/file observations and evidence-enabled QA. |

M0–M4 implementation and review evidence is recorded in the earlier milestone
history and engine baseline report. M5 used those contracts without adding another
transport, QA engine, dependency, automatic reload, upscale or InputMap action.

## M5 changes and regressions

- Native `--check-only` can finish successfully after its script changed on
  disk. The red fixture rewrote itself in a static initializer and previously
  returned `valid: true`, exit 0. Detailed validation now compares pre/post hashes
  and fails changed/unavailable evidence while retaining the actual child exit.
- A success-shaped screenshot result with `evidence.available: false` previously
  counted toward QA coverage. Evidence-enabled steps now fail and retain the
  returned evidence. Existing runtime assertion failures still fail the scenario
  even after a successful screenshot covers the same requirement.
- Windows file locks and readonly attributes caused native safe-save failure
  despite a successful engine save return. Fresh selected-property verification
  detected the old disk value and reported `effect: applied`, `persistence: failed`.
  The in-memory resource was explicitly retained during this observation.
- A synchronous editor-plugin disable inside an evaluated tool method could
  leave `_process` dispatching the next drained item on a freed plugin. A local
  queue reference now stops the loop on retirement. Both a standalone drained
  queue regression and the real two-connection editor reproduction passed.

## Verification

Executed on the installed direct Windows Godot 4.7.2 .NET engine, commit
`ed1daf0bf001b61586d9930840f2f1394092c079`:

- Go build, vet, tests with native Godot validation, shuffled tests and race tests
  passed. All touched Go source was formatted; CLI contract goldens were updated.
- Thirteen affected scripts passed bounded native `--check-only`. All 28 isolated
  standalone Godot regressions passed. Existing standalone editor teardown
  scan/RID/ObjectDB messages were retained; no script/parse error passed the runner.
- Final isolated GUI/headless QA passed 37 real CLI calls plus a two-connection
  synchronous-retirement scenario. Fresh HOME/USERPROFILE/APPDATA and explicit
  runtime PIDs isolated discovery and save data. Only owned processes were stopped.
- GUI captures reported actual 640×360 PNGs against 960×540 project design size;
  resizing returned 480×270 without upscaling. The live button changed from
  `Count 0` to `Count 1`, with both a PNG delta and a successful count assertion.
  A deliberate count mismatch failed. Paused capture, stale runtime rejection,
  headless unavailability, missing sequence action, hash conflict, unsaved scene,
  locked save and readonly save paths were exercised.
- Packaging inspection found the addon plugin manifest, addon license and both
  evidence helpers in the local addon archive. No release/version changes occurred.

The controlling agent independently rebuilt source commit
`3e38db70dba9985dbbcba79cdd13b7cd64b485bc` and reran the complete parameterized
live driver against fresh isolated evidence. It independently confirmed exit 0,
all 37 CLI calls and synchronous retirement, read the applied/succeeded/failed
lock result and separate QA timestamps, and visually inspected the 640×360
`Count 1` PNG. This manual QA remains bound to that unchanged source commit;
subsequent changes in this milestone are documentation only.

The final editor log was available and complete, but deliberately not clean:
two injected safe-save errors and one subsequent engine dialog-parent error,
zero warnings, zero dropped entries. All owned editor/runtime stderr files were
checked for script/parse/freed-instance errors and had none in the final run.

## Measured output and limits

One successful validation response was 184 bytes in legacy mode and 1108 bytes
with evidence. Final editor-preview evidence was 643 bytes; runtime evidence was
844 bytes; failed locked/readonly save evidence was 725/726 bytes. These are raw
stdout sizes including newline from one fixture, not token or latency benchmarks.
Latency percentiles, token use and capture byte deltas against a baseline were not
measured. The original failing lock output and final failing-persistence output
remain in private raw evidence.

The detailed command contract is in [Commands](../COMMANDS.md#linked-evidence-opt-in).
Hashes are separate observations, not an atomic snapshot. Editor buffers, loaded
scripts and running code are unavailable to disk validation. Capture frame counters
are sampled at readback; animation stability and state/image synchronization are
not claimed. Capture operation IDs are caller-supplied correlation labels, not
receipt lookup or causality. Scene memory equivalence remains unknown; readable
disk evidence does not imply full persistence. Resource verification covers only
supported selected properties, not nested object/container dependencies or durable
storage-device guarantees. Receipts remain in-process and disappear on restart.
Only the installed Windows engine was executed; older engines and other platforms
were not revalidated in this milestone.

The existing sequence missing-InputMap check was preserved. Direct action input's
pre-existing acceptance of unknown action names was observed during driver setup
and was not changed in M5. Native dependency initialization remains possible during
validation/fresh resource loading; see [Security](../SECURITY.md).

No gopls installation was attempted: the user had already declined it. Post-edit
hooks also warned that a private temporary QA script was outside the LSP cwd.
Those tooling warnings are distinct from Go/Godot compilation and from approval
policy; no approval-policy rejection occurred in this M5 run. Temporary project
cleanup was paused by the controller while investigating reported permission
messages. Private logs and the parameterized QA helper were
retained for the final independent review and manual QA.

## Final review correction: batch evidence

Final review found that batch children bypassed linked-evidence preflight and
guarded save actions. The CLI now applies both to non-nested children, including
hash-only scene/resource save preconditions. Missing/unavailable child evidence
becomes an explicit failure with its data preserved; ordinary batches keep their
existing output and exit behavior. This is a Go CLI correction; addon source is
unchanged from the independently exercised M4/M5 surfaces above.

Focused regressions failed before the correction and passed afterward (32 table
cases). Fresh Go build, vet, full uncached tests, shuffled tests, race tests, and
contract golden regeneration passed; regeneration produced no golden changes.
A parameterized real-binary loopback harness exercised 12 negative cases against
legacy, replaced-after-preflight and evidence-free response doubles. The prior
binary failed all 12; the corrected binary passed all 12. Legacy doubles received
only status; replaced doubles received guarded actions and performed zero modeled
legacy mutations. These are transport/CLI observations, not new native Godot
executions. The prior installed-engine verification remains applicable to the
unchanged addon; its older-engine/platform and evidence limits remain unchanged.
