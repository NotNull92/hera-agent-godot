# v2.0.0 release gates — 2026-10-16

Implementation candidate: `87368e3` on main. Victor authorized direct main
release work and accepted unknown for legacy CLI requests without a version
header. That row now asks the user to upgrade the CLI. No user project was used.

## Local verification

- Go build, vet, `test -race -shuffle=on -count=1 ./...`: PASS; tracked Go files
  produce no gofmt output. Contract goldens regenerated; only the new installer
  help line changed. Existing stable fields were not changed.
- 59 addon scripts: check-only PASS on standard Godot 4.2 and 4.7; panel,
  HTTP, eval-result and plugin-load tests PASS on both. Temporary evidence:
  `hera-w2-check-30brxa3b`, `hera-w2-check-bd1v9o55` under Windows TEMP.
- `run.sh`: `PASS: 31 standalone Godot regressions` (4.7). Evidence:
  `hera-v2-release-regressions.log`, fixture `hera-regressions.3jed2v`.
  Intentional negative cases and engine teardown warnings remain in those logs.
- .NET 4.7.2 language runner: `PASS: GDScript, C# creation, assembly metadata,
  attachment and runtime QaReady`; `hera-v2-release-languages-85je6ukp/run.log`.
- Exact release publish command on Windows, SDK 10.0.204: PASS:
  `dotnet publish csharp-host/Hera.CSharpHost.csproj -c Release -f net8.0 --self-contained false -p:UseAppHost=false -o dist/csharp-host`.
- `host_checks.py` on that output: expression/void compilation, syntax/bounds
  rejection and MVID mismatch PASS.
- `w1_live.py --host-archive dist/hera-csharp-host.zip`: PASS for C#
  eval/validate/run, explicit GDScript, mixed batch, placeholder and same-editor
  reload. Evidence: `hera-csharp-w1-i6ttxx2r`. A loopback HTTP release fixture
  serves the freshly published ZIP/checksum before the public release exists;
  the test build overrides only the download base. The actual `csharp-host
  install --version v2.0.0` command installs into the isolated home. Eval uses
  normal home discovery, without `--csharp-host` or its environment override.
- npm scratch pack/install/aliases and unrelated `hera` preservation: PASS.

The W1 startup probe now waits for the requested startup scene via read-only
status. A heartbeat can appear before import finishes; the previous immediate
scene-open attempt correctly returned `editor_busy` and was not a compiler defect.
The initial version-only push had a stale panel patch-version test; `29004c2`
updates its injected patch to the new addon major/minor and both local engines pass.

## Installer and distribution boundary

Tests use a local HTTP fixture and verify valid replacement, bad/missing checksum,
unsafe ZIP traversal and missing required files. Failures preserve an existing
installation. Downloads and expansion are bounded; archive links are refused.
SHA256 is checked against the same release's checksum file, not a separate signature.
The download command is explicit; eval never downloads a helper.

`actions/setup-dotnet` pins commit `a98b56852c35b8e3190ac28c8c2271da59106c68`,
release v6.0.0 published 2026-07-16 (GitHub API verified), older than two weeks.
The helper is framework-dependent net8.0 with LatestMajor roll-forward, no
platform apphost. Native macOS and .NET 8-only execution were not reproduced here.

`npm whoami` returned E401. Publishing requires the owner's authenticated npm
session/2FA; no npm login, credential change or publish retry was attempted.
Final release assets, CI and channel state are recorded in the v2 release record
and local result handoff after publication.
