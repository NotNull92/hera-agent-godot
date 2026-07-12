# Contract golden test data

Used by `cmd/contract_golden_test.go` (see `docs/CONTRACT.md` for the contract
these tests pin).

- `responses/*.json` — canned `/rpc` response bodies served by the mock editor.
  Captured live from a Godot 4.7-stable editor on 2026-07-13, except the
  `game`/`run` runtime shapes, which are synthesized from the addon source
  (capturing them would require a real play session).
- `*.golden` — the expected CLI stdout, byte-for-byte (trailing newline
  included). `instances.golden` stores `port`/`ts` normalized to `8770`/`0`.
- `batch_input.json` — the `--file` input used by the `batch` case.

Regenerate goldens after an **intentional** contract change with:

```
go test ./cmd -run TestContract -update
```

then list the change in the release notes (and, post-1.0, follow the
deprecation policy for stable commands).
