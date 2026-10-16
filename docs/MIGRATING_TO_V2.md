# Moving to v2.0.0 / v2.0.0 전환

This describes v2.0.0 behavior. Upgrade CLI and addon together.
v2.0.0 동작 안내입니다. CLI와 애드온을 함께 업그레이드하세요.

## Select the eval language / eval 언어 선택

`eval` and omitted-language batch eval children now default to C#. Existing
GDScript expressions need an explicit language or the migration environment variable:
`eval`과 언어를 생략한 batch eval은 C#이 기본값입니다. 기존 GDScript 식은 다음처럼 실행하세요.

```sh
hera-godot eval --lang gdscript 'Vector2(1, 2)'
# Shell environment alternative (PowerShell: $env:HERA_GODOT_EVAL_LANG='gdscript'):
export HERA_GODOT_EVAL_LANG=gdscript
```

Explicit `--lang` (batch `params.lang`) wins over the environment. There is no
silent fallback: standard Godot returns `csharp_unavailable` and suggests GDScript.
명시한 언어가 환경변수보다 우선합니다. 표준 Godot에서 C# 요청은 자동 전환하지 않고 실패합니다.

For C#, use Godot 4.7+ .NET, build/reload the project with Hera's native `[Tool]`
loader, and install the external [compiler helper](CSHARP_SUPPORT.md#compiler-helper):
`hera-godot csharp-host install --version v2.0.0`. Package managers do not bundle it.
`SceneRoot` refers to the edited root. This example works in PowerShell, cmd.exe
and POSIX shells (cmd.exe does not strip single quotes):
`SceneRoot`는 편집 중인 루트입니다. 다음 예제는 PowerShell·cmd.exe·POSIX 셸에서
사용할 수 있습니다(cmd.exe는 작은따옴표를 제거하지 않습니다).

```text
hera-godot eval "((Node2D)SceneRoot).Position"
```

The string `result`, `type`, `undoable` and structured-value encoding retain their
types; successful C# responses additionally have `lang:"csharp"`.
C#은 4.7+ .NET, 빌드·리로드된 프로젝트와 별도 헬퍼가 필요합니다. 결과의 기존 필드 타입은 유지됩니다.

## Build and execution boundaries / 빌드·실행 경계

- `script validate res://Player.cs` explicitly restores/builds with isolated
  outputs; default 60 seconds, overridden by global `--timeout MS`. One root
  csproj, no ProjectReference. MSBuild targets/generators execute code.
- `script run res://Tool.cs -- a b` executes a prebuilt SceneTree/MainLoop
  harness directly, without a GDScript shim or implicit build.
- `.cs` debug breakpoints return `csharp_debug_unsupported` (exit 2); use an IDE.
- The editor UI compares the last observed CLI's major.minor version, not
  protocol compatibility or continued process liveness. Patch differences are
  neutral. A missing header is unknown: upgrade the older CLI. Copy setup contains no auth token/path.

검증 빌드는 격리 출력 경로를 사용하지만 샌드박스가 아닙니다. 실행은 이미 빌드된
C# 하니스를 사용하며, C# 디버깅은 IDE에서 수행합니다. UI 버전 일치는 호환성이나
CLI가 계속 실행 중이라는 보증이 아닙니다. 자세한 제한은 [C# 지원](CSHARP_SUPPORT.md)과
[보안](SECURITY.md)을 확인하세요. Godot 지원 하한 4.2는 유지됩니다.
