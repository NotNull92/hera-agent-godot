# Security: threat model & shared-token auth

Hera exposes a **localhost HTTP bridge into a live Godot editor**. Anything
that can POST to it can read project state, mutate scenes, run `eval`
expressions, and launch play sessions — that is, it can act as the editor's
user. This document states what the bridge does and does not defend against,
and how to enable the opt-in shared-token auth.

## What the transport guarantees

- The addon binds **`127.0.0.1` only** (ports 8770–8785, first free). It never
  listens on `0.0.0.0`; nothing off-machine can connect directly.
- **Browser-origin requests are rejected** (403 on any `Origin` header), which
  blocks the straightforward malicious-web-page CSRF and DNS-rebinding paths —
  browsers attach `Origin` to cross-origin POSTs.
- Requests are capped at 1 MiB, one request per connection
  (`Connection: close`), 5 s per-connection timeout.
- No TLS: traffic never leaves the loopback interface.
- Heartbeat files (`~/.hera-agent-godot/instances/*.json`) advertise pid,
  port, project path, and Godot version. They contain **no secrets**, and live
  under the user's home directory with default user permissions.

## Threat model

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
  GDScript expression in the editor; treat any process allowed to reach Hera
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

To enable:

```
# any random string; keep it out of the repo
openssl rand -hex 24 > ~/.hera-agent-godot/token   # or write the file by hand
```

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
