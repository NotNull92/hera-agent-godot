#!/usr/bin/env bash
# Run standalone behavioral regressions without using the developer's project or home.
set -euo pipefail
: "${GODOT_BIN:?Set GODOT_BIN to an absolute direct Godot executable (not a console launcher)}"
repo_dir=$(cd "$(dirname "$0")/../.." && pwd -P)
run_dir=$(mktemp -d "${TMPDIR:-/tmp}/hera-regressions.XXXXXX")
run_dir=$(cd "$run_dir" && pwd -P)
cleanup() {
    result=$?
    if [[ $result != 0 || ${KEEP:-0} == 1 ]]; then
        printf 'Regression evidence: %s\n' "$run_dir" >&2
    else
        case "$run_dir" in
            */hera-regressions.*) rm -rf -- "$run_dir" ;;
            *) printf 'Refusing unexpected cleanup path: %s\n' "$run_dir" >&2; exit 1 ;;
        esac
    fi
}
trap cleanup EXIT
native_path() {
    if command -v cygpath >/dev/null; then cygpath -m "$1"; else printf '%s\n' "$1"; fi
}
count=0
for source in "$repo_dir"/tests/headless/*_test.gd; do
    name=$(basename "$source" .gd)
    fixture="$run_dir/$name"
    mkdir -p "$fixture/addons" "$fixture/tests/headless" "$fixture/home" "$fixture/appdata" "$fixture/cache" "$fixture/data" "$fixture/config"
    cp -R "$repo_dir/addons/hera_agent_godot" "$fixture/addons/"
    cp "$repo_dir"/tests/headless/*.gd "$fixture/tests/headless/"
    cat > "$fixture/project.godot" <<PROJECT
config_version=5
[application]
config/name="$name"
[rendering]
renderer/rendering_method="gl_compatibility"
PROJECT
    test_env=(env "HOME=$(native_path "$fixture/home")" "USERPROFILE=$(native_path "$fixture/home")" "APPDATA=$(native_path "$fixture/appdata")" "XDG_DATA_HOME=$(native_path "$fixture/data")" "XDG_CACHE_HOME=$(native_path "$fixture/cache")" "XDG_CONFIG_HOME=$(native_path "$fixture/config")" HERA_AGENT_GODOT_TOKEN=)
    editor_args=()
    if grep -q '^# Hera test: editor$' "$source"; then
        editor_args=(--editor)
        "${test_env[@]}" timeout 45s "$GODOT_BIN" --headless --editor --path "$fixture" --import >"$fixture/import.log" 2>&1
        if grep -Eq 'SCRIPT ERROR|Parse Error|Parser Error' "$fixture/import.log"; then cat "$fixture/import.log"; exit 1; fi
    fi
    printf 'Testing %s\n' "$name"
    code=0
    "${test_env[@]}" timeout 45s "$GODOT_BIN" --headless "${editor_args[@]}" --path "$fixture" --script "res://tests/headless/$name.gd" >"$fixture/test.log" 2>&1 || code=$?
    cat "$fixture/test.log"
    if [[ $code != 0 ]] || grep -Eq 'SCRIPT ERROR|Parse Error|Parser Error' "$fixture/test.log"; then
        printf 'FAIL: %s (exit %s)\n' "$name" "$code" >&2
        exit 1
    fi
    count=$((count + 1))
done
printf 'PASS: %d standalone Godot regressions\n' "$count"
