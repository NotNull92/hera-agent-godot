# CLI package-manager manifests

How the companion `hera` CLI is distributed beyond the one-line installers.
Every active manifest pins the versioned release URLs
(`/releases/download/v<ver>/...`) and SHA256 values from that release's
`checksums.txt` — never the version-less `latest/download` URLs, which
would break hash pinning.

## Scoop (Windows) — [`bucket/hera.json`](../bucket/hera.json)

```powershell
scoop bucket add hera-agent-godot https://github.com/NotNull92/hera-agent-godot
scoop install hera
```

Per release: bump `version`, the two `architecture` URLs, and their `hash`
values (lowercase, from `checksums.txt`). The `checkver`/`autoupdate`
blocks let Scoop's `checkver -u` tooling do the bump mechanically.

## Homebrew (macOS / Linux) — [NotNull92/homebrew-hera](https://github.com/NotNull92/homebrew-hera)

```sh
brew install NotNull92/hera/hera
```

Per release: in the tap repo's `Formula/hera.rb`, bump `version`, the four
platform URL/`sha256` pairs, and the `test do` version assertion.

## npm (any platform) — [`npm/`](npm/)

```sh
npm install -g hera-godot     # or: npx hera-godot status
```

The `hera-godot` package is a thin wrapper: its postinstall (or the first run,
when scripts were skipped) downloads the pinned release binary for the current
platform/arch, verifies the SHA256 from [`npm/manifest.json`](npm/manifest.json),
and unpacks it next to the launcher. `hera` and the transitional
`hera-agent-godot` alias are both exposed as bins.

Per release: bump `version` in `npm/package.json` and `npm/manifest.json`,
refresh every `sha256` in `manifest.json` from the release's `checksums.txt`,
then `npm publish` from `packaging/npm/` (test first with `npm pack` + a
scratch `npm install <tarball>`).

## Retired: winget (Windows)

**Decision recorded 2026-07-14: do not submit Hera to
`microsoft/winget-pkgs`.** No public winget PR was opened.

Hera is a Godot addon plus a companion CLI, not a standalone Windows app. A
general Windows catalog distributes only the CLI and creates a misleading,
separate distribution focus for an addon whose supported installation path is
the Godot Asset Store or the addon release ZIP. Keep the CLI installation
paths above; do not use `winget` as an installation option.

[`packaging/winget/manifests/`](winget/manifests/) is retained only as a
historical, locally validated artifact. Do **not** bump, revalidate, submit, or
advertise it. Reopen this decision only after a new explicit user product
decision, not merely because the files still exist.
