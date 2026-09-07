# GDScript and C# scripts

Choose the language by `.gd` or `.cs`;
`script create --lang gdscript|csharp` must agree with that extension. Keep
the existing GDScript behavior and the GDScript editor bridge. Do not infer
a default language from project files or invoke a build automatically.

## Use

```sh
hera status
hera script create res://Player.cs --lang csharp --extends Node2D --ready --export Speed:float=3.5f
hera script create res://helper.gd --lang gdscript --ready
```

`status.csharp_supported` reports whether the connected editor has C# support;
it does not check for the .NET SDK. C# creation/opening/attachment require the
Godot .NET editor. Initialize the C# solution using Godot's C# tooling if the
project does not have one yet, and install the SDK required by that Godot
version. Hera writes script files only.

Build using Godot's Build action or `dotnet build` in the project directory.
Reload the assembly (restart the editor if needed), then:

```sh
hera script inspect res://Player.cs
hera node attach-script . res://Player.cs
hera scene save
hera run --current --wait
hera game qa discover
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
methods. Call them using `hera game node call /root/Main QaReady`.
`Qa` must be followed by an uppercase ASCII letter. `eval` continues to use
Godot's GDScript expression evaluator for both project languages.

## Verification (2026-09-07)

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
