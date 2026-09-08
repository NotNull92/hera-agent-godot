# Resource validation delta security review — 2026-09-08

Verdict: **PASS**. No new security blocker found in the resource codec non-string input branch and Gradient regression additions.

Base commit: `d6543c421e9b830301a8e0329c7ba56eaa2a1fd8`.
Source snapshot SHA256: `23b9ba74f80ea0571e743015e46d4a1cfbff070404d85eae42b50a8dbae3e3ad`.
Both were independently verified. The snapshot covers 46 changed/untracked non-document files, ordinal-sorted paths, hashing UTF-8 path + NUL + raw bytes + NUL.

## Evidence

- `addons/hera_agent_godot/tools/resource_value_codec.gd:36` accepts matching native types, null for object properties, and the two explicit numeric conversions. Other raw JSON type mismatches fail. In particular a Dictionary cannot reach a PackedFloat32Array setter.
- The float-to-integer condition checks finiteness and the signed 64-bit interval `[-2^63, 2^63)` before either integer conversion. The final equality rejects fractional numbers. The upper bound is exclusive, avoiding the rounded float representation of the signed maximum. Native integers remain integers; integer-to-float conversion introduces no resource loading or execution.
- `apply_props` stages all coerced values before its setter loop. Both callers (`resource_tool.gd:86` and `:114`) return on rejection before saving. The delta adds no filesystem operation, resource load, expression evaluation, or setter before validation completes. Existing string/object handling is unchanged.
- `tests/headless/resource_mutation_test.gd:23` checks a Dictionary supplied for Gradient offsets is rejected while both the earlier resource name and offsets remain intact. Following cases cover integral acceptance, fractional rejection without earlier mutation, and oversized-number rejection. The coordinator also reports a live CLI rejection preserving the original name and offsets; this is supplied execution evidence, not an independent rerun by this lane.

The unchanged security surface and existing limits are covered by [the prior security review](2026-09-08-final-security.md), stamped `81fa9cf375ea75f4fcf6c487b08768cf5862d47083243c14987ddda8ca873410`. That review is consumed for unchanged code only. This delta does not strengthen the trust model or promise disk-save rollback. Exact lower/upper integer boundaries and nonfinite values are source-reviewed here but are not explicit new regression cases. This lane performed source review and snapshot verification only; it did not run Godot or change source files.
