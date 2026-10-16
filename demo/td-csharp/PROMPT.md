# Demo prompt (fixed)

This is the exact prompt given to the coding agent in the v2.0.0 introduction
video. It is frozen once the rehearsal passes; if the agent does not use Hera's
C# commands on its own, fix the driver card in `AGENTS.md`, not this prompt.

```text
Build a playable mini tower defense game inside the live Godot editor that is already open on this project, using the hera-godot CLI for every editor action. Read AGENTS.md first and follow its driver card. This is a Godot 4.7 .NET project: write all game scripts in C#. Use the sprites in res://assets. Requirements: a fixed enemy path on a small map, wave spawning, tower placement by clicking, towers that auto-target and shoot, enemy health, player money and lives, and a HUD. Run the game through Hera, check it with runtime inspection and screenshots, fix what you find, and finish with a short summary of what works and what does not.
```
