# Independent final quality review — 2026-09-08

Verdict: **FAIL — one major blocker; no critical blockers identified.**

Reviewed base SHA: `d6543c421e9b830301a8e0329c7ba56eaa2a1fd8`.
Reviewed dirty-source SHA256: `81fa9cf375ea75f4fcf6c487b08768cf5862d47083243c14987ddda8ca873410` (supplied frozen snapshot; sorted changed/untracked non-document paths, path + NUL + bytes + NUL). This verdict applies to that snapshot, not future edits.

Read the original `2026-09-07-code-review.md`, the implementation evidence `2026-09-08-review-fixes.md`, current implementation diffs, new source helpers and regression tests, and relevant unchanged callers.

## Major blocker Q1 — R6 preflight still accepts incompatible native JSON values

Locations: `addons/hera_agent_godot/tools/resource_value_codec.gd:35`–`36`, mutation at `:23`; persistent callers `addons/hera_agent_godot/tools/resource_tool.gd:86`–`89` and `:114`–`117`.

`_coerce` accepts every non-string raw Variant without checking the property's type. Both resource set and create pass arbitrary `params.props` through this helper; the batch/RPC transport can supply JSON objects, numbers, arrays, booleans and null. Staging these values does not validate them. A dictionary for a packed float array passes preflight and is silently converted by Godot to an empty array. The tool then reports success and its callers proceed to save the resource.

Executed reproduction on Godot 4.7 using an isolated copied addon and a fresh Gradient:

```gdscript
var resource := Gradient.new()
resource.resource_name = "original"
var result: Dictionary = Codec.apply_props(resource, {
    "resource_name": "changed", "offsets": {"bad": true}
})
```

Observed stdout:

```text
RESULT={"ok":true,"properties":{"offsets":"[]","resource_name":"changed"}} NAME=changed OFFSETS=[]
```

The Gradient's default `[0, 1]` offsets were cleared and the first property changed. This is invalid-input preflight, not the excluded disk-save rollback boundary. The existing resource mutation regression checks invalid property names and invalid string values, leaving this supported transport form uncovered.

Required correction: validate/coerce non-string values against the destination type before applying any setters, preserving intentionally supported numeric/array conversions. Add a regression with a valid first property and wrong-typed native JSON second property that asserts rejection and preservation of both original values. Also keep valid native typed values covered.

Reproduction fixture: `C:/Users/PC/AppData/Local/Temp/hera-quality-05c3e4c759754dfeb67db2a258618d01/repro.gd`. No source edits or user-project mutations were made.

## Other reviewed flows

- HTTP connection entries remain tracked through reading, queued async work, and bounded writes; inline errors retain their entries. Async late responses safely find no active connection after stop/disconnect. The plugin polls before draining the queue, so response lookup sees the retained entry. Token failures prevent listener and heartbeat startup.
- Remove/undo snapshots the entire subtree's owners before detachment, restores after reattachment, and holds an undo reference. Resolved-node containment/root checks cover aliases and the shared resolver's callers; regression covers undo/redo and saved descendants.
- Theme changes stage all groups before setters. Resource string validation also stages correctly, subject to Q1.
- Runtime request publication closes a unique temporary file before renaming to the consumed JSON name. The runtime removes only a parseable dictionary before dispatch/await; response identity and PID selection remain intact. Self-freeing calls capture path before invoking user code.
- UI tree, click targets and audit use the shared canvas-to-viewport transform. Click centers use transformed local centers, and another viewport is explicitly rejected. Tests cover translated/scaled CanvasLayer input, camera positions, viewport bounds and cross-viewport rejection.
- Scenario preflight runs before editor discovery; required values distinguish omission from explicit null. Diagnostics decoding is shared, rejects unavailable/malformed evidence and invalid counts, and preserves omitted-versus-zero warning thresholds. Failed steps do not earn successful requirement coverage. Parameter preflight remains intentionally limited as stated in implementation evidence.
- Main-scene CLI mutation uses the guarded project RPC and bare run uses native `play_main`; the existing editor setter persists ProjectSettings. The no-final-newline writer was deleted. RPC contract/guard tests and the documented isolated live round trip support convergence.
- Shared path checks, Variant hint text and input-name helpers remove duplication without new dependencies. CI invokes standalone regressions in separate fixtures and user directories. No unrelated framework or abstraction was introduced.

## Verification limits

This lane executed the targeted Godot reproduction and reviewed test assertions; the parent owns final Go/Godot suite execution. The supplied live checks were read as evidence, not rerun or represented as this lane's executions. Save-error rollback, arbitrary scripted-setter side effects, and exhaustive engine/version/security coverage are outside this verdict.
