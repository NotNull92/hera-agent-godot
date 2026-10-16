# Operation limits CI failure — 2026-10-16

## Finding

[CI 38869747650](https://github.com/NotNull92/hera-agent-godot/actions/runs/38869747650) failed the Godot 4.7 standalone `operation_limits_test` after the documentation-only commit `63c362b`. Two fixture admissions failed, followed by the expected-record-count assertion. The following documentation-only commit passed CI 38870328797 with the same test and production implementation.

The capacity test used a fresh wall-clock reading plus exactly 60,000 ms for its deadline. `operation_records.gd` measures now from its construction-time wall-clock origin plus elapsed monotonic ticks. Millisecond rounding/sampling differences can make the independently sampled deadline 1 ms greater than the ledger's maximum accepted future time. `accept` correctly rejects it as `invalid_operation: deadline must be within 60 seconds`; a missing fixture record then also fails the record-count assertion.

The original CI log did not include admission responses, so the precise clock offset in that runner cannot be recovered. The matching failure mechanism was reproduced locally: 8 of 10,000 admissions using the original construction were rejected, each recorded with `ahead_ms=1` and the deadline-limit error. This supports a fixture timing failure, not evidence of lost receipts under byte pressure.

## Fix

Only `tests/headless/operation_limits_test.gd` changes. It derives its deadline from `ledger._now()` with 30 seconds of headroom, as this test exercises capacity/evidence retention rather than the deadline boundary. Admission assertions now include the returned dictionary to preserve the actual rejection reason on future failures.

The production 60-second rule, record count/byte limits, result eviction behavior and all capacity assertions are unchanged. No addon/CLI release or version bump is required.

## Reproduction and focused validation

Evidence is retained under ignored `dist/ci-operation-limits/`:

- `clock-probe.log`: original deadline construction, 8/10,000 rejected; reported offset 1 ms.
- `before-skew.log`: original full capacity fixture with its ledger clock origin deliberately shifted back 1,000 ms, exit 1. This amplifies the observed clock difference for deterministic reproduction.
- `after-skew.log`: same shifted clock and same production code with the corrected test, exit 0.
- `after.log`: corrected full capacity test without injected skew, exit 0.
- `fixed-probe.log`: corrected construction, 0/10,000 rejected.
- Godot 4.7.2 check-only passes for both original and corrected fixture; the failure is behavioral, not a parse error.

No user editor, scene or recording terminal was touched. This is a standalone test-only change; live editor smoke and Go tests do not exercise the changed fixture and were not repeated.

## Final validation

Fix commit `146a9ec40aba83e7dca6ca1a3e0792cc86719c19` passed [CI 38870922140](https://github.com/NotNull92/hera-agent-godot/actions/runs/38870922140): all seven jobs, including the previously failing Godot 4.7 standalone regression job. Local Godot 4.2 check-only and focused execution also passed. The complete local Godot 4.7.2 standalone suite passed all 31 tests (`dist/ci-operation-limits/headless.log`). Existing engine headless teardown warnings remain; these passes do not claim globally warning-free engine exits.

The recording prerequisite is resolved. No new addon release, npm publication or store upload is needed for this test-only correction.
