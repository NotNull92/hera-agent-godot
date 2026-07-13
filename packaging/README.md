# Package manager manifests

How the `hera` CLI is distributed beyond the one-line installers. Every
manifest pins the versioned release URLs
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

## winget (Windows) — [`packaging/winget/manifests/`](winget/manifests/)

Not yet live: the manifest set passes
`winget validate --manifest packaging/winget/manifests/n/NotNull92/Hera/0.8.0`
but `winget install NotNull92.Hera` only works once a copy is accepted
into [microsoft/winget-pkgs](https://github.com/microsoft/winget-pkgs).
Submit with either:

- `wingetcreate submit packaging/winget/manifests/n/NotNull92/Hera/0.8.0`
  (needs a GitHub token), or
- fork `microsoft/winget-pkgs`, copy the folder to
  `manifests/n/NotNull92/Hera/0.8.0/`, and open a PR.

Per release: copy the folder to the new version, bump `PackageVersion`,
`ReleaseDate`, the URLs, and the `InstallerSha256` values (uppercase),
re-validate, and submit a new winget-pkgs PR.
