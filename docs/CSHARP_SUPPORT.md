# GDScript and C# scripts

Choose the language by `.gd` or `.cs`;
`script create --lang gdscript|csharp` must agree with that extension. Keep
the existing GDScript script-file behavior and editor bridge. `eval` separately
defaults to C#, regardless of the project language. Only explicit C# validation
invokes a build, with isolated output paths; create/open/attach never build.

## Use

```sh
hera-godot status
hera-godot script create res://Player.cs --lang csharp --extends Node2D --ready --export Speed:float=3.5f
hera-godot script create res://helper.gd --lang gdscript --ready
```

`status.csharp_supported` reports whether the connected editor has C# support;
it does not check for the .NET SDK. C# creation/opening/attachment require the
Godot .NET editor. Initialize the C# solution using Godot's C# tooling if the
project does not have one yet, and install the SDK required by that Godot
version. Hera never generates or modifies csproj/solution files.

Build using Godot's Build action or `dotnet build` in the project directory.
Reload the assembly (restart the editor if needed), then:

```sh
hera-godot script inspect res://Player.cs
hera-godot node attach-script . res://Player.cs
hera-godot scene save
hera-godot run --current --wait
hera-godot game qa discover
```

The C# class name is the filename without `.cs`. If supplied, `--class-name`
must agree. Lifecycle, `--tool`, signal, and export flags generate C# syntax;
use C# types and expressions for exports, such as `Speed:float=3.5f`.
Type/expression validity and Godot export compatibility are checked by the C#
compiler, not by Hera's template generator. Creation reports
`build_required: true`; attachment rejects classes not yet loaded.

## Inspection and runtime QA

C# inspection uses Godot's loaded assembly metadata. Before an assembly is
loaded, method/signal/export lists are unavailable, rather than inferred by
an incomplete C# source parser. Even loaded metadata may be stale after a
source edit; rebuild and reload the assembly before relying on it.

`assembly_loaded` means the engine has a native base type for the script; it
does not prove that source changes have been compiled. `class_name` is derived
from the filename, and `extends`/`base_type` describe the native Godot base.
See [COMMANDS](COMMANDS.md#script-languages) for response fields. Standard
Godot can inspect a `.cs` file's path/line count but has no assembly metadata.
`script current` reports Godot's focused script, not an external IDE's tab.

C# custom method names keep their case. Public QA helpers such as
`public bool QaReady() => true;` are discovered alongside GDScript `qa_*`
methods. Call them using `hera-godot game node call /root/Main QaReady`.
With the v1.2.0 addon, a true return includes:
`{"method":"QaReady","path":"/root/Main","result":"true","result_value":true,"type":"bool"}`.
The stable text `result` is unchanged; read experimental `result_value` directly.
`Qa` must be followed by an uppercase ASCII letter. `eval --lang gdscript`
selects Godot's GDScript expression evaluator; the new default is C#.

## Verification (2026-09-07)

The following is historical evidence, not proof of the new eval path. Current
branch evidence is in [W1](reports/2026-10-16-csharp-w1.md) and the W2 report.

- Go build, vet, and `go test -race -shuffle=on -count=1 ./...` passed;
  `gofmt` and contract goldens were refreshed.
- Godot 4.2 standard and 4.7.2 standard/.NET: affected GDScript files passed
  `--headless --check-only --script`; both headless language/template tests passed.
- Isolated live standard 4.7.2 editor: GDScript create/inspect/open/current
  passed; C# creation/opening/attachment rejected and inspection reported no
  assembly metadata.
- Isolated live .NET 4.7.2 editor with SDK 10.0.201: generated lifecycle,
  signal, and export C# code compiled with zero warnings/errors. Unbuilt
  attachment rejected; built metadata and attachment succeeded; runtime
  `QaReady` discovery and call returned `true`. A GUI play session also
  verified `Speed == 3.5` and an empty runtime error log.
- First failing checks demonstrated the missing `--lang`, `.cs` path support,
  C# template, and `QaReady` discovery before implementation.
- Independent code review found no blocking correctness or regression issues.

Reproduce the live checks with the built CLI and the direct Godot executable:

```sh
go build -o hera .
GODOT_BIN=/absolute/path/to/Godot HERA_BIN="$PWD/hera" GODOT_SDK_VERSION=4.7.2 bash tests/headless/script_languages.sh
```

Use a standard Godot executable to exercise unsupported-C# behavior (no
`GODOT_SDK_VERSION` or dotnet required). The .NET run also needs `dotnet` and
`jq`; `DOTNET_TFM` may override the installed SDK's major-version target, and
`NUGET_CONFIG` may point to an offline feed configuration. `KEEP=1` preserves
test evidence. If editor-spawned games cannot start headlessly, use
`GODOT_EDITOR_GUI=1` on a desktop/virtual display.

The smoke uses a unique temporary project, canonical paths, and explicit
editor PIDs. macOS symlinked project paths can generate incorrect C# script
path attributes; the runner resolves them before building. C# runtime support
on earlier Godot .NET versions was not exercised in this change.

Engine references:

- [C# setup, script naming, and rebuilding](https://github.com/godotengine/godot-docs/blob/master/tutorials/scripting/c_sharp/c_sharp_basics.rst)
- [CSharpScript metadata implementation](https://github.com/godotengine/godot/blob/master/modules/mono/csharp_script.cpp)

## Install the compiler helper / 컴파일러 헬퍼 설치

```sh
hera-godot csharp-host install --version v2.0.0
```

Release builds default to their own tag; development builds require `--version`.
The explicit command downloads `hera-csharp-host.zip` and `checksums.txt` from
GitHub, verifies SHA256, and replaces `~/.hera-agent-godot/csharp/` after staging.
Existing helper files are replaced as a directory; close active helper processes
before upgrading. Failures before replacement preserve the old installation.
It needs no live editor and does not change project files. Default download
budget is 120 seconds; global `--timeout MS` overrides it. Success is compact
JSON (`installed`, `version`, `path`); usage errors exit 2, installation failures
exit 1 with `csharp_host_install_failed` on stderr. Checksums provide integrity
against the downloaded release list, not an independent signature.

npm, Scoop and Homebrew install only the CLI. Install the helper separately;
`eval` never downloads it automatically. .NET 8 or a compatible newer runtime
is required to execute the framework-dependent helper.

패키지 매니저는 CLI만 설치합니다. 위 명령으로 헬퍼를 별도 설치하세요. SHA256을
검증한 후 사용자 홈의 헬퍼 폴더를 교체하며 사용자 프로젝트는 변경하지 않습니다.
eval은 자동 다운로드하지 않습니다. 헬퍼 실행에는 .NET 8 이상 호환 런타임이 필요합니다.

## Compiler helper

New C# eval/validate/run require Godot 4.7+ .NET; the general addon floor remains
4.2. Build the CLI and the external Roslyn helper from matching source:

```sh
go build -o hera-godot .
dotnet build csharp-host/Hera.CSharpHost.csproj -c Release -o dist/csharp
hera-godot --csharp-host dist/csharp/Hera.CSharpHost.dll eval '1+1'
```

Keep the entire helper output directory, including dependencies and runtimeconfig.
It targets .NET 8 with roll-forward to newer major runtimes. Only .NET 10 execution
has been verified locally. Resolve order: `--csharp-host`, `HERA_GODOT_CSHARP_HOST`,
`csharp/Hera.CSharpHost.dll` beside the CLI, then
`~/.hera-agent-godot/csharp/Hera.CSharpHost.dll`. Only `csharp-host install` downloads the helper.
The release includes the framework-dependent helper ZIP. Roslyn is not added to user projects.
The native addon loader is entirely `#if TOOLS`, with nullable enabled locally;
export builds exclude its types. Build/reload the project after installing it.

새 C# 기능은 4.7+ .NET 전용이며 애드온 지원 하한 4.2는 유지됩니다. 헬퍼 출력
폴더 전체를 보관하세요. 사용자 csproj에 Roslyn을 추가하거나 솔루션을 생성하지
않습니다. 로더는 에디터용으로만 컴파일되며 설치 후 프로젝트 빌드·리로드가 필요합니다.

## Eval and validation limits / 평가·검증 제한

- C# `eval` executes one expression with `SceneRoot` and Godot/System imports.
  Non-Tool C# scene scripts are editor placeholders: native Node API access works,
  but casting to the user's C# class can fail. This is editor eval, not game eval.
- The disk project DLL's module ID is checked against the loaded assembly before
  compilation and execution. This is not proof of dependency freshness or source
  equivalence. Debug output paths must be the standard Godot SDK layout.
- The collectible ALC is unloaded and checked. Retained delegates/handlers can
  prevent unloading (`csharp_unload_failed`) and require an editor restart.
  Failure does not undo already executed effects. Arbitrary expressions may block
  the editor; an HTTP timeout does not forcibly stop them. There is no sandbox.
- Variant-compatible results reuse the existing structured serializer. Unsupported
  CLR types fail; Godot container/size/depth limits can omit `result_value`.
- C# validation restores and builds exactly one root csproj, refusing
  ProjectReference graphs and sources excluded from Compile items. The default
  combined SDK/query/restore/build deadline is 60 seconds; HTTP context selection
  retains its independent 5-second default. Global `--timeout MS` overrides both.
  Build/intermediate outputs are redirected; custom targets may still write
  elsewhere. It does not observe unsaved buffers, update editor assemblies or
  certify warning-free code. Requested-file diagnostics and other-file errors
  are distinguished. A changed source or truncated output fails verification.
- `script run` requires the existing Debug DLL and a SceneTree/MainLoop C# harness;
  it invokes `.cs` directly and never builds. The built revision may be stale.

에디터 placeholder는 사용자 C# 클래스가 아닐 수 있습니다. eval은 undo·샌드박스·
강제 중단을 제공하지 않으며 ALC 언로드 실패 후 에디터 재시작이 필요할 수 있습니다.
검증은 루트 csproj 하나와 ProjectReference 없는 프로젝트만 지원하며, 출력 경로
격리는 사용자 MSBuild 타깃의 임의 파일 쓰기를 막지 않습니다. 실행은 빌드된 리비전을
사용하므로 최신 소스인지 별도로 확인하세요.

## C# debugging / C# 디버깅

`debug attach --break res://Player.cs:12` returns `csharp_debug_unsupported`,
exit 2, and IDE guidance before reading the project or connecting to DAP.
Hera's Godot DAP commands do not implement .NET debugging. Follow the official
[Godot C# external-editor setup](https://docs.godotengine.org/en/stable/tutorials/scripting/c_sharp/c_sharp_basics.html#configuring-an-external-editor):

- Rider: select Rider in Godot's external-editor settings and enable its Godot
  support (built in since Rider 2024.2).
- Visual Studio: open the generated solution, create an Executable debug profile
  pointing to the main Godot .NET executable, and set the working directory to `.`.
- VS Code: install C# tooling, configure `tasks.json`/`launch.json` per the official
  guide, then launch or attach the .NET debugger. Hera creates no IDE settings.

C# 중단점은 Rider·Visual Studio·VS Code의 .NET 디버거에서 설정하세요. 실행 중인
Godot 게임 프로세스에 연결하거나 IDE의 Godot 실행 구성을 사용합니다. Hera는
IDE 설정을 만들거나 사용자 프로젝트를 수정하지 않습니다.
