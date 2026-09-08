# Review Corrections Implementation Plan

> For agentic workers: execute the approved review task-by-task, with isolated regressions and integration verification.

Goal: correct the demonstrated data-preservation, transport and QA failures and consolidate the small shared policies identified in the accepted review.
Architecture: preserve Go CLI -> loopback HTTP -> editor addon -> process-isolated runtime. Fix existing boundaries without adding dependencies, generic schedulers, or a new protocol surface.
Tech stack: Go standard library, typed GDScript, Bash test runner, Godot 4.7 behavioral validation and existing 4.2/4.7 static CI.
Spec: docs/reviews/2026-09-07-code-review.md (reviewed source d6543c421e9b830301a8e0329c7ba56eaa2a1fd8).

## Constraints

- Compact stable CLI output and explicit mutation targeting remain authoritative.
- Preserve per-node ownership on undo, and process identity in runtime transport.
- Use isolated test projects and user-data directories; do not mutate the user's open project during QA.
- No release/version changes or external publication. No new dependencies.
- Each owner reads connected callers, writes a failure-distinguishing regression, and records red/green evidence.

## Work and ownership

- [x] Scene/resource owner: node_tool.gd, resource_value_codec.gd, theme_tool.gd; add node_remove_undo_test.gd and resource_mutation_test.gd. Snapshot original subtree owners, reject resolved roots/outside nodes, validate all requested properties before setters. Prove undo/redo/pack and rejected partial updates.
- [x] HTTP/auth owner: http_server.gd, plugin startup integration, SECURITY.md, http_server_test.gd. Poll partial writes with a deadline and reject configured-token read errors. Prove nonreading client does not block ticks, normal replies and auth rejection.
- [x] Runtime owner: runtime helpers, game_tool.gd, runtime_request_test.gd and runtime_geometry_test.gd. Publish complete requests, snapshot call target before user code, share viewport geometry. Prove partial publication, self-free response, CanvasLayer transforms and viewport identity.
- [x] Go QA owner: project.go, game_qa*.go and tests, diagnostics/output read error handling. Remove the text writer after native editor convergence proof, share diagnostics validation, and preflight scenarios before execution. Prove malformed scenarios issue no requests and unavailable diagnostics cannot satisfy coverage.
- [x] Coordinator: tests/headless/run.sh and CI wiring; small remaining path-helper reuse; docs synchronization. Runner copies addon/tests into a fresh project, uses editor mode only where required, bounds each test and rejects script errors.
- [x] Coordinator: combined Go build/vet/shuffled tests/format and all-addon Godot check-only; execute standalone regressions and real CLI on an isolated editor with the current addon.
- [x] Fresh review: inspect final diff and evidence; correct findings before reporting. Preserve explicit limitations and remove task-owned processes/artifacts.

Main-scene setter convergence is conditional on proving immediate disk/in-memory/default-run equivalence; retain the existing workaround if that cannot be established. Small pure path and syntax helpers may be consolidated within owned files without changing object-property coercion policies. Report any deferred structural proposal with evidence rather than silently claiming it shipped.

Completed: main-scene convergence proof passed, so the duplicate writer/workaround was removed. Initial independent review found Q1; the focused correction passed red/green/live QA and all five fresh delta review lanes. Final evidence and limitations: [review corrections](../../reviews/2026-09-08-review-fixes.md).
