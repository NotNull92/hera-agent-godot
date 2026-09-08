# Whole-code review and refactoring proposals

Reviewed source: `d6543c421e9b830301a8e0329c7ba56eaa2a1fd8`.
Date: 2026-09-07. Verdict: **changes recommended**. This is a review, not an implementation or release approval. No product code was changed. The user's open project was inspected only with read commands; reproductions used temporary copies.

## Findings

P1 means prioritize because of data loss or editor availability. P2 means a concrete correctness, security-configuration, or regression-coverage problem. Runtime reproduction and static analysis are distinguished below; no failure frequency is inferred from static evidence.

### R1 — P1: subtree deletion undo loses saved descendants

Location: `addons/hera_agent_godot/tools/node_tool.gd:217`.

`_remove_node` saves/restores only the removed node's owner. Removing a branch also clears descendant ownership when its owner is outside that branch. Undo restores the hierarchy but not those descendants' ownership. Packing/saving can consequently omit them.

Isolated reproduction with the actual EditorUndoRedoManager on Godot 4.7: a three-node Scene/Branch/Leaf becomes a two-node PackedScene after remove/undo. Evidence: `branch_owner_restored=true`, `leaf_owner_restored=false`, `packed_nodes_after_undo=2`, `remove_ok=true`.

Proposed correction: snapshot each affected descendant's original owner and restore it after reattachment. Preserve nested scene ownership; assigning every descendant to the edited root is not equivalent. Regression: remove, undo, pack, instantiate, and compare saved hierarchy and owners.

### R2 — P1: equivalent paths bypass root-removal protection

Location: `addons/hera_agent_godot/tools/node_tool.gd:203`.

The guard checks the raw path for empty or `.` but never checks the resolved node's identity. `Branch/..` resolves to the scene root and is accepted. Isolated copied-addon reproduction returned `remove_ok=true` and `scene_detached=true`. The harness used a synthetic scene, so the precise editor UI aftermath was not tested.

Proposed correction: reject `node == root` after resolution, as reparent already does. Review containment for editor mutations consistently. Regression: direct, relative-equivalent, and absolute root paths must all reject without scene changes.

### R3 — P1: HTTP response backpressure can freeze the editor

Location: `addons/hera_agent_godot/server/http_server.gd:114`.

HTTP replies call `put_data` on the editor main thread. A client that requests a response larger than socket buffers and stops reading can block that thread; the receive-side five-second timeout cannot run while the write blocks. Editor UI and heartbeat publication then stop progressing. This is a source/API trace, not a reproduced freeze or an attribution of any earlier incident.

Godot documents blocking behavior for [StreamPeer.put_data](https://github.com/godotengine/godot/blob/master/doc/classes/StreamPeer.xml); the [socket implementation](https://github.com/godotengine/godot/blob/master/core/io/stream_peer_socket.cpp) waits for writable output in the blocking path.

Proposed correction: retain pending response bytes and offset in connection state, advance with `put_partial_data` during polling, and enforce a write deadline. Regression: a nonreading loopback client must not stop editor ticks/heartbeats. A thread pool or new server framework is unnecessary.

### R4 — P2: main-scene writer corrupts a valid file without a trailing newline

Location: `cmd/project.go:234`.

When the final application section has no existing main-scene key and no trailing newline, insertion concatenates the new setting with the previous line. The freshly built CLI returned exit 0 and wrote `config/name="X"run/main_scene="res://Main.tscn"` on one line. An otherwise identical fixture with a trailing newline worked. This was reproduced through the real CLI with isolated discovery and a temporary project.

Proposed correction: handle the line boundary before insertion and cover both fixtures. Separately evaluate convergence with the addon's existing ProjectSettings writer; preserve immediate `set-main-scene` followed by bare `run`, disk/in-memory agreement, batch behavior, instance guard, and output contract. History shows disk writing and disk rereading were introduced together to support the no-restart workflow.

### R5 — P2: scenario QA can pass without readable diagnostics

Location: `cmd/game_qa.go:237`.

`validateDiagnosticsThresholds` ignores `available:false` and failed count conversions. The addon deliberately returns unavailable/unclean with zero counts when logging is absent or disabled. That step passes, and its `covers` entries count toward successful requirement coverage. The sibling diagnosis command already rejects explicitly unavailable diagnostics.

Evidence: traced scenario dispatch, threshold validation, result assembly, and successful coverage; not a new runtime reproduction.

Proposed correction: share a small diagnostics decoder/validator between scenario and diagnosis paths. Distinguish unavailable evidence from zero errors, validate counts, and make warning-limit omission distinct from an explicit zero where the contract requires it. Regression: unavailable diagnostics must fail the step and leave its requirement uncovered.

### R6 — P2: rejected multi-property changes leave partial state

Locations: `addons/hera_agent_godot/tools/resource_value_codec.gd:17`, `addons/hera_agent_godot/tools/theme_tool.gd:101`.

Validation and setters are interleaved. If a later item fails, earlier items remain changed. Resource setters operate on a cached loaded resource, so subsequent consumers can observe the rejected edit and later saves can persist it.

Codec-boundary reproduction: applying `resource_name="changed"` followed by invalid `zz_invalid` returned `ok=false` while `name_after_failure="changed"`. Theme's analogous path was inspected, not separately reproduced. No disk persistence claim is based solely on this boundary test.

Proposed correction: validate/coerce all items first, then apply. Treat rollback on save failure as a separately tested boundary. Regression: a bad second property must leave the first unchanged.

### R7 — P2: partial runtime request publication loses commands

Locations: `addons/hera_agent_godot/tools/game_tool.gd:73`, `addons/hera_agent_godot/runtime/game_inspector.gd:69`.

The editor writes directly to the final JSON path. The runtime reads and deletes that path before parsing. If it observes a partial write, parsing fails after the request has been discarded; response polling cannot retry the lost request. Static interleaving analysis; frequency was not measured.

Proposed correction: write a unique temporary file, close it, then publish the final JSON name. Consumers inspect only published names. Regression: deliberately poll between creation and completion, then verify exactly one execution. Preserve PID targeting and response identity; do not automatically retry mutations at HTTP level.

### R8 — P2: UI targeting and audit mix canvas and viewport coordinates

Locations: `addons/hera_agent_godot/runtime/game_ui_inspector.gd:93`, `addons/hera_agent_godot/runtime/game_ui_audit_checks.gd:40`.

Clicks and bounds checks use `get_global_rect` directly as viewport coordinates. Translated/scaled CanvasLayers or camera transforms invalidate that assumption, producing wrong clicks and audit classifications. Static source/API verification; no rendered transformed fixture was run in this review.

[Control.get_global_rect](https://github.com/godotengine/godot/blob/master/doc/classes/Control.xml) returns canvas-relative geometry. [CanvasItem.get_global_transform_with_canvas](https://github.com/godotengine/godot/blob/master/doc/classes/CanvasItem.xml) converts local coordinates to the viewport.

Proposed correction: share explicitly viewport-space geometry across targeting, summaries, and clipping checks. Derive a click center from transformed local center. Handle another Viewport explicitly instead of injecting its coordinates into the root. Regression: translated CanvasLayer and camera
fixtures with expected target positions and bounds classifications.

### R9 — P2: self-freeing runtime calls fail during response construction

Location: `addons/hera_agent_godot/runtime/game_inspector.gd:219`.

After arbitrary `node.callv`, the code dereferences that same node for its path. A method that immediately frees itself leaves an invalid reference, so the operation can occur but its response fails. Deferred `queue_free` is not the immediate trigger. Static trace, not reproduced here.

Proposed correction: capture the path before calling user code and avoid dereferencing the target afterward. Regression: a disposable node's synchronous self-free helper returns a response without a timeout or script error.

### R10 — P2: configured token read errors silently disable authentication

Location: `addons/hera_agent_godot/server/http_server.gd:181`.

If a token exists but cannot be opened at startup and no environment override is set, the loader returns the same empty string as intentional auth-off. Plugin startup still opens the listener. This condition can expose the editor to other local accounts contrary to the configured-token boundary. Static trace; no credential permissions were changed on this machine.

Proposed correction: distinguish missing token from read failure, and refuse bridge startup for the latter. Also correct `docs/SECURITY.md:65`: under a permissive POSIX umask and traversable home, its token-generation example can create a world-readable secret. Document private directory/file permissions for both new and existing files. This is conditional POSIX guidance, not a finding about the current Windows ACLs.

### R11 — P2: existing behavioral regressions are not wired into CI

Locations: `.github/workflows/ci.yml:114`, `.github/workflows/ci.yml:394`.

CI parses addon scripts and executes `runtime-logic.json`, but does not call the nine standalone `tests/headless/*_test.gd` regressions. The scenario covers launch, counter mutation, and clean UI audit; it does not substitute for ownership, autoload lifecycle, target selection, clock sequencing, or script-language checks. Contract goldens mock addon responses and therefore cannot catch changed addon behavior.

Proposed correction: run the existing behavioral tests in isolated projects before structural refactoring. Respect their different editor/SceneTree setup requirements. Keep static version-floor checks and actual behavioral coverage separate.

## Refactoring order

| Order | Scope | Concrete outcome and guard |
|---|---|---|
| 1 | Existing test scripts and small reproductions | Connect behavioral tests to CI; add regressions for R1/R2/R4/R5 before touching their behavior. |
| 2 | Node mutation and project writes | Fix ownership restoration, resolved root identity and line insertion. Preserve undo/redo and pack/save results. |
| 3 | HTTP and runtime file publication | Bounded output and complete-file publication; prove editor responsiveness and exactly-once consumption. Keep WorkQueue simple. |
| 4 | QA and resource mutation boundaries | One diagnostics interpretation; complete validation before setters; preflight scenario tools/arguments before executing earlier steps. |
| 5 | Runtime UI geometry | One viewport-space conversion used by clicks/tree/audit, tested with transforms and viewport identity. |
| 6 | Small shared helpers | Reuse existing ProjectPathSafety across duplicated lexical guards; share pure Variant syntax hints and input-name conversions. Preserve caller-specific object/resource and path policies. |
| 7 | Main-scene setting convergence | Consolidate Go text writing and addon ProjectSettings writing only after a live disk/in-memory/run round trip proves equivalence. |

Avoid mechanically splitting files to meet a line-count threshold. The 713-line runtime inspector combines transport, clock, calls, and input logging, but extract only the boundaries above that have demonstrated costs. Keep the stdlib-only CLI, explicit tool registry, compact stable output, process isolation, owned-autoload export cleanup, and existing runtime helpers.

Further low-priority observations: architecture text still says one discovery retry while code intentionally performs four growing delays; diagnostics/output check file existence without explicitly handling a subsequent read failure. Update documentation and add a read-error case when touching that boundary. Neither establishes the cause of previous observed editor stalls.

## Verification and scope

- Go 1.26.2 Windows: fresh build, `go vet ./...`, `go test -count=1 ./...`, tracked Go formatting check passed.
- Godot 4.7: all 50 addon scripts passed check-only with no parser errors.
- Nine standalone headless test assertions exited 0. Two editor-mode tests emitted RID/ObjectDB/resource shutdown warnings; this is not a clean-log pass. Initial harness setup failures were corrected before the successful runs.
- Five installed CLI read commands passed against the already-open Godot 4.7.2 .NET test project. It was not playing. Diagnostics showed a clean previous game log, not proof of an error-free editor console.
- R1/R2 and R6 were reproduced in copied-addon fixtures; R4 through the freshly built CLI. Other findings explicitly state their static evidence.
- No user project mutations, product code changes, commits, releases, or dependency updates. Task-owned QA Godot processes were cleaned up.
- Race execution was skipped under the documented local antivirus limitation. No new cross-version matrix, GUI/input visual run, full .NET compilation, export smoke, or exhaustive security audit was performed.
- Review reading covered Go command/client/discovery/protocol code, addon tools/runtime/server/core/plugin, relevant tests, architecture/contracts/incidents, and packaging/CI/integration paths. This is broad source review with selected tests, not proof that every code path is correct.

## Review lane evidence

All lanes reviewed the exact source SHA at the top; all ended FAIL because of the findings above. No previous-turn approval was reused.

| Lane | Verdict | Local detailed report |
|---|---|---|
| Contract and behavior | FAIL | `C:/Users/PC/AppData/Local/Temp/hera-contract-review-0e58d03310574535a9a1e99503c58668/contract-review.md` |
| Executed QA | FAIL; existing gates passed | `C:/Users/PC/AppData/Local/Temp/hera-review-qa-1788775116613/REPORT.txt` |
| Runtime/server quality | FAIL; static findings | `C:/Users/PC/AppData/Local/Temp/hera-review-quality-d6543c421e9b830301a8e0329c7ba56eaa2a1fd8.md` |
| Security | FAIL; conditional medium issues | `C:/Users/PC/AppData/Local/Temp/hera-security-review-6edd403749db49a3915cf91137978a75.md` |
| Design/history/CI | FAIL; regression coverage gap | `C:/Users/PC/AppData/Local/Temp/hera-review-context-d6543c4-20260907.md` |

Temporary reports may be removed by OS cleanup; the core findings, reproduced outputs, proposed verification, and limitations are preserved in this document.
