# Engine capability baseline — 2026-09-14

M0 evidence for the local Windows installation. Starting repository commit:
`c044cab` on `codex/editor-reliability-handoff`; the pre-existing untracked
implementation handoff document was preserved. Tests below cover that baseline
and the additive session/capability change in implementation commit
`bc2973bc74fbfcf3b569198cfaa3a09baeda6681`, not M1–M8.

## Executable identity

The inspected directory was
`C:\Users\PC\Desktop\Godot_v4.7.2-stable_mono_win64`.
Every `.exe`, `.dll`, and `.pdb` was enumerated without executing the inventory.
PE headers were read with the installed PowerShell/.NET binary reader; no tools
were installed. Paths below are relative to that directory.

| Executable | Bytes | SHA-256 |
|---|---:|---|
| `Godot_v4.7.2-stable_mono_win64.exe` | 181449736 | `45336315eb6f1a52a8923bc4f2ce8079a03dc4939dcb7d531047890f1f7cdfab` |
| `Godot_v4.7.2-stable_mono_win64_console.exe` | 198152 | `2445d009a5e0474fc7064b9767100e9d2c09521890ac5625476bfb81cf03f2d4` |

Both have file version `4.7.2`, product version
`4.7.2.stable.mono.official`, PE machine `0x8664` (x64), and no CLR header.
The large executable was selected as the direct engine candidate; its bounded
invocation then confirmed `OS.get_executable_path()` and the launched PID.
The small console executable was never run.

Nearby evidence comprises 50 DLLs with PE machine `0x014c` and a CLR header,
plus 13 PDBs, all under `GodotSharp/`. This establishes managed assemblies;
the PE machine field alone does not establish a managed assembly's runtime
architecture. Debug and Release API directories each contain
`GodotSharp.dll`, `GodotSharpEditor.dll`, and `GodotPlugins.dll` with matching
PDBs. The remaining DLLs/PDBs are tools and localized Roslyn resources.
There is no root native-engine PDB in this inventory. Exact paths, sizes,
versions, and SHA-256 values for all 65 files are retained only in private
`binary-inventory.json` evidence.

The direct engine's `--version` and `--help` each exited 0 within 15 seconds.
It reports `4.7.2.stable.mono.official.ed1daf0bf`. The isolated engine's
`Engine.get_version_info()` reports the full commit
`ed1daf0bf001b61586d9930840f2f1394092c079`, version
`4.7.2-stable (official)`, and build `official`. Fresh Hera status agrees.
Observed help includes `--headless`, `--editor`, `--path`, `--script`,
`--check-only`, `--import`, `--log-file`, and `--dap-port`.

## Capability evidence and limits

`supported` means the specifically named API exists in the selected running
engine. `unsupported` means its required API check is false. `unverified` means
the addon has not established the broader behavior. No state comes from a
version comparison, and API presence does not assert that Hera implements the
feature or has an active connection.

| Status key | Local state | Direct evidence / remaining boundary |
|---|---|---|
| `editor_logger_api` | supported | `Logger`, `OS.add_logger`, and `OS.remove_logger` exist. An isolated editor registered a temporary logger, received one marker through `_log_message`, and removed it. Hera does not yet collect editor logs. |
| `debugger_messages_api` | supported | `EditorPlugin.add_debugger_plugin`, `EditorDebuggerSession.send_message`, `EngineDebugger.register_message_capture`, and `EngineDebugger.send_message` exist. A runtime/debugger handshake was not tested. |
| `dap` | unverified | Help advertises `--dap-port`; no DAP initialization/connection was performed. A version or CLI option is insufficient to claim a usable DAP session. |
| `script_metadata_api` | supported | `Script.get_script_method_list` exists and returned a temporary script's `marker` method. |
| `script_symbol_lookup` | unverified | A direct `ScriptLanguage.lookup_code` bound-method check returned false. This does not establish that all possible symbol-resolution paths are absent; none is implemented or certified here. |
| `global_class_lookup_api` | supported | `ProjectSettings.get_global_class_list` returned the imported temporary `HeraM0GlobalProbe` class. |
| `import_state_api` | supported | `EditorFileSystem.is_scanning` and `get_scanning_progress` exist and returned `false` and `1.0` after isolated import. This is not a per-resource import-success guarantee. |
| `render_frame_event_api` | supported | `RenderingServer.frame_post_draw` signal exists. No rendered-frame completion or visual capture is claimed for the headless run. |
| `csharp` | supported | The running engine exposes `CSharpScript`; its boolean `csharp_supported` remains unchanged. |
| `dotnet_sdk` | unverified | Status does not launch SDK checks. Separately, `dotnet --list-sdks` found 7.0.101 and 10.0.204; restore, build, and loaded-assembly readiness were not tested. |

The matching engine's [Logger API](https://github.com/godotengine/godot/blob/ed1daf0bf001b61586d9930840f2f1394092c079/doc/classes/Logger.xml)
defines registration and callback signatures, including callbacks from multiple
threads. The disposable probe used a mutex and logged nothing from its callback.
The [script-language extension API](https://github.com/godotengine/godot/blob/ed1daf0bf001b61586d9930840f2f1394092c079/doc/classes/ScriptLanguageExtension.xml)
lists `_lookup_code` as a virtual extension hook; that is not proof of a callable
symbol-lookup service for the built-in language.

## Added output

Status adds `godot_commit`, `editor_session_id`, and the ten-key `capabilities`
map above. A 16-byte random session is created once per status-tool/addon
lifetime, retained across requests, and shared with its heartbeat. A restarted
addon gets a new identity independent of its PID. `instances` preserves the
optional field; legacy heartbeats remain readable and omit it on output.
Existing status fields and compact default JSON remain intact. This is evidence
only, with no session mutation guard, capability negotiation, or logger collector.

## Verification

Commands used the direct executable above. Godot processes were bounded and
used disposable copies with separate `HOME`, `USERPROFILE`, and `APPDATA`;
regression fixtures also isolated XDG directories. The engine's reported
`user://` resolved inside its fixture. No user editor/game was terminated.

| Check | Result |
|---|---|
| Initial `go test ./...`, `go build ./...`, `go vet ./...` | Pass |
| Initial `GODOT_BIN=<direct-exe> bash tests/headless/run.sh` | Pass, 16 regressions |
| New discovery session test before Go change | Failed as intended: session field dropped |
| New status/heartbeat regression before addon change | Failed as intended: session/commit/capability fields absent |
| Updated status/help contract expectations before help change/regeneration | Failed as intended; goldens then regenerated |
| Final `go test ./...`, `go build ./...`, `go vet ./...` | Pass |
| `go test -shuffle=on -count=1 ./...` | Pass |
| `go test -race -shuffle=on -count=1 ./...` | Pass; Go 1.26.2 windows/amd64, CGO enabled, installed UCRT64 GCC |
| `gofmt -l .`, `git diff --check` | Clean |
| Direct `--headless --path <copy> --check-only --script <file>` | Pass, all 52 addon scripts and the new status regression script |
| Final isolated standalone regressions | Pass, 17 including status/session/capabilities |
| Fresh isolated `status`, `instances` | Pass; engine identity and heartbeat/status session agree |
| Isolated `smoke --skip-game` | Pass, 3 steps (status, diagnostics, scene) |
| Isolated `diagnostics --lines 20`, before and after smoke | Command succeeds but log is unavailable: `available:false`, `clean:false` |

There was no fresh user-owned editor: discovery reported four expired
heartbeats, which were not removed. The isolated editor's separately captured
stderr was empty. Diagnostics' unavailable result is not a clean-project
certificate; it still reads the project log, not the explicit editor log path.
Several standalone editor scripts emitted scan-aborted/RID/ObjectDB teardown
messages, including before the change and in a no-addon probe. They exited 0
without script/parse errors. The runner's pass is not a warning-free claim.

Only the supplied 4.7.2 .NET engine was run. This does not refresh the 4.2–4.6
matrix, verify standard-build rejection live, compile C#, test GUI behavior,
exercise runtime debugger traffic, or validate DAP. Existing CI configuration
was inspected and left unchanged.

## Evidence and cleanup

Raw inventory, bounded help/version output, probe output, test logs, live
status/heartbeat, and smoke logs are outside the repository under the private
temporary evidence directory `hera-m0-20260914`. No API dump, token, or raw
user project output is committed. Task-owned editor processes were stopped and
no Godot processes remained at the cleanup check. Successful regression runs
cleaned their own fixtures. Automatic execution policy rejected recursive
cleanup of the remaining task-owned project copies, both with validated
temporary-directory bounds and with explicit literal paths. No more specific
reason was returned. `%TEMP%/hera-m0-20260914/live`,
`%TEMP%/hera-m0-20260914/probe`, and the expected-failure fixture
`%TEMP%/hera-regressions.kUqeXU` therefore remain, including their generated
files. No smoke artifacts were written into the working tree. The user's
untracked handoff and expired heartbeat files remain unchanged.
