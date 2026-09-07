#!/usr/bin/env bash
# GODOT_BIN=/path/to/Godot HERA_BIN=/path/to/hera GODOT_SDK_VERSION=4.7.2 bash "$0"
# Set GODOT_EDITOR_GUI=1 if editor-spawned games cannot start headlessly.
# Mono also needs dotnet; NUGET_CONFIG may point to an offline feed config. KEEP=1 keeps evidence.
set -euo pipefail
: "${GODOT_BIN:?Set GODOT_BIN to the direct Godot executable (not a launcher)}"
: "${HERA_BIN:?Set HERA_BIN to the built Hera executable}"
command -v jq >/dev/null
repo=$(cd "$(dirname "$0")/../.." && pwd)
work=$(mktemp -d "${TMPDIR:-/tmp}/hera-script-languages.XXXXXX")
work=$(cd "$work" && pwd -P)
editor_pid=
editor_args=(--headless)
if [[ ${GODOT_EDITOR_GUI:-0} == 1 ]]; then editor_args=(); fi
cleanup() {
    code=$?
    if [[ -n "$editor_pid" ]]; then hera stop --wait >"$work/cleanup-stop.log" 2>&1 || true; fi
    for pid in "$editor_pid"; do
        if [[ -n "$pid" ]]; then kill "$pid" 2>/dev/null || true; wait "$pid" 2>/dev/null || true; fi
    done
    if [[ ${KEEP:-0} == 1 || $code != 0 ]]; then
        echo "Evidence: $work" >&2
    else
        rm -rf "$work"
    fi
}
trap cleanup EXIT
mkdir "$work/addons"
cp -R "$repo/addons/hera_agent_godot" "$work/addons/"
cat > "$work/project.godot" <<PROJECT
config_version=5
[application]
config/name="$(basename "$work")"
[autoload]
HeraGameInspector="*res://addons/hera_agent_godot/runtime/game_inspector.gd"
[editor_plugins]
enabled=PackedStringArray("res://addons/hera_agent_godot/plugin.cfg")
[dotnet]
project/assembly_name="Smoke"
[rendering]
renderer/rendering_method="gl_compatibility"
PROJECT
hera() { "$HERA_BIN" --instance "$editor_pid" --timeout 2000 "$@"; }
check() { local query=$1; shift; hera "$@" | tee "$work/last.json" | jq -e "$query" >/dev/null; }
reject() {
    if hera "$@" >"$work/rejected.log" 2>&1; then echo "Unexpected success: $*" >&2; exit 1; fi
}
start_editor() {
    "$GODOT_BIN" "${editor_args[@]}" --editor --path "$work" --log-file "$work/editor.log" >"$work/editor.stdout" 2>&1 &
    editor_pid=$!
    for ((attempt=0; attempt<60; attempt++)); do
        kill -0 "$editor_pid" 2>/dev/null || { cat "$work/editor.stdout" >&2; return 1; }
        if hera status >"$work/status.json" 2>/dev/null && jq -e --argjson pid "$editor_pid" '.pid == $pid' "$work/status.json" >/dev/null; then return; fi
        sleep 1
    done
    echo 'Editor did not become ready' >&2; return 1
}
start_editor
check '.csharp_supported | type == "boolean"' status
check '.created == "res://probe.gd"' script create res://probe.gd --lang gdscript --ready --signal ping --export speed:float=3.5
check '.found and (.functions | any(.name == "_ready"))' script inspect res://probe.gd
check '.opened == "res://probe.gd"' script open res://probe.gd
check '.found and .path == "res://probe.gd"' script current
hera scene create res://Main.tscn --root Node --open >/dev/null
reject script create res://Mismatch.cs --lang gdscript
[[ ! -f "$work/Mismatch.cs" ]]
if [[ $(jq -r '.csharp_supported' "$work/status.json") == false ]]; then
    reject script create res://Probe.cs
    [[ ! -f "$work/Probe.cs" ]]
    printf 'using Godot; public partial class Probe : Node {}\n' > "$work/Probe.cs"
    reject script open res://Probe.cs
    reject node attach-script . res://Probe.cs
    check '.language == "csharp" and .assembly_loaded == false and .functions == []' script inspect res://Probe.cs
    echo 'PASS: GDScript and standard-editor C# rejection'
    exit 0
fi
: "${GODOT_SDK_VERSION:?Set GODOT_SDK_VERSION to the installed Godot .NET SDK version}"
command -v dotnet >/dev/null
check '.language == "csharp" and .build_required and .exports == 1' script create res://Probe.cs --lang csharp --class-name Probe --ready --process --physics-process --input --unhandled-input --signal Ping --export Speed:float=3.5f
reject script create res://Wrong.cs --class-name Different
[[ ! -f "$work/Wrong.cs" ]]
check '.assembly_loaded == false and .functions == []' script inspect res://Probe.cs
reject node attach-script . res://Probe.cs
# Shut the editor down before adding build files and a second partial declaration.
kill "$editor_pid"; wait "$editor_pid" || true; editor_pid=
major=$(dotnet --version); major=${major%%.*}
tfm=${DOTNET_TFM:-net${major}.0}
cat > "$work/Smoke.csproj" <<PROJECT
<Project Sdk="Godot.NET.Sdk/$GODOT_SDK_VERSION">
  <PropertyGroup><TargetFramework>$tfm</TargetFramework><EnableDynamicLoading>true</EnableDynamicLoading></PropertyGroup>
</Project>
PROJECT
cat > "$work/Probe.Qa.cs" <<'CS'
public partial class Probe
{
    public bool QaReady() => Speed == 3.5f;
}
CS
build_args=()
if [[ -n ${NUGET_CONFIG:-} ]]; then build_args+=(--configfile "$NUGET_CONFIG"); fi
dotnet build "$work/Smoke.csproj" "${build_args[@]}" >"$work/build.log" 2>&1
start_editor
check '.assembly_loaded and (.functions | any(.name == "QaReady")) and (.signals | any(.name == "Ping")) and (.exports | any(.name == "Speed"))' script inspect res://Probe.cs
check '.opened == "res://Probe.cs" and .language == "csharp"' script open res://Probe.cs
hera scene open res://Main.tscn >/dev/null
check '.script == "res://Probe.cs"' node attach-script . res://Probe.cs
check '.properties.script | contains("res://Probe.cs")' node get . --prop script
hera scene save >/dev/null
hera run --current --wait >"$work/run.json"
check '.methods | any(.name == "QaReady")' game qa discover
check '.result == "true" and .type == "bool"' game node call /root/Main QaReady
hera stop --wait >"$work/stop.json"
echo 'PASS: GDScript, C# creation, assembly metadata, attachment and runtime QaReady'
