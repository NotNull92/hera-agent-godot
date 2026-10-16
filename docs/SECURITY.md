# Security: threat model & shared-token auth

Hera exposes a **localhost HTTP bridge into a live Godot editor**. Anything
that can POST to it can read project state, mutate scenes, run `eval`
expressions, and launch play sessions — that is, it can act as the editor's
user. This document states what the bridge does and does not defend against,
and how to enable the opt-in shared-token auth.

## What the transport guarantees

Detailed evidence is observational, not an authorization token or file lock.
Hashes and session IDs do not prevent manual/external changes between reads.
Capture operation IDs are caller-supplied correlation labels, not proof of an
operation's effects. Disk validation may run native dependency initializers;
explicit resource persistence verification may load resource scripts/dependencies
again. Use isolated copies for untrusted projects. Neither action automatically
reloads editor buffers or updates running game code. Evidence adds local engine
and file paths to output; keep private artifacts out of public reports.

- The addon binds **`127.0.0.1` only** (ports 8770–8785, first free). It never
  listens on `0.0.0.0`; nothing off-machine can connect directly.
- **Browser-origin requests are rejected** (403 on any `Origin` header), which
  blocks the straightforward malicious-web-page CSRF and DNS-rebinding paths —
  browsers attach `Origin` to cross-origin POSTs.
- Requests are capped at 1 MiB, one request per connection
  (`Connection: close`), with separate 5 s receive and response-write deadlines.
  Response writes advance in bounded, nonblocking chunks from the editor loop;
  a client that stops reading cannot stall it. Queued tools retain their own
  execution deadlines; disconnected clients and expired writes are discarded.
- No TLS: traffic never leaves the loopback interface.
- Heartbeat files (`~/.hera-agent-godot/instances/*.json`) advertise pid,
  port, project path, and Godot version. They contain **no secrets**, and live
  under the user's home directory with default user permissions.

## Threat model

Editor log collection is memory-only and uses the same authenticated RPC
boundary. It retains at most 1024 events, with 4096 message characters and
512 characters per location string. Logs can contain project secrets or local
paths; authorized clients can read them, so redact evidence before sharing.
The fixed Logger adapter source accepts no client code. Callbacks only store
bounded metadata under a mutex; they do not access editor objects, write files,
capture variables, or log recursively. Plugin teardown removes the registered
logger and clears its buffer. Session cursors are observation markers, not
authentication credentials or proof that an action caused an error.

**In scope** (what token auth addresses):

- *Other OS users on a shared machine.* Loopback is host-wide: any local user
  can reach `127.0.0.1:8770`. Without a token, another account on the same
  host can drive your editor.
- *Unprivileged local processes you don't fully trust* (sandboxed apps, dev
  containers with host-loopback access, forwarded ports via `ssh -L` or dev
  tunnels). A token turns "can reach the port" into "must also know the
  secret".
- *Non-browser confused deputies* — anything that can be tricked into POSTing
  a fixed body to a localhost URL but cannot read your files.

**Out of scope** (a token does not help):

- *Malware running as you.* It can read the token file, or skip Hera entirely
  and edit project files directly. Hera does not try to defend against a
  compromised user account.
- *`eval` and mutation misuse by an authorized client.* `eval` executes a
  C# expression by default, or an explicit GDScript expression, in the editor;
  treat any process allowed to reach Hera
  as having editor-level control. There is no sandbox and no per-command
  permission tiering (all-or-nothing by design — see the CLI-first identity).
- *Timing side channels on token comparison* — accepted as irrelevant for a
  loopback dev tool.

## Opt-in shared-token auth

Off by default. When enabled, every `/rpc` request must carry a matching
`X-Hera-Token` header; mismatches get HTTP 401 and the CLI reports
`unauthorized: ...` with exit code 1.

Both sides resolve the token identically:

1. `HERA_AGENT_GODOT_TOKEN` environment variable (wins if non-empty), else
2. `~/.hera-agent-godot/token` (contents, whitespace-trimmed), else
3. empty → auth off.

A missing or empty token file intentionally leaves auth off. If the file cannot
be opened or read for another reason, the addon refuses to start its HTTP
listener or publish a heartbeat. Fix the file access error and restart the
editor, or supply a non-empty environment token override.

To enable:

```sh
# POSIX shell: restrict both newly created and existing files before writing.
umask 077
mkdir -p ~/.hera-agent-godot
chmod 700 ~/.hera-agent-godot
touch ~/.hera-agent-godot/token
chmod 600 ~/.hera-agent-godot/token
openssl rand -hex 24 > ~/.hera-agent-godot/token
```

On Windows, use a file in your own profile with an ACL that excludes other
accounts, or use the environment variable. Keep the secret out of the repo.

then **reload the plugin** (or restart the editor): the addon reads the token
once at plugin start. The CLI re-reads it on every invocation, so no CLI-side
action is needed. The editor log and the Hera panel show `token auth on` when
the addon loaded a token.

Notes:

- The token protects the **editor side**. The CLI sending a token to a
  no-auth editor is harmless (the header is ignored).
- Rotation: write a new token, reload the plugin.
- CI/headless: set `HERA_AGENT_GODOT_TOKEN` for both the editor process and
  the CLI instead of creating the file.

## Reporting

Found a hole in the above? Open a GitHub issue (or a private security
advisory on the repo) with reproduction steps.

See [ARCHITECTURE.md](./ARCHITECTURE.md) §7 for where these boundaries live in
code, and [CONTRACT.md](./CONTRACT.md) for the failure semantics.

## C# build and evaluation / C# 빌드와 평가

C# validation runs MSBuild input evaluation, restore and build. **MSBuild targets
and source generators execute code; path redirection is not a sandbox.** Custom
project logic can write outside the isolated output directory or access network
and user files. Use trusted projects or an isolated account/container. Hera never
modifies csproj/solution files or builds into the editor's standard output paths;
this is not a restriction on arbitrary project code.

The Roslyn helper compiles outside the editor, then the native `#if TOOLS` loader
executes the expression inside it. Editor expressions are arbitrary synchronous
code: they can block, retain callbacks, change files and mutate scenes without
undo. HTTP timeout does not stop execution or establish that no effects occurred.
ALC unload checks can fail after side effects; an editor restart may be necessary.
The helper and DLL references must be trusted. Module-ID checks detect a project
revision mismatch but do not provide a sandbox or certify dependency freshness.

The panel shows auth On/Off (Unknown if reading failed), never token values or
paths. Copy setup includes the project path, port and addon version, not token
values, paths or environment values. Treat project paths as private when sharing.
Version headers are client-supplied observations and do not authenticate a CLI.

MSBuild 타깃과 소스 생성기는 코드를 실행합니다. 출력 경로 격리는 샌드박스가
아니며 사용자 코드는 다른 파일·네트워크에 접근할 수 있습니다. eval은 동기 실행으로
에디터를 멈추거나 콜백을 유지할 수 있고, 실패·타임아웃은 변경 취소가 아닙니다.
상태 패널과 복사 텍스트에는 토큰 값·경로·환경변수 값이 없지만 프로젝트 경로는 포함됩니다.
