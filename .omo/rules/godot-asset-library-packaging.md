---
description: Godot Asset Library packaging rules for Hera Agent Godot
alwaysApply: true
---

# Godot Asset Library Packaging

When preparing the Godot Asset Library download ZIP for this repository:

- Include the license inside the add-on content folder at `addons/hera_agent_godot/LICENSE`.
- Do not add a duplicate `LICENSE` file at the ZIP download root when `addons/hera_agent_godot/LICENSE` is present.
- Keep the repository root `LICENSE` for GitHub/source distribution.
- The upload ZIP root should contain `addons/hera_agent_godot/...` as the installed content; avoid repo-only files unless a release process explicitly requires them.
- Before handoff, verify the archive contains `addons/hera_agent_godot/plugin.cfg` and `addons/hera_agent_godot/LICENSE`.
