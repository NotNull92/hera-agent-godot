# Wave 1 implementation result — 2026-10-23

Branch: `refactor/wave1-token-diet`, based on `532b5a3`.
Eleven numbered commits in the requested order; no merge, tag, version bump,
release, history rewrite, or changes to `CLAUDE.md` / `PROMPT.md`.
All live checks used disposable projects and isolated user directories. No user
editor was selected or modified. The branch is the review artifact.

## Commits and gates

Counts are Git added/removed lines across each whole commit, including tests and
docs; they are not a claim of production-code reduction alone.

| Item | Commit | Added | Removed | Gates before commit |
|---|---|---:|---:|---|
| 1. Docs truth and measured costs | `67e12d3d28d68325c926230b5e12c8139b13d3f6` | 9 | 7 | G: pass |
| 2. Avoid redundant verification reads | `c915b6d5888d4cfc36c18243bb962e147fff63dc` | 33 | 23 | G: pass |
| 3. Constrain scene node resolution | `5e3d1d1c9f3d11e8e6f01268c8849ec72bee74c1` | 53 | 7 | G + D: pass |
| 4. Shared value codec and log hint | `f2cc3082a5a78899e26e190dffc6318e64d24b13` | 149 | 193 | G + D: pass |
| 5. Remove impossible capture freshness checks | `1e7de2b8b6c98deb5532ff931bb469ad6fa86bfd` | 54 | 336 | G: pass |
| 6. Consistent equals-form value flags | `76cc075dca103bd8eb1419496b83bb05603ad0b6` | 643 | 466 | G: pass |
| 7. Typed eval errors and bounded validation | `9cab538fa9cf381be95e98c43efbee48259151a6` | 91 | 57 | G: pass |
| 8. Input-name dictionaries | `671a57bd3e0b7fd2dbbe9a5e44f4fddca906587b` | 111 | 184 | G + D: pass |
| 9. Immediate first HTTP response chunk | `386791533359cb95265f5ea3445d502a3de1cea4` | 49 | 11 | G + D: pass |
| 10. Focused editor reads | `8e174287db4736a4e18f40b45f3ab51aa77fe9bd` | 319 | 45 | G + D: pass |
| 11. Compact driver card and this report | containing commit¹ | 512 | 274 | G: pass |

¹ Item 11 is the commit containing this report. Its hash cannot be embedded in
its own contents. Resolve the exact hash with
`git log -1 --format=%H -- docs/handoff/2026-10-23-codex-wave1-result.md`.
This keeps item 11 the final, independently droppable docs-only commit instead
of introducing a twelfth bookkeeping commit.

**G**, run before every commit: `gofmt -l .` empty, `go build ./...`,
`go vet ./...`, `go test -race -shuffle=on -count=1 ./...`.

**D**, for every changed GDScript file and the plugin entry point:
`--headless --path <isolated-project> --check-only --script res://...` on both
`4.2.stable.official.46dc27791` and
`4.7.2.stable.mono.official.ed1daf0bf`, followed by the complete
`GODOT_BIN=<direct-4.7.2-executable> bash tests/headless/run.sh` suite.
The suite passed 32 tests at item 3, 33 at item 4, 34 at items 8/9, and 35 at
item 10. Final item-10 gates include the compatibility follow-up described below.
Docs-only and Go-only commits did not rerun GDScript gates.

Additional evidence:

- The scene-escape regression failed against the original resolver behavior,
  then passed after item 3; select, signal target, and instance parent are covered.
- Item 7 native engine tests passed for GDScript validation, bounded output,
  structured eval transport errors, and explicit `engine info --project`.
- Item 10 isolated live smoke passed: untouched Node2D `--changed` was empty;
  setting Child.visible=false returned and subsequently read false; scoped tree
  depth/type/props and IDs worked; connected signals retained external targets;
  ancestor select/tree paths failed. `smoke --skip-game` passed. Editor diagnostics
  reported available=true, complete=true, dropped_count=0, errors=0 and warnings=0
  for the observed interval. This does not assert anything about earlier sessions.
- Extra 4.2 execution passed scene containment, value codec, input dictionaries,
  and read-narrowing tests. The full required behavioral suite ran on 4.7.2.
- Existing contract JSON goldens are byte-identical to baseline, including
  `node_get`, `scene_tree`, `scene_tree_ids`, and `capture_evidence`. Only the
  existing **help** golden changed, as required by item 10's new help entries.
  Four new opt-in read goldens were added. Both live-editor skills remain identical.
- Item 11 retained every pre-split guide sentence; only Markdown destination
  paths changed for relocation. Quick gate and verification text were copied
  exactly. The driver is 3,830 UTF-8 bytes with LF endings (under 4,500); the full
  guide is 19,937 bytes. README EN/KO and LOW_TOKEN describe the split accurately.

Local raw evidence remains under ignored `dist/wave1/`: `before-test.log`,
`item3`, `item4-final`, `item5` through `item9`, `item10-complete`, `item11`,
`latency-before`, `latency-after`, `live-item10-final`, `compat42`, and
`http42-baseline`. Those machine-local logs and engine binaries are not pushed.
The tests and this report are pushed.

## Item 9 status latency

Ten end-to-end CLI `status` processes per build, same machine and Godot 4.7.2,
separate minimized disposable editors with isolated homes. Timed with Python
`perf_counter`, including CLI startup, discovery, HTTP, and output collection.
Minimized is the measured condition; no claim that the editor actually used a
100 ms idle interval. Units: milliseconds.

| Run | Before | After |
|---:|---:|---:|
| 1 | 28.843 | 29.009 |
| 2 | 27.245 | 34.435 |
| 3 | 34.182 | 20.821 |
| 4 | 27.740 | 19.984 |
| 5 | 27.765 | 22.083 |
| 6 | 26.879 | 26.782 |
| 7 | 27.955 | 26.871 |
| 8 | 27.945 | 28.308 |
| 9 | 27.441 | 28.017 |
| 10 | 27.475 | 28.040 |
| Median | 27.7525 | 27.4440 |
| Mean | 28.3470 | 26.4350 |

The median improved by only 0.3085 ms; these ten runs do not establish a
meaningful general latency gain or the audit's proposed ~100 ms saving.
The deterministic regression proves first-chunk delivery without another server
poll; partial-write continuation and disconnect behavior remain covered.

## Findings, boundaries, and skipped work

1. Eval defaults to C#; both skill copies and the architecture table now say so.
   LOW_TOKEN's initial figures were labeled with the baseline revision; item 11
   measures the final split relative to the item-10 revision.
2. Docs now use returned set values, automatic evidence negotiation, status UI
   mode, and diagnostics first. `script validate` is preferred for disk scripts.
3. The audit's phrase “node not found” was generic: instance errors actually say
   `parent not found`, and signal errors distinguish the target. Exact strings
   were preserved rather than forcing a new common error.
4. The codecs were similar, not identical. Resource numeric coercion and parsed
   NodePaths, node object hints, runtime nested reads/type fields, and the two
   different truncation suffixes remain intact. Tests consume some line savings;
   this item removes a net 44 lines, not the predicted ~150. Tree walkers remain
   separate because their output differs. The repeated log hint is shared.
5. Capture freshness was indeed impossible in newly created staging directories.
   Documented counts and artifact/error checks remain. `capture_evidence.golden`
   covers screenshot provenance rather than Movie Maker freshness and is unchanged.
6. Equals-form parsing extends through evidence, QA, runtime PID, and sequence
   parsing too. The audit's `game input --kind=key` example is not a valid command;
   tests exercise actual key/button/axis commands. Boolean/global parser semantics
   were left alone; no global-parser refactor was introduced.
7. `engine info --project` really ignored the supplied directory. A regression
   now verifies resolution; explicit use reports project_path, while the default
   response stays unchanged. `script validate` could use the bounded runner without
   changing normal output/evidence contracts, so it was migrated. Typed eval errors,
   print error propagation, and dead DAP constant removal were completed.
8. OS key-name conversion was not used: its accepted spellings/canonical labels
   do not preserve all existing aliases. Dictionaries retain the existing contract
   and tests round-trip every entry. Official 4.2/4.7 docs and engine key-name
   sources were checked.
9. Optional runtime directory polling throttle was skipped; it was not necessary
   for the HTTP fix and would require its own runtime timing comparison.
10. Read flags are opt-in. Default comparison follows custom inspector revert,
    then Script defaults, then ClassDB defaults. 4.2 lacks the ClassDB query, so
    a temporary native class instance supplies native defaults and is freed.
    No user-script instance is created for comparison. Godot 4.2 can return null
    for uncached tool-script defaults, causing unchanged exports to be included;
    COMMANDS/CONTRACT document this limit. Tests cover available script defaults,
    custom reverts, and the engine-reported fallback. The 4.2 implementation's
    [default cache](https://github.com/godotengine/godot/blob/4.2-stable/modules/gdscript/gdscript.cpp#L363)
    explains why this cannot be treated as a reliable default-availability signal.
    Extra execution exposed item-4 unqualified static callable references that
    failed on 4.2; item 10 qualifies the shared callable and uses typed forwarding
    lambdas for resource/runtime callbacks. The float-format assertion now uses
    the engine's own string conversion (4.2 yields `0`, 4.7.2 yields `0.0`).
11. Full operating rules remain authoritative in AGENT_GUIDE; this is a driver-card
    split, not a deletion of safety or verification requirements.

The extra 4.2 HTTP test fails “environment token takes precedence”; the untouched
`532b5a3` server and test reproduce the same failure on this Windows binary.
It is not fixed in this wave. The changed immediate-response behavior did not
produce a separate failure. Headless editor teardown logs also contain RID and
ObjectDB leak diagnostics; passing regressions are not a warning-free claim.

No runtime polling throttle, combined commands, default response diet, debugger
migration, global parser redesign, package publication, or release work was done.
No full 4.2 behavioral-suite pass is claimed. Remaining review risks are the
4.2 script-default limitation and baseline 4.2 token-test failure above.
