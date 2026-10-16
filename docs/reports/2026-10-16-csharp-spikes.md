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
