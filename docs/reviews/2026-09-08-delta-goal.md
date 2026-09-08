# Goal and contract delta review — 2026-09-08

Verdict: **PASS for Q1 correction and R6 contract compliance within the reviewed delta.** Other goals inherit the inspected coverage in [the original goal review](2026-09-08-final-goal.md); this is not a fresh execution of the full suite or release approval.

Base SHA: `d6543c421e9b830301a8e0329c7ba56eaa2a1fd8`.
Original source fingerprint: `81fa9cf375ea75f4fcf6c487b08768cf5862d47083243c14987ddda8ca873410`.
Assigned final source fingerprint: `23b9ba74f80ea0571e743015e46d4a1cfbff070404d85eae42b50a8dbae3e3ad`.
Fingerprints are coordinator-supplied SHA256 stamps of sorted changed/untracked non-document paths, path + NUL + bytes + NUL. This verdict binds the assigned final snapshot.

Read both original goal/quality reports, the current codec and regression, both persistent resource callers, the approved R6 plan, implementation evidence, and relevant command/contract text. Reviewed correction scope is non-string validation in `addons/hera_agent_godot/tools/resource_value_codec.gd` and the Gradient regression in `tests/headless/resource_mutation_test.gd`.

- Q1 is addressed at the shared coercion boundary (`resource_value_codec.gd:33`). A Dictionary no longer passes as packed float offsets: incompatible non-string types return failure. `apply_props` completes coercion into its staged dictionary before the setter loop (`:22`), so failure also preserves the valid first property. Both resource set and create stop on that failure before saving (`resource_tool.gd:86`, `:114`).
- The regression (`resource_mutation_test.gd:22`) repeats the original valid-name/invalid-offsets request and asserts rejection plus preservation of both the original name and duplicated offsets. It additionally checks accepted integral numeric input, rejected fractional integer input without preceding mutation, and rejected out-of-range input. Exact native types and integer-to-float input remain accepted in the implementation; float-to-integer input must be finite, in range, and integral. Packed-array Godot literal strings retain their existing parsing route. `docs/COMMANDS.md:53` documents Godot literal strings, not an implicit native JSON Array-to-packed-array conversion guarantee.
- Coordinator evidence reports a failing regression before correction, passing regression after correction, and an actual live batch rejection of Dictionary-valued offsets while retaining `resource_name=original` and offsets `[0,1]`. These executions are supplied evidence, not reruns by this review lane.
- R6 requires validation before setters, as written in the approved plan. `docs/CONTRACT.md:281` explicitly distinguishes rejected property batches from disk-save-error rollback. The correction matches this boundary; it does not claim a transaction over filesystem failures or arbitrary scripted setter side effects.

The original Q1 FAIL remains accurate for its original stamp and is superseded for this delta only. Full-suite final gates remain the coordinator's responsibility. No implementation files were changed by this lane.
