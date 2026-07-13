# Nonvisual CI recipe

This recipe exercises a real Hera-enabled Godot editor without claiming that a
headless process can render or operate a GUI. The static script tier is
headless; the live runtime tier uses a private virtual display so that the
editor can launch the game process it controls. It is the repeatable lifecycle
for the pinned runtime check and for equivalent local diagnosis.

## Support boundary

The tiers below are deliberately narrower than the full command surface. A
green static check does not imply that every Hera command, every older Godot
runtime, or GUI-editor behavior works headlessly.

| Tier | Godot versions | Evidence and scope | Not implied |
| --- | --- | --- | --- |
| Static addon scripts | 4.2 and 4.7 | The existing CI matrix runs `--headless --check-only --script` over every addon script. | A live editor, addon protocol, game runtime, or visual behavior. |
| Nonvisual editor, addon process/protocol, and runtime logic | 4.7 only | The pinned 4.7 lifecycle starts the enabled addon inside an isolated Xvfb display, publishes a fresh heartbeat, answers CLI requests, passes `smoke --skip-game`, and runs [`tests/headless/runtime-logic.json`](../tests/headless/runtime-logic.json). | Support for this live path on 4.2–4.6, all Hera commands, screenshots, visual UI, renderer output, window/input behavior, or GUI-editor behavior. |
| Excluded from this runtime tier | All versions | **Screenshots, visual UI, renderer output, window/input behavior, and GUI-editor claims are excluded.** The virtual display only lets the editor launch a controlled game; no visual result is asserted. | A blank or absent image is not a visual test result. |

The distinction is required by Godot itself: `--headless` selects the headless
display driver and dummy audio driver, while the rendering and display-server
APIs document that headless mode disables rendering and window management. See
the canonical [Godot CLI documentation source](https://github.com/godotengine/godot-docs/blob/master/tutorials/editor/command_line_tutorial.rst),
[RenderingServer source documentation](https://github.com/godotengine/godot-docs/blob/master/classes/class_renderingserver.rst),
and [engine command-line implementation](https://github.com/godotengine/godot/blob/master/main/main.cpp).

Godot does not forward `--headless` from an editor to the game process it
launches, so the runtime scenario cannot run under a headless editor on a
display-less hosted runner. The recipe instead uses a private Xvfb server and
the compatibility renderer with software OpenGL; it does not capture or judge
images. GitHub's standard hosted-runner specification does not promise a GPU,
so this remains a nonvisual, software-rendered lifecycle check. See
[GitHub-hosted runners](https://docs.github.com/en/actions/reference/runners/github-hosted-runners).

## Checksum decision: do not add a one-sided check

**Decision (2026-07-13):** do not add SHA-256 verification to the CI Godot
download by itself. The intended requirement was to verify both that editor
archive and add-on archives downloaded from the Godot Asset Store. GitHub
release metadata provides a SHA-256 digest for the pinned Godot editor asset,
so the editor half is technically possible. The Asset Store half is not.

The public Asset Store download page provides a URL and file size, but not a
SHA-256 value for the exact hosted archive. The store also repackages uploaded
ZIPs: the public `Hera Agent Godot` v0.7.0 download is not byte-identical to
the v0.7.0 GitHub Release ZIP. Its GitHub Release checksum therefore cannot
validate the Store download. The legacy official Asset Library API confirms
the same limitation: its `download_hash` field is always empty for the
official library, so the editor skips the hash check. See the
[Asset Library API contract](https://github.com/godotengine/godot-asset-library/blob/master/API.md#api-get-asset-id)
and the [current Store download page](https://store.godotengine.org/asset/notnull92/hera-agent-godot/).

Do not reopen a combined SHA-256 implementation unless the Store publishes a
trusted hash tied to each exact hosted archive, or the release process adds a
separate, versioned, trusted manifest for the Store-repackaged ZIP. Revisit
the editor-only GitHub-release check only if its separate security benefit is
explicitly requested; it does not satisfy the two-download requirement.

## Preconditions

- Use an official **Godot 4.7 editor binary**, not an export template, and set
  `GODOT_BIN` to its absolute path.
- Run from the repository root, with the Hera plugin enabled by this checkout.
- Use Bash on Linux/macOS or Git Bash/WSL on Windows. The same lifecycle is
  suitable for a GitHub Actions `ubuntu-24.04` job and for local reproduction.
- Keep all generated state under a per-run directory. Do not reuse a
  developer's home directory, a previous heartbeat directory, or a token file.

The command examples below build a fresh CLI into the run directory. An
installed `hera` binary may be substituted only when it is the checkout being
tested.

## Lifecycle

Set `GODOT_BIN` before running this block. The live runtime path also requires
`xvfb-run` (on Ubuntu: install `xvfb` and `xauth`). In GitHub Actions,
`RUNNER_TEMP` and `GITHUB_WORKSPACE` already exist; locally, the fallbacks keep
all state under the checkout.

```bash
set -euo pipefail

repo_dir="${GITHUB_WORKSPACE:-$PWD}"
temp_root="${RUNNER_TEMP:-$repo_dir/.tmp}"
run_root="$temp_root/hera-headless-${GITHUB_RUN_ID:-local}-$$"
artifact_dir="$run_root/artifacts"
export HOME="$run_root/home"
export USERPROFILE="$HOME"
export XDG_CONFIG_HOME="$HOME/.config"
export XDG_DATA_HOME="$HOME/.local/share"
export XDG_CACHE_HOME="$HOME/.cache"
export HERA_AGENT_GODOT_TOKEN="$(openssl rand -hex 24)"

mkdir -p "$artifact_dir" "$XDG_CONFIG_HOME" "$XDG_DATA_HOME" "$XDG_CACHE_HOME"
if [ -n "${GITHUB_ACTIONS:-}" ]; then
  printf '::add-mask::%s\n' "$HERA_AGENT_GODOT_TOKEN"
fi

: "${GODOT_BIN:?Set GODOT_BIN to the pinned Godot 4.7 editor binary}"
go build -o "$run_root/hera" .
hera_bin="$run_root/hera"
heartbeat_dir="$HOME/.hera-agent-godot/instances"
launcher_pid=""
heartbeat_pid=""
heartbeat_file=""

"$GODOT_BIN" --headless --path "$repo_dir" --import \
  >"$artifact_dir/import.stdout.log" \
  2>"$artifact_dir/import.stderr.log"

cleanup() {
  original_status=$?
  cleanup_failed=0
  trap - EXIT
  set +e

  if [ -n "$heartbeat_pid" ]; then
    "$hera_bin" --instance "$heartbeat_pid" stop --wait \
      >>"$artifact_dir/cleanup-stop.stdout.json" \
      2>>"$artifact_dir/cleanup-stop.stderr.log"
  fi

  # The background launcher and the heartbeat process can differ on Windows.
  # Both values came from this run, so cleanup never finds processes by name.
  last_cleanup_pid=""
  for cleanup_pid in "$launcher_pid" "$heartbeat_pid"; do
    case "$cleanup_pid" in
      ""|*[!0-9]*) continue ;;
    esac
    if [ "$cleanup_pid" = "$last_cleanup_pid" ]; then
      continue
    fi
    last_cleanup_pid="$cleanup_pid"
    if kill -0 "$cleanup_pid" 2>/dev/null; then
      kill "$cleanup_pid" 2>>"$artifact_dir/cleanup-kill.stderr.log" || true
    fi
  done
  if [ -n "$launcher_pid" ]; then
    cleanup_deadline=$((SECONDS + 10))
    while kill -0 "$launcher_pid" 2>/dev/null && [ "$SECONDS" -lt "$cleanup_deadline" ]; do
      sleep 0.2
    done
    if kill -0 "$launcher_pid" 2>/dev/null; then
      echo "captured launcher survived cleanup: $launcher_pid" >&2
      cleanup_failed=1
    else
      wait "$launcher_pid" 2>>"$artifact_dir/cleanup-wait.stderr.log" || true
    fi
  fi

  if [ -n "$heartbeat_file" ]; then
    deadline=$((SECONDS + 10))
    while [ -e "$heartbeat_file" ] && [ "$SECONDS" -lt "$deadline" ]; do
      sleep 0.2
    done
    if [ -e "$heartbeat_file" ]; then
      if kill -0 "$heartbeat_pid" 2>/dev/null; then
        echo "captured heartbeat survived cleanup: $heartbeat_file" >&2
        cleanup_failed=1
      else
        # The file was selected from this run's private HOME and its filename
        # was verified against its embedded PID. Remove it only after that PID
        # is gone, so a forced editor shutdown cannot leak stale discovery.
        rm -f -- "$heartbeat_file"
        if [ -e "$heartbeat_file" ]; then
          echo "could not remove stale captured heartbeat: $heartbeat_file" >&2
          cleanup_failed=1
        fi
      fi
    fi
  fi
  if [ "$original_status" -eq 0 ] && [ "$cleanup_failed" -ne 0 ]; then
    exit 1
  fi
  exit "$original_status"
}
trap cleanup EXIT

mkdir -p "$heartbeat_dir"
before="$artifact_dir/heartbeats-before.txt"
current="$artifact_dir/heartbeats-current.txt"
fresh="$artifact_dir/heartbeats-fresh.txt"
find "$heartbeat_dir" -maxdepth 1 -type f -name '*.json' -printf '%f\n' \
  | LC_ALL=C sort >"$before"

xvfb-run --auto-servernum \
  --server-args="-screen 0 1280x720x24 -nolisten tcp" \
  "$GODOT_BIN" --editor --rendering-method gl_compatibility --path "$repo_dir" \
  >"$artifact_dir/editor.stdout.log" \
  2>"$artifact_dir/editor.stderr.log" &
launcher_pid=$!
printf '%s\n' "$launcher_pid" >"$artifact_dir/launcher.pid"

# Select only a heartbeat file that appeared after this launch in this run's
# private HOME. On Windows, $! can be a console launcher rather than Godot.
deadline=$((SECONDS + 60))
while [ "$SECONDS" -lt "$deadline" ] && [ -z "$heartbeat_pid" ]; do
  find "$heartbeat_dir" -maxdepth 1 -type f -name '*.json' -printf '%f\n' \
    | LC_ALL=C sort >"$current"
  comm -13 "$before" "$current" >"$fresh"
  while IFS= read -r heartbeat_name; do
    candidate_pid="$(basename "$heartbeat_name" .json)"
    case "$candidate_pid" in
      ""|*[!0-9]*) continue ;;
    esac
    candidate_file="$heartbeat_dir/$heartbeat_name"
    if grep -Eq "\"pid\"[[:space:]]*:[[:space:]]*$candidate_pid([[:space:]]*[,}])" "$candidate_file"; then
      heartbeat_pid="$candidate_pid"
      heartbeat_file="$candidate_file"
      break
    fi
  done <"$fresh"
  if [ -z "$heartbeat_pid" ]; then
    sleep 0.2
  fi
done
if [ -z "$heartbeat_pid" ]; then
  echo "Timed out waiting for a fresh heartbeat in the isolated HOME" >&2
  exit 1
fi

status_ready=0
deadline=$((SECONDS + 30))
while [ "$SECONDS" -lt "$deadline" ]; do
  if "$hera_bin" --instance "$heartbeat_pid" status \
    >"$artifact_dir/status.json" 2>"$artifact_dir/status.stderr.log"; then
    status_ready=1
    break
  fi
  sleep 0.2
done
if [ "$status_ready" -ne 1 ]; then
  echo "Hera status did not reach fresh heartbeat pid $heartbeat_pid" >&2
  exit 1
fi
if ! grep -Eq "\"pid\"[[:space:]]*:[[:space:]]*$heartbeat_pid([[:space:]]*[,}])" "$artifact_dir/status.json"; then
  echo "Hera status pid did not match fresh heartbeat pid $heartbeat_pid" >&2
  exit 1
fi

"$hera_bin" version >"$artifact_dir/hera-version.txt"
"$GODOT_BIN" --headless --version >"$artifact_dir/godot-version.txt"
"$hera_bin" instances >"$artifact_dir/instances.json"

"$hera_bin" --instance "$heartbeat_pid" smoke --skip-game \
  >"$artifact_dir/smoke-skip-game.json" \
  2>"$artifact_dir/smoke-skip-game.stderr.log"

"$hera_bin" --instance "$heartbeat_pid" game qa --file tests/headless/runtime-logic.json \
  >"$artifact_dir/runtime-logic.json" \
  2>"$artifact_dir/runtime-logic.stderr.log"

"$hera_bin" --instance "$heartbeat_pid" stop --wait \
  >"$artifact_dir/stop-before-cleanup.json" \
  2>"$artifact_dir/stop-before-cleanup.stderr.log"
```

The editor and the CLI inherit the same `HOME`, `USERPROFILE`, XDG variables,
and `HERA_AGENT_GODOT_TOKEN`. This matters because Hera discovers editor
heartbeats under the user's home directory and resolves opt-in token auth from
the environment before consulting a token file. The recipe's before/after
heartbeat snapshot is only the scoped selection step for the editor it just
launched; all requests then use the public `hera --instance <pid>` and
exit-code surface. See [SECURITY.md](SECURITY.md) and
[CONTRACT.md](CONTRACT.md).

### Fresh-heartbeat selection

`$!` is recorded as `launcher_pid`, not assumed to be the editor PID. On
Windows, a console launcher can create the actual Godot editor as a child with
a different PID. Before launch, the recipe snapshots heartbeat filenames; after
launch, it accepts only a newly created `<pid>.json` whose embedded `pid`
matches its filename, and records that value as `heartbeat_pid`. The bounded
`status` loop then proves that this fresh heartbeat answers through the addon
protocol. Every `hera --instance` call uses `heartbeat_pid`; `launcher_pid` is
retained only for exact, scoped cleanup. Only after status succeeds does the
recipe capture `instances` for diagnostics. Do not select the first listed
editor, and do not kill by process name.

This avoids two distinct hazards:

- A heartbeat can be stale after a crash. The CLI intentionally drops stale
  entries; it is not enough for a JSON file to exist.
- A machine can contain another editor. The fresh heartbeat selection keeps
  `--instance "$heartbeat_pid"` tied to the editor this recipe created, even
  when its launcher PID differs.
- A forced shutdown can leave its final heartbeat file behind. After the
  selected PID is confirmed gone, cleanup removes only that already-validated
  file from the private per-run home; it never deletes a heartbeat by glob.

For a slow machine, extend the **bounded lifecycle deadline** first. If one RPC
needs more than the default five seconds, use the documented global flag before
the command, for example `hera --instance "$heartbeat_pid" --timeout 10000 status`.
That flag bounds one request, not the whole polling loop.

## Why these two checks

`smoke --skip-game` verifies the live editor/addon path without starting a game.
Its implementation checks editor `status`, `diagnostics`, and open scenes, then
returns before any runtime operation. The subsequent JSON scenario explicitly
starts `HeadlessRuntimeFixture.tscn`, asserts `counter == 0`, calls
`qa_increment`, and asserts `counter == 1`; it is the deliberate nonvisual
runtime-logic proof. The scenario leaves shutdown to the explicit
`stop --wait` step.

Do not replace this pair with either of the following:

- `smoke --run-game` starts a game, waits for it, reads the game tree, captures
  and analyzes a runtime screenshot, then stops it. Screenshot capture is
  outside this headless tier.
- `game qa diagnose` is a broad runtime health report that includes runtime UI
  and screenshot checks. It is useful in a graphical QA environment, but its
  screenshot requirement makes it the wrong acceptance test here.

These behaviors are defined by the current [smoke implementation](../cmd/smoke.go),
[diagnostic checks](../cmd/game_qa_diagnose_checks.go),
[nonvisual scenario](../tests/headless/runtime-logic.json), and
[screenshot tool](../addons/hera_agent_godot/tools/screenshot_tool.gd).

## Artifacts and failure diagnosis

The configured GitHub Actions job uploads its runtime logs only when the job
fails. For local diagnosis, retain `$artifact_dir` when investigating a
failure. It should contain:

- the pinned Godot and Hera versions;
- import and editor stdout/stderr, which are the first place to inspect
  project imports, plugin startup, and parse failures;
- the selected heartbeat-PID `status` result, launcher PID, and `instances` snapshot;
- compact JSON and stderr for `smoke --skip-game`, the runtime scenario, and
  both explicit stop attempts.

Never upload the token, a home-directory dump, heartbeat files, or screenshots
for this tier. Command exit codes are the verdict; logs are diagnostic evidence.

| Symptom | Check and corrective action |
| --- | --- |
| No fresh heartbeat before the deadline | Read `editor.stderr.log` and `editor.stdout.log`; confirm the plugin is enabled, `HOME` and `USERPROFILE` are identical, and the editor was started from the intended checkout. Keep the bounded loop; do not add an unbounded sleep. |
| `unauthorized` | Confirm the editor child and CLI inherited the same nonempty `HERA_AGENT_GODOT_TOKEN`. The environment variable intentionally takes precedence over a token file. |
| More than one editor appears | Retain `instances.json`, continue using the captured `heartbeat_pid`, and do not use `pkill`, `killall`, or a global process-name kill. |
| Scenario cannot find a runtime | Preserve the scenario stderr and editor logs, then use `hera --instance "$heartbeat_pid" game instances` as a diagnostic read. The explicit `stop --wait` remains the only normal shutdown path. |
| Cleanup leaves the editor alive | The trap targets only the captured `heartbeat_pid` and `launcher_pid`; inspect their logs and terminate only those known processes. The launcher wait is bounded, so cleanup cannot wait forever. Do not broaden cleanup to unrelated Godot processes. |

## CI and remote status

Use a pinned `ubuntu-24.04` runtime job with its own Xvfb server for this
lifecycle, not an implicit graphics environment. GitHub-hosted jobs are fresh
virtual machines, so keep the runtime logs in the job workspace and make the
runtime job self-contained. See
[GitHub's runner reference](https://docs.github.com/en/actions/reference/runners/github-hosted-runners).

**Remote runtime status: passed.** The authorized [2026-07-13 GitHub Actions
run](https://github.com/NotNull92/hera-agent-godot/actions/runs/29256396824)
passed at commit
[`5c0ba6562961a6a11ab581d0f4eab440d34ce008`](https://github.com/NotNull92/hera-agent-godot/commit/5c0ba6562961a6a11ab581d0f4eab440d34ce008): Go build/vet/test/race/gofmt,
GDScript `--check-only` on Godot 4.2 and 4.7, and the pinned nonvisual
editor→game lifecycle with all runtime-logic requirements covered. It remains
evidence for this documented nonvisual boundary only, not visual, GUI, or
4.2–4.6 live-runtime behavior.

## Source authority

Hera-specific lifecycle and output rules come from this repository's
[support matrix](SUPPORT_MATRIX.md), [commands reference](COMMANDS.md),
[output contract](CONTRACT.md), and [security guide](SECURITY.md). Godot facts
are checked against the canonical [Godot engine](https://github.com/godotengine/godot)
and [Godot documentation](https://github.com/godotengine/godot-docs)
repositories; hosted-runner facts come from [GitHub Docs](https://docs.github.com/en/actions/reference/runners/github-hosted-runners).
