# Hera: live Godot workflow

Use `hera` to inspect and control the **running** Godot editor when a task
needs scene/node/UI state, editor changes, play control, diagnostics, or runtime
QA. Do not infer those facts from project files. Hera is verified on Godot
4.2–4.7 (4.7 recommended) when the Hera Agent Godot addon is enabled.

1. Start with `hera status`. If no editor is found, ask the user to enable the
   addon under **Project Settings → Plugins**. If multiple editors are present,
   run `hera instances` and add `--instance <pid>` to mutations.
2. Before UI work, run `hera guidance ui`. Keep reads small: use compact output,
   `hera --ids scene tree`, selected `node get --prop/--props`, scoped
   `game ui tree`, and `game qa discover` before full dumps.
3. Prefer undoable editor commands (`node add/set/remove`, `signal
   connect/disconnect`) to `eval`. Scene/resource/script/project operations are
   persistent. `game node set/call`, `game click`, and `game input` are
   runtime-only and disappear when the game stops.
4. Confirm each edit with `node get` or `scene tree`; after a run, read
   `hera output --type error` or `hera diagnostics`. For visual/UI work, use
   `game ui tree`, semantic `game click`, and
   `hera screenshot --runtime --analyze`. For prompt requirements, prefer a
   `game qa --file` scenario with `requirements` and per-step `covers`.
5. After GDScript changes, run the project's headless `--check-only` gate when
   available, then re-check diagnostics. Never print or commit
   `HERA_AGENT_GODOT_TOKEN`.
