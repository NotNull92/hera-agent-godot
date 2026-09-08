# Final context and compatibility review — 2026-09-08

Verdict: **PASS for the context/compatibility lane at the reviewed snapshot**. No additional missed requirement was found in this lane. This is not overall implementation approval: the independent quality lane's subsequent R6 resource-coercion finding requires correction and a fresh delta review.

- Base: `d6543c421e9b830301a8e0329c7ba56eaa2a1fd8`.
- Dirty source SHA256: `81fa9cf375ea75f4fcf6c487b08768cf5862d47083243c14987ddda8ca873410`.
- Independently reproduced the digest over 46 changed/untracked non-document files using ordinal path sorting and path + NUL + file bytes + NUL. The base SHA alone does not identify reviewed work.

## Evidence and compatibility

Read `AGENTS.md`, the original `2026-09-07-code-review.md`, the implementation plan, `2026-09-08-review-fixes.md`, the support matrix, changed public documentation, CI/runner, and connected main-scene, batch, runtime geometry, input-name and path-policy callers. Used the git-master skill in HISTORY/STATUS mode; no Git history or source mutation.

`git show d7cb0f93acca27926201e4acec8eb867c150ee5b -- cmd/project.go cmd/run.go` confirms that **Add project set-main-scene command** introduced the direct disk writer and default-run disk reread together. They were one no-restart mechanism. The correction removes both sides together: `runProject` routes through the guarded editor RPC, `_set_main_scene` changes and saves ProjectSettings, and `runRun` sends native `play_main` to the existing run tool. The evidence report records isolated Old-to-New disk and in-memory agreement followed by bare `run --wait` launching `/root/New`. This lane inspected that reported proof and the code, rather than claiming to have repeated its live editor run.

`cmd/project_rpc_test.go` preserves output fields, editor rejection propagation, and multiple-editor rejection without issuing a request. Its bare-run fixture deliberately supplies an old disk setting and requires `play_main`, protecting removal of the workaround. `tests/headless/main_scene_write_test.gd` checks engine serialization of the no-final-newline fixture, memory/disk agreement, response fields, and rejected paths preserving settings. Batch still goes through the same registry/ProjectTool; the added `project_path` field is additive for existing batch callers. Save failure rollback is not promised by the correction evidence.

Ran `go test ./cmd -run 'Test(MainScene|BareRun|Contract)' -count=1`: PASS (`ok`, 1.719s). No contract-golden content changed. The setter retains the documented `main_scene` and `project_path` keys; its editor-produced Windows path spelling follows Godot's existing output conventions. Default compact output, targeting flags, explicit tool registry, and the CLI-to-loopback architecture remain intact. No MCP implementation, dependency change, installation, release or version bump is included.

The shared lexical path checker retains caller-owned prefix/extension rules; resource-list root allowance remains explicit. Variant syntax hints and input-name formatting reuse existing definitions rather than changing coercion policy. The R6 coercion correctness finding belongs to the separate quality review and is not overridden by this context verdict.

The CI change invokes every `tests/headless/*_test.gd` on the 4.7 row, with fresh copied projects and separate user directories. Editor-only fixtures opt in through their marker. Import/test execution is bounded and script/parser errors fail the runner; shutdown diagnostics remain visible. Existing 4.2/4.7 static validation and Go race coverage remain present. Documentation correctly separates this new behavioral tier from the older-version static floor and does not claim a fresh full version matrix.

Checked maintained upstream Godot 4.2 source documentation using CLI HTTP reads: [EditorInterface](https://github.com/godotengine/godot/blob/4.2-stable/doc/classes/EditorInterface.xml) includes `play_main_scene`; [CanvasItem](https://github.com/godotengine/godot/blob/4.2-stable/doc/classes/CanvasItem.xml) defines the local-to-viewport transform; [StreamPeer](https://github.com/godotengine/godot/blob/4.2-stable/doc/classes/StreamPeer.xml) includes partial writes and their result structure. These key API changes do not introduce a newer engine API requirement. This is a source compatibility check, not a fresh 4.2 execution claim.

README EN/KO, COMMANDS, CONTRACT, ARCHITECTURE, ROADMAP, SECURITY and HEADLESS_CI document the changed QA semantics and mutation/transport boundaries. Contract text explicitly preserves missing-availability legacy responses, distinguishes omitted warning limits from zero, and limits preflight guarantees to checked fields. The prior discovery retry documentation mismatch is corrected. Existing external state was not changed.

## Remaining gate boundary

Full combined Godot and live QA results belong to the independent QA lane; the evidence document still marked final static/standalone gates pending when read. This report does not substitute for those results. The parent reported a later quality finding requiring a resource codec/test delta; this PASS covers only the stamped snapshot and context lane, not that future delta or the overall merge gate. The worktree remains dirty and no commits were made. Only this report was written.
