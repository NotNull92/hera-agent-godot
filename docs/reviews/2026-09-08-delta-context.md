# Resource correction context delta review — 2026-09-08

Verdict: **PASS for the context/compatibility delta lane.** No blocking contract or compatibility regression found in this correction.

- Base: `d6543c421e9b830301a8e0329c7ba56eaa2a1fd8`.
- Initial reviewed source SHA256: `81fa9cf375ea75f4fcf6c487b08768cf5862d47083243c14987ddda8ca873410`.
- Corrected source SHA256: `23b9ba74f80ea0571e743015e46d4a1cfbff070404d85eae42b50a8dbae3e3ad`.
- Snapshot stamps supplied by the parent; this lane did not independently recompute them.

Read the stamped final-context and final-quality reports, the full current resource codec and mutation regression, resource set/create callers, CLI property parsing, shared Variant hints, and advertised COMMANDS entries. Scope is the non-string validation correction and Gradient regressions; the initial context findings remain applicable outside that delta.

Both persistent resource callers return on codec failure before saving. The codec validates every property before its setter loop. A Dictionary for Gradient offsets now fails the destination-type check while the staged resource name remains unapplied, addressing Q1 at the shared boundary. The added regression asserts rejection and preservation of both original values.

CLI properties remain strings (`parseResourceProps` constructs `map[string]string`), and the complete string coercion path remains intact. Godot Variant text for packed arrays, colors and other complex types still reaches the existing parser and destination-type check. Native Array-to-packed-array conversion is rejected by the correction; this does not remove an advertised form, since resource set/create advertise Godot literal strings and packed-array hints explicitly show the constructor syntax. Exact native types, null object values and integer-to-float values remain accepted.

Integral JSON numbers remain usable for integer properties: finite float values in the signed 64-bit range are converted only when the conversion preserves their value. The exclusive upper bound avoids converting positive 2^63. Fractions and out-of-range floats reject before mutation; the new tests cover 1.0, 0.5 and 1e30. Godot documents JSON number parsing through floating-point conversion, so this branch preserves the transport's expected integer use. It cannot recover integer precision already lost in JSON float parsing; existing integer text remains available. See maintained upstream [Godot 4.2 JSON documentation](https://github.com/godotengine/godot/blob/4.2-stable/doc/classes/JSON.xml).

The newly used `is_finite` exists at the supported 4.2 floor in maintained upstream [GlobalScope documentation](https://github.com/godotengine/godot/blob/4.2-stable/doc/classes/%40GlobalScope.xml). This is source compatibility evidence, not a fresh 4.2 execution claim. No public command, success-response field, dependency or version changes are introduced by the delta.

Parent evidence reports red/green verification and an actual live batch rejecting Dictionary offsets while preserving the original name and packed array. This lane inspected the regression assertions and code; it did not rerun the engine or live editor. Final combined test gates remain the parent's responsibility. Arbitrary scripted setter effects and disk-save rollback remain outside this correction's guarantee. Only this report was written; no source changes or commits were made.
