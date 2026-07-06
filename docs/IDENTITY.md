# Hera Identity

Hera is a shell-native control layer for AI agents working inside real game
editors.

For Godot, the short version is:

> Hera gives agents eyes, hands, and proof in the live Godot editor.

## Product Promise

Hera should feel like a precise assistant sitting inside the editor, not a broad
automation framework guessing from disk files.

Every feature should reinforce three promises:

1. **Live editor truth**
   Hera reads and changes the actual open Godot editor or running game process.
   If a command cannot target the real process safely, it should fail clearly.

2. **Low-token control**
   Hera keeps the agent's context small: shell commands, compact JSON by
   default, scoped reads before full dumps, and no resident MCP tool schemas.

3. **Proof-first QA**
   Hera should help agents prove work through diagnostics, runtime screenshots,
   semantic UI clicks, deterministic `qa_*` hooks, and requirement-covered QA
   scenarios.

## Voice

Use direct, concrete product language.

Prefer:

- "Inspect the live editor."
- "Click the visible Restart button by text."
- "Fail if a requirement is not covered."
- "Capture a runtime screenshot and check clipping."

Avoid:

- Vague automation claims.
- Treating disk files as proof of runtime behavior.
- Presenting MCP as an enemy. Hera is the shell-native, low-token counterpart.
- Flavor text that hides what the command actually does.

## Naming

Command names should stay boring and discoverable:

- Use nouns from the Godot surface: `scene`, `node`, `resource`, `signal`,
  `game`, `diagnostics`.
- Use scoped subcommands before new top-level commands.
- Keep whimsical Hera language in the plugin panel and docs tagline, not in
  command names.

Good:

```sh
hera game qa discover
hera game ui tree --type Button --fields name,path,text,disabled
hera game qa --file scenario.json
```

Avoid:

```sh
hera magic
hera divine-click
hera fix-it
```

## Differentiators

Hera's identity is strongest when these are visible together:

- **Shell-native:** works anywhere an agent can run a command.
- **Editor-resident:** executes through a Godot addon, on the editor side.
- **Compact by default:** output is terse unless the agent asks for detail.
- **Runtime-aware:** can inspect and click the running game, not just the scene
  file.
- **QA-oriented:** diagnostics, screenshots, and requirement coverage are part
  of the normal path, not optional polish.
- **Godot-native:** follows Godot concepts and GDScript rules instead of copying
  another engine's abstractions.

## Release Message Template

Use this shape for release notes and store descriptions:

```text
Hera Agent Godot gives AI coding agents eyes, hands, and proof inside a live
Godot editor. It exposes shell-native, low-token commands for inspecting,
editing, running, screenshotting, and QA-checking Godot projects without loading
large MCP tool schemas into every turn.
```

For feature notes, connect each change to one promise:

- Token reduction: "compact/scoped reads"
- Precision: "live editor/runtime truth"
- Implementation accuracy: "requirement-covered QA"
- UI work: "Game Feel guidance plus runtime visual proof"
