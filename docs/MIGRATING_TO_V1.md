# Migrating to v1

Hera v1.0.0 does not intentionally break the v0.9.0 CLI or addon workflow.
The release turns the documented stable output contract into a compatibility
promise and adopts semantic versioning for future changes.

## Upgrade

1. Upgrade the `hera-godot` CLI through GitHub Releases, npm, Homebrew, Scoop, or the
   one-line installer.
2. Replace `addons/hera_agent_godot/` with the matching addon from the same
   release (currently `v1.2.0` in this tree; published channels may still be
   `v1.1.0` until tagged) or the Godot Asset Store.
3. Fully quit and restart Godot. Toggling the plugin does not reload every
   preloaded addon script from disk.
4. Run `hera-godot version`, `hera-godot status`, and `hera-godot smoke --skip-game`.

5. Update scripts to call `hera-godot` (or the retained `hera-agent-godot` alias).
   New installs do not register `hera`; npm upgrades remove that package bin.
   One-line installers preserve an existing `hera`. Check its resolved path and
   ownership before manually removing it, and remove it only if it belongs to an
   older Hera Godot install. Another tool may own it. Verify with
   `hera-godot version`. Archive names and internal `hera` binaries are unchanged.

The CLI and addon ship together and should use the same release version.
`v1.2.0` preserves stable JSON and changes the installed command name from `hera` to `hera-godot`.
Existing project scenes, the opt-in shared token, instance discovery location,
and `hera-agent-godot` transitional command alias remain compatible.

## Automation consumers

- Stable commands retain their documented fields, JSON types, output streams,
  and exit-code meanings for the v1 major line.
- Additive fields may appear in minor or patch releases. JSON consumers must
  ignore unknown fields.
- Experimental commands and fields may still change in a minor release; those
  changes are called out in release notes.
- Error message text is not a machine-readable contract. Branch on exit codes
  and documented response fields instead.

The binding surface and deprecation rules are defined in
[CONTRACT.md](CONTRACT.md).
