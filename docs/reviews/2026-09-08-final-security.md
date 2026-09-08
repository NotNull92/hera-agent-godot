# Final security review — 2026-09-08

Verdict: **PASS** for the review-fix diff under the existing opt-in loopback trust model. No evidenced security blocker found. This is an independent source review, not an exhaustive penetration test or release approval.

Base commit: `d6543c421e9b830301a8e0329c7ba56eaa2a1fd8`.
Source snapshot SHA256: `81fa9cf375ea75f4fcf6c487b08768cf5862d47083243c14987ddda8ca873410`.
Both were independently verified. Snapshot calculation covered 46 changed/untracked non-document files, sorted by ordinal path, hashing each UTF-8 path + NUL + raw file bytes + NUL. Documents and Markdown were excluded.

## Evidence

| Boundary | Review result |
|---|---|
| Configured token failures | `server/http_server.gd:206` distinguishes missing files from other open failures, rejects directory paths and read errors, closes opened files, and preserves explicit empty/missing opt-out plus nonempty environment precedence. `hera_agent_plugin.gd:90` returns on error before creating a listener or heartbeat. Exit cleanup tolerates absent server/heartbeat. Errors contain paths and error codes, not token contents. |
| HTTP response backpressure | `server/http_server.gd:72` advances at most 65,536 bytes per connection per poll through `put_partial_data`; pending output remains tracked and expires using an absolute five-second write deadline. Late responses cannot recreate removed entries. Receive timeout, body-size rejection, loopback binding, token verification and browser-origin rejection remain in place. |
| Path and mutation validation | The shared lexical path helper preserves the former component/backslash checks; changed operational callers retain their scheme checks. `node_tool.gd:344` additionally limits resolved nodes to the scene subtree, and deletion checks resolved root identity. Resource/theme updates now finish input validation before invoking setters. No new arbitrary external path or shell execution surface was introduced. |
| Runtime IPC | `tools/game_tool.gd:63` closes and checks a temporary request before publishing its final name, and removes failed publications. The runtime scans only `.json`, leaves incomplete/non-dictionary input unconsumed, consumes a valid request before dispatch/await, and retains target-PID rejection. Response reading still requires the generated request ID within the selected PID directory. IDs and target PIDs are set by the editor, overriding RPC input. |
| Regression isolation | `tests/headless/run.sh` creates separate copied projects, home/profile and platform data/config/cache directories for each script, clears inherited token auth, bounds each process, retains failures, and restricts successful cleanup to its generated regression root. CI invokes this harness. The token fixture uses synthetic values in a temporary directory and restores its process environment. |

The reviewed regressions cover correct/wrong/missing tokens, browser-origin denial, oversize declaration and invalid JSON rejection, real socket backpressure and late responses, temporary/partial IPC requests, exactly-once consumption, and wrong-PID refusal. The implementation record reports an exclusive-lock token failure with zero heartbeats. These execution claims were inspected as supplied evidence; this lane did not rerun Godot or alter credentials, user projects, or source.

## Nonblocking limits

- The one-MiB request limit is a rejection threshold after reading currently available bytes, not an allocation ceiling. There is no aggregate connection/output memory budget. These existing resource-exhaustion limits are not fixed by per-connection nonblocking output; this PASS does not certify resistance to sustained local flooding.
- Queued requests intentionally have no transport deadline; ordinary asynchronous tools own their deadlines. Arbitrary authorized script execution remains outside the security boundary.
- Project path checks are lexical, not symlink confinement. Runtime IPC is local filesystem coordination with PID/request identity, not a separate authentication system. Neither was promoted to a sandbox in this review.
- Missing/empty tokens remain intentionally unauthenticated. POSIX setup instructions now restrict both directory and token permissions before secret generation. No existing machine ACL was inspected or changed.

Severity of blocking findings: **none**. The above limits do not establish a regression in the reviewed changes.
