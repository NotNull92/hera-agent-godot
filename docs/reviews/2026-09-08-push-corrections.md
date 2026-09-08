# Pre-push re-review corrections — 2026-09-08

Fresh source review at `2f10af4fb4fc32299177bef61dfeac105e06e046` found two additional failures after the original correction pass. Existing Go tests and all 16 Godot regressions passed at that revision; the new edge cases therefore needed failure-distinguishing regressions.

## QA lifecycle actions

`run` scenario steps accepted `action:"stop"` and `action:"state"` but always waited for `playing=true` when `wait:true`. The new HTTP regression failed for both cases: stop sent another `run/state` request where runtime-instance shutdown polling was expected; state sent an extra request after its snapshot.

Execution now shares one run/stop branch. Play actions wait for playing/runtime readiness; stop waits for stopped editor and runtime instances; state returns directly. Rejected responses still return without polling. No new command or dependency was introduced.

Live CLI verification on an isolated Godot 4.7 editor: set main scene to Other, then execute three requirement-covered steps (run, run-stop alias, stopped state snapshot). Result: `ok:true`, three requirements covered, none missing. Runtime instances were empty afterward; final diagnostics were readable with zero errors/warnings. `script validate`, status, scene tree and editor smoke also passed.

## Reparent ownership

When a moved node and its descendant are owned by the old parent, native reparent clears the owners that are no longer ancestors. Undo previously restored only the moved node's owner. The isolated reproduction observed a null descendant owner, and packing its original owner omitted `Item/Descendant`.

Reparent now reuses the existing subtree ownership snapshot and restores each original owner after undo reattachment. The strengthened existing regression covers root-owned and parent-owned subtrees, nested ownership, name collisions, ordering, global/local transforms, repeated redo/undo, and PackedScene round-trip preservation. It failed before the fix and passed afterward on the real Godot EditorUndoRedoManager.

## Verification and boundaries

- Fresh Go build/vet and uncached shuffled full tests passed; focused lifecycle tests passed after their initial failures.
- All 52 addon scripts passed Godot check-only before this delta; the changed node tool then passed its own fresh check and the strengthened editor regression.
- Dedicated final-commit review reports and raw logs are kept under the local Git metadata directory `hera-push-review`; exact revision/verdict pairs are recorded there before push.
- No user-project changes, installation, release, dependencies or MCP server. Save-error rollback remains outside scope. The documented local Windows race-test restriction still applies; Godot 4.2 execution was not repeated locally.
