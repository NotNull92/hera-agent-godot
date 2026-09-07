# DoneClaim: task 22 repair Hera external runtime

## Claim matrix

| Criterion | Scenario and invocation | Binary observable | Artifact |
|---|---|---|---|
| Baseline embedded flow | `hera run --scene res://game/Main.tscn --wait`; `hera game screenshot --analyze` | PID 25626, 927×1649, nonblank | `baseline/embedded-screenshot.json`, `baseline/embedded.png`, `baseline/embedded-ihdr.txt` |
| Failing-first external route | Direct installed Godot at `--resolution 1080x1920`; then unqualified `hera game screenshot` | PID 25806 listed; exit 1 with editor-play refusal | `red/external-instances.json`, `red/external-screenshot-failure.json`, `red/external-screenshot-exit.txt` |
| Explicit PID routing | External launcher PID 28969; evidence bridge sends `params.pid=28969` | RPC `ok=true`, response PID 28969, runtime scene resolved | `green/external-screenshot.json`, `manual-qa.md` |
| Exact repo CLI syntax | `tools/hera --json game --pid 33533 screenshot --path ... --analyze` | exit 0; result `ok=true`; PID 33533 | `cli/green-exact-cli.json`, `cli/green-exact-cli-observable.txt`, `cli/green-native.png` |
| Native 1080×1920 screenshot | Same PID-targeted screenshot with analyzer | IHDR `0000043800000780`; 1080×1920; nonblank; 67 colors | `green/native-1080x1920.png`, `green/native-ihdr-xxd.txt`, `green/native-ihdr.txt` |
| Default selection unchanged | Pure selection test plus real pre-change embedded characterization | One editor-play runtime remains implicit; multiple matching runtimes fail | `tests/target-selection-final.log`, baseline artifacts |
| Invalid and stale PID | Evidence bridge with `nope`, 999999, and stopped PID 29377 | exits 2/1/1; positive integer parse error or no live process | `tests/malformed-pid.log`, `tests/missing-pid.log`, `tests/stale-pid.log`, `tests/edge-summary.txt` |
| Multiple instances without PID | Two isolated external runtimes; installed CLI screenshot without PID | exit 1 and explicit instruction to pass PID | `tests/edge-instances-two.json`, `tests/multiple-without-pid.log` |
| Bounded timeout | SIGSTOP live PID 29351, PID-targeted screenshot with wrapper timeout 5 | addon timeout error in 3.112 seconds, exit 1 | `tests/hung-timeout.log`, `tests/edge-summary.txt` |
| Repeatability | Two fixed-FPS PID-targeted captures of PID 29351 | both IHDR values `0000043800000780` | `tests/repeat-1.png`, `tests/repeat-2.png`, `tests/repeat-ihdr.txt` |
| Addon checks and integration build | Two Godot script tests; BasedPyright; `dotnet build monstel.csproj --no-restore` | both PASS; 0 type errors; build 0 warnings/0 errors | `tests/final-verification.log` |
| Cleanup | audit every task PID; delete isolated temp homes/editor project | all 15 task PIDs dead; no task temp paths remain | `cleanup/pid-audit.txt`, `cleanup/temp-dirs-after.txt`, `cleanup/manual-green-cleanup.txt`, `cleanup/edge-cleanup.txt` |

## Source and scope

Product/UI files, `project.godot`, plan, ledger, user settings, installed CLI,
and global packages were not edited. Assigned changes are limited to the addon
and the repo-owned `tools/hera.py` plus `tools/hera` symlink. The wrapper handles
the installed CLI's missing parser through its existing batch transport. See `source.diff`,
`source-diff.sha256`, `source-scope.txt`, and `worktree-status.txt`.

The real `project.godot` and editor settings hashes stayed identical. Early
non-isolated probes restored the captured save snapshot after stopping their
PIDs; later probes used isolated HOME directories. The final real save hash
differs because concurrent non-task embedded PID 27265 was observed writing the
same save during verification, so restoring the older bytes again would have
overwritten another worker's state. Evidence: `cleanup/state-audit.txt`.

The exact CLI rerun used an isolated HOME and stopped PIDs 33533/33504;
`cli/cleanup.txt` proves both dead and the temporary home removed. Concurrent
non-task embedded PID 32213 was recorded in
`cli/foreign-runtime-attribution.txt` and was not stopped or mutated.
