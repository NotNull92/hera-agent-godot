# C# W1 Phase 0 spikes

Base: `5b97973b450690eddcaa37261174b1d4d51c66ab`; branch `feat/v2-csharp`.
Victor authorized Phases 0–5, small commits and branch pushes, no PR/merge/tag,
version changes or Phase 7 documentation renewal. K1–K4 and D-UI1–3 accepted.
Production C# execution must use a native `[Tool]` C# loader, never a GDScript
shim. User projects/build files must not be edited. All projects below are
disposable fixtures created by the probes, with isolated user/config paths.

## Results

Windows x64, Godot `4.7.2.stable.mono.official.ed1daf0bf`, SDK `10.0.204`,
fixture target `net10.0`. This is not evidence for other SDKs or macOS.

| Spike | Result | Observed evidence |
|---|---|---|
| S1 | PASS for the basic Approach A lifecycle | C# EditorPlugin loads separately compiled expression bytes in a collectible ALC, shares Godot/project references, invokes an engine Node2D API and returns Vector2 via Variant. WeakReference becomes dead after GC. Rebuild changes project stamp 1→2; the same editor reloads and executes stamp 2, then repeats eval/unload. Editor exits 0. |
| S2 | PASS with explicit paths | `dotnet build` with BaseOutputPath, OutputPath, BaseIntermediateOutputPath, IntermediateOutputPath and MSBuildProjectExtensionsPath redirected produces an assembly outside the project. Existing `.godot/mono/temp` file SHA256 and mtime snapshots are identical before/after. |
| S3 | PASS; no shim needed | Built C# `Tool : SceneTree`, invoked `--headless --script res://Tool.cs -- one two`: `_Initialize` prints `HERA_CSHARP_DIRECT_RUN_OK:one,two`, exit 0. |
| S4 | PASS within stated export scope | Standard 4.7 editor loads the Hera addon with the C# probe source present; bridge listening, exit 0, no script errors. .NET Windows PCK export runs publish successfully. Metadata inspection of the export-compiled ExportRelease assembly finds control type ProbeStamp but no ProbePlugin/EvaluationContext. This checks `#if TOOLS`; no exported executable was launched. |
| S5 | PASS | Official 4.2-stable: theme probe check-only and editor execution pass; `get_editor_theme`, positive editor scale and Control theme-color lookup work. No UI redesign is implemented. |

Reproducers: `tests/spikes/csharp/s1.py`, `s2_s3.py`, `s4_export.py`,
`s4_s5.py`; all take explicit engine paths. These are opt-in local spikes,
not new CI jobs. The C# fixture is entirely under `#if TOOLS`.

Final local evidence:

- S1–S3 and S4 export: `%TEMP%/hera-csharp-s1-qp7lqted`.
- S4 standard/S5: `%TEMP%/hera-csharp-s4s5-2pswcdg7`.
- Official floor binary: `%TEMP%/hera-godot42-nb7wa1ps` from the
  godotengine/godot-builds `4.2-stable` Windows x64 release.

## Discoveries and limits

- First S1 observer incorrectly expected `_EnterTree` to repeat on reload.
  Godot had unloaded successfully; observing the new `_Process` stamp corrected
  the harness without removing the reload/eval assertions. Orphan test Node2Ds
  needed `Free`, not merely Dispose; final S1 log has no leak warnings.
- Base output properties alone do **not** isolate Godot SDK builds: Sdk.props
  explicitly sets OutputPath and defaults IntermediateOutputPath. The first
  disposable attempt changed editor outputs. Both concrete paths must also be
  overridden. Custom MSBuild targets remain arbitrary code, not sandboxed IO.
- Export requires a solution in this fixture. Initial export without it emitted
  errors despite exit 0. With a fixture-only solution, the offline feed then
  lacked runtime packs. Adding official NuGet to the fixture allowed publishing.
  Final export log is checked for errors as well as exit status. No user
  solution, NuGet configuration or editor project was changed.
- `dotnet --info` failed in workload inventory on this host; version and actual
  builds succeeded. Probes disable workload resolution. The first SDK startup
  reported creating an HTTPS development certificate; subsequent runs explicitly
  disable certificate generation. No certificate trust command was run.
- S1 uses a temporary Node2D, not an actual edited-scene user class. Non-Tool
  placeholder casts, all result shapes, compiler failures, repeated stress and
  final loader integration remain Phase 4 acceptance work, not proven here.
- This does not prove arbitrary user code releases references. Timers, event
  handlers or retained delegates can keep an evaluation ALC alive; production
  must report unload failures rather than claiming unconditional unload.
- .NET 8 helper compatibility and macOS live checks are not established.

Upstream lifecycle reference examined:
[HotReloadAssemblyWatcher](https://github.com/godotengine/godot/blob/master/modules/mono/editor/GodotTools/GodotTools/HotReloadAssemblyWatcher.cs).
The installed Godot.NET.Sdk 4.7.2 Sdk.props/Sdk.targets supplied the output-path
and Debug-only TOOLS definitions; local execution above is the decisive evidence.

Phase 0 basic feasibility is established. Phases 1–5 remain to implement and
validate; these spike results are not a completed W1 claim.

## Phase 1 — protocol and observations

CLI requests now include the actual CLI version header (development fallback).
The server remembers a bounded version and timestamp only for accepted requests;
missing/oversized headers remain unknown. Experimental C# capability keys are
present without claiming unimplemented functionality. Only explicit status
capability reads probe the local SDK with a bounded `dotnet --version`; SDK
presence never implies editor C# support. Stable status fields are unchanged.

Validation: Go build/vet, full uncached tests and targeted golden pass. Updated
HTTP and capability tests pass on official Windows 4.2 and 4.7; HTTP also passes
on 4.7.2 .NET. Initial harness inherited auth and imported unrelated 4.7-only
tests into 4.2; corrected isolation (empty auth, minimal test copy, proper editor
mode) passes without assertion changes. 4.2 editor teardown emits known RID
warnings, not a zero-warning claim. Evidence: `%TEMP%/hera-v2-protocol-_sq9i68t`.

## Phase 2 — C# disk validation

C# validate now requests a guarded context from a 4.7+ .NET editor, queries
MSBuild Compile items, restores/builds into isolated paths and reports file
error/warning diagnostics plus other-file error counts. Requested-file hashes
bracket compilation. Missing SDK/project, excluded source, restore/build errors,
truncation and deadline failures are failures; unfinished workers retain staging.
MSBuild targets/generators execute code; redirection is not a security sandbox.

Current conservative boundary: exactly one root csproj and no ProjectReference
items. References are explicitly refused as `project_references_unsupported`,
rather than allowing different projects to share concrete output/intermediate
paths. Supporting project graphs needs per-project isolated paths. No user
csproj/solution is generated or edited. Phase 7 must document this boundary and
the changed build/security wording before release.

Validation: real SDK/Godot.NET.Sdk compilation for clean, requested-file error,
and other-file-only error passes; each preserves all baseline editor artifact
hashes and mtimes. The Go live test uses mock editor discovery/context plus real
.NET compilation in a disposable project, not a claim of full live HTTP coverage.
Go build/vet/full uncached tests pass. Updated script/status GDScript check-only
and validation-guard regression pass on 4.2 and 4.7.2 .NET. Full live integration
will be repeated with the final loader in Phase 4.

## Phase 3 — direct C# harness execution

`script run` now preflights C# paths, known incompatible base classes, Godot
4.7+ .NET identity and the existing Debug project assembly. It invokes the .cs
file directly; no GDScript shim and no implicit build. Unknown custom bases
remain the engine's responsibility, as with GDScript. Assembly presence does
not prove freshness relative to source; the engine executes the built revision.

Go unit/full tests, build and vet pass. Actual development CLI ran the S3
fixture `Tool.cs` with `one two`: expected marker, exit 0, duration 317 ms.
This run used the disposable project but inherited the invoking user's profile
(unlike the isolated Phase 0 runs); no user project was opened or changed.
All subsequent integration uses isolated profiles explicitly.

## Phase 4 — native C# eval

The CLI defaults eval (including batch children) to C#. Explicit `--lang`
overrides `HERA_GODOT_EVAL_LANG`; `gdscript` retains its original result fields.
The external net8 Roslyn helper compiles one expression against platform,
Godot and project references. `--csharp-host` / `HERA_GODOT_CSHARP_HOST` selects
an installed helper; adjacent and private-home locations are also searched.
Packaging/downloading the helper belongs to the later release work, not W1.

The native `[Tool]` loader is entirely `#if TOOLS`. It executes on the edited
scene through a collectible ALC and checks unloading. Godot loads the project
from bytes, so `Assembly.Location` is empty: the Debug DLL is resolved separately,
and the helper verifies the loaded module's MVID against the exact reference
bytes it compiles with. The loader checks MVID again before execution. A rebuild
between observations fails rather than silently evaluating against a different
project revision. External dependency freshness is not independently certified.

Results use the existing Variant serializer; unsupported CLR result types fail
instead of being stringified as successful values. Expressions and compiler
output are bounded. The editor expression itself is arbitrary synchronous code,
not sandboxed or forcibly interruptible by an HTTP timeout. It can mutate state,
retain delegates or register handlers; unload failure reports effects may have
occurred. A retained ALC can still require an editor restart. This must be carried
into Phase 7 security and migration documentation before release.

Validation:

- `host_checks.py`: scalar/member/void expressions compile, malformed statements,
  oversized/empty expressions and mismatched MVID fail. Helper build: zero
  warnings/errors. Tests ran using installed .NET 10 roll-forward; .NET 8-only
  runtime execution remains unverified.
- `w1_live.py`: actual CLI → HTTP → native loader on 4.7.2 .NET passes scalar,
  Vector2, dictionary, NaN, large integer, void mutation/readback, parse failure,
  mixed-language batch, env fallback, real C# disk validation and direct .cs run.
  A non-Tool C# scene is a Node2D placeholder: `SceneRoot is Probe` is false and
  casting fails honestly. Same-editor rebuild changes `Stamp.Value` from 1 to 2;
  engine log confirms successful project ALC unload. Editor quits with exit 0;
  the quit request itself loses its HTTP reply, as explicitly asserted.
- Evidence: `%TEMP%/hera-csharp-w1-7tyolj9c`. `s4_export.py` on this fixture
  performs a real PCK export and inspects ExportRelease metadata: project control
  type remains, HeraCSharpEval/EvaluationContext types are absent.
- Go parser/env/batch/host tests, error/help goldens, full uncached tests,
  build and vet pass. No stable result fields or version numbers changed.

The first integration failed because Assembly.Location was empty, then cleanup
used an invalid EditorInterface.GetTree call and finally assumed a quit reply;
all three are corrected in the reproducible harness. Standard-editor checks use
separate projects per engine and only the relevant test, avoiding 4.7-import
cache reuse and unrelated newer test APIs on 4.2.

## Phase 5 — C# debugging guidance

`debug attach --break res://Player.cs:12` now returns usage exit 2 and
`csharp_debug_unsupported` before reading the project, hashing scripts,
connecting to DAP or writing a session. Mixed breakpoint lists also refuse the
whole request when any path ends in `.cs` (case-insensitive). GDScript DAP paths
are unchanged. This does not add a .NET debugger or detect a project's language
when attach contains no C# breakpoint.

For C# debugging / C# 디버깅 안내: use Visual Studio, Rider, or VS Code with a
.NET debugger, following the official
[external-editor setup](https://docs.godotengine.org/en/stable/tutorials/scripting/c_sharp/c_sharp_basics.html#configuring-an-external-editor).
Hera's Godot DAP endpoint is not the .NET debugging endpoint. Hera does not
generate IDE settings, launch a debugger, or change project files for this.
C# 중단점은 IDE의 .NET 디버거에서 설정하고 실행하세요. Hera의 `debug` 명령은
C# 디버깅을 대신하지 않으며 IDE 설정이나 사용자 프로젝트를 수정하지 않습니다.

Unit tests verify both `.cs` and `.CS` reject even with a nonexistent project
and unreachable endpoint. A contract golden pins error, guidance URL and exit.
The documentation section is recorded here under the user's Phase 0–5 scope;
the Phase 7 CSHARP_SUPPORT/COMMANDS/SECURITY/README migration remains deferred.

## W1 local gate summary and handover

- Go build, vet, full uncached tests, shuffled uncached tests and race + shuffle
  pass on Windows. C# compiler-helper build and assertion suite pass. Live
  4.7.2 .NET execution and real export verification pass as detailed above.
- Standard 4.7 eval check-only, explicit GDScript regression, default C# refusal
  and plugin load pass. 4.2 eval/tool-test check-only and the same regression
  pass (`hera-w1-standard-rbwmzuof`). Full 4.2 import/plugin-load gate is **not
  established**: startup without a completed import misses the addon font;
  a clean isolated import exceeds 60 seconds. The same clean import on the
  untouched base `5b97973` also exceeds 60 seconds, captured in
  `%TEMP%/hera-42-baseline-zd27i338/import.log`. Current evidence is
  `%TEMP%/hera-w1-standard-8s7xsind/0-import.log`. This is not reported as a
  passing 4.2 full editor gate or attributed to the C# change.
- The helper assertion suite is a Python subprocess test (`host_checks.py`),
  not a `dotnet test` project. Live W1 coverage is a dedicated disposable-project
  Python runner rather than an expansion of `script_languages.sh`. Successful
  eval/validate outputs are recorded in live evidence; committed contract
  goldens currently cover language errors, GDScript compatibility, help and
  C# debug refusal, not live C# success/validation output normalization.
- No macOS live execution, .NET 8-only host execution, full Phase 8 headless
  suite or new CI result is claimed. No release readiness claim: Phase 6–8,
  Phase 7 documentation/security migration and helper packaging remain later
  work. General v1 documentation intentionally still describes the released
  behavior, per the user's explicit scope restriction.
- ProjectReference validation remains conservatively unsupported. C# eval
  still cannot cast non-Tool editor placeholders to user classes, and an
  arbitrary expression can retain its ALC or block the editor. Local gates do
  not erase those boundaries. Existing `csharp_eval: unverified` capability
  does not claim the helper/project has been built merely from .NET presence.

User projects, csproj/solution files, main, tags, versions and CI configuration
were not changed. Only owned temporary fixtures were built/exported; launched
editors were shut down or killed by their owning harness. Preserve the user's
untracked PLAN/REVIEW files and v1 HTML explainer. Stop at Phase 5; no PR opened.
