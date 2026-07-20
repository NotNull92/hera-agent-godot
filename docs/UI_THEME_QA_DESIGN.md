# UI Theme QA — design

> A Godot-native design-QA capability for Hera: it measures `Control`-node
> theme tokens in a live editor, finds undisciplined spacing, unscaled type,
> ad-hoc colors, and failing contrast, and snaps them to a reference corpus —
> using Hera's existing inspect/mutate/QA primitives.

**Status: complete.** All planned phases have shipped. The `ui-theme-qa` skill
covers six areas in both plugin trees — `spacing`, `type-scale`, `color` and
`contrast` enforce theme tokens; `containers` and `decoration` report only. The
gaps that blocked project-level work are closed by `hera theme get/set` (G1) and
`hera screenshot diff` (G2). The one remaining phase, a wholesale restyle mode,
is **deliberately unbuilt** — see §11.1 for why, so it is not mistaken for
pending work. Capability claims cite code paths (`file:line`) verified against
the addon sources.

---

## 1. Motivation

Agent-generated Godot UI drifts toward the same statistical defaults every
time: spacing constants picked ad hoc, a random spread of font sizes with no
modular relationship, per-node color overrides with no shared palette, and text
whose contrast was never checked. None of these are taste failures — they are
*undecided values*, and each is a single measurement away from being detected.

Hera can already **see** and **change** a live editor, but it has no notion of
"this UI is undisciplined." The closest existing surface, `game qa diagnose`,
is deliberately render-focused: error counts, tree truncation, blank frames,
clipping (`cmd/game_qa_diagnose_checks.go`). It answers *did this render*, not
*is this designed*.

This capability fills that gap without inventing taste: it removes the
statistical defects and forces undecided values to become decided ones.

---

## 2. Operating principles

Each of these is forced by a property of Godot or of Hera's transport — they
are not stylistic preferences.

1. **A check is a predicate, not a status.** A finding never stores
   "done/not-done". It carries a `check` that is **re-measured from the live
   editor every time** (inspect → enforce → re-inspect). Stored status rots the
   moment anything else touches the scene, and produces the "trust the earlier
   note and silently skip" failure. Hera already applies this shape in
   `game qa diagnose`, where every check re-evaluates live data.
2. **Snap, don't invent.** Replacement values (spacing rungs, type scale,
   palette, contrast target) come from a fixed reference corpus. A model asked
   to "pick a better number" reintroduces exactly the arbitrariness being
   removed. The corpus is rooted in Godot's own default theme, so the values it
   snaps to are the engine's, extended only by rules stated on its page.
3. **Inspect is static and parallel; enforce is sequential and undoable.**
   Reading geometry and theme tokens is side-effect-free, so ordering does not
   matter and areas can run concurrently. Mutations register with
   `EditorUndoRedoManager` (`node_tool.gd:151`), so the developer keeps Ctrl+Z
   over everything the tool did — which is what makes a mechanical pass safe to
   run on a real project.
4. **Thin orchestrator, isolated areas.** The skill holds routing only; each
   area's rules live in a reference doc and each area's findings live in files.
   One context holding every rule while fixing a large UI lets loud items crowd
   out quiet ones, so rules get silently dropped. Isolation is a correctness
   measure, not an organizational one.
5. **Render confirms; it does not measure.** Design facts are measured
   structurally (rects, theme tokens, contrast math). The screenshot analyzer is
   a coarse whole-image heuristic (`game_image_analyzer.gd:10`) — it can tell
   you the frame is blank or clipped, never that spacing is off a ladder.

---

## 3. Scope

**In scope (reductive only):**

- Edited-scene `Control` trees — the primary, static surface.
- Measurable defect areas: `spacing`, `type-scale`, `color`, `contrast` —
  enforced; `containers` and `decoration` — measured and reported only.
- Per-node `theme_override_*` enforcement — directly settable and undoable.

The split is not arbitrary. Enforcing `containers` or `decoration` would mean
deleting or re-parenting nodes, which collides with the inviolability of node
order below, and neither is mechanically decidable: a node that looks decorative
may be a divider (the dock's 1px `ColorRect` is exactly that), and flattening a
wrapper breaks the `$Path/To/Node` references scripts address nodes by. So they
surface findings instead.

**Out of scope:**

- Wholesale restyle ("adopt a different visual language"). That borrows a whole
  external contract for coherence and is a different mode; removing
  undisciplined values does not require it. Closed rather than deferred —
  §11.1 records the reasoning and what it would take to reopen.
- Project-wide `Theme` resource construction (see §6, gap G1).
- 2D/3D "game feel" visuals — that is the existing Game Feel surface.
- Copy, information architecture, and node order — inviolable.

---

## 4. Capability map (code-grounded)

Every link in the loop resolves to an existing Hera primitive:

| Stage | Hera primitive | Evidence | Status |
|---|---|---|---|
| Enumerate UI | `scene tree`, `node find --type Control` | `node_tool.gd:59` | ✅ |
| Read layout geometry | `game ui tree` rect x/y/w/h | `game_ui_inspector.gd:113` | ✅ (runtime) |
| Read design tokens | `node get --props theme_override_*` | `node_value_codec.gd:45` | ✅ (edited scene) |
| Read *effective* color | `eval "get_node(p).get_theme_color('font_color')"` | `eval_tool.gd:48` | ✅ read-only |
| Contrast math | skill-side WCAG on the read colors | — | ✅ values only |
| Enforce token | `node set --prop theme_override_*` (undoable) | `node_tool.gd:134` | ✅ scalars |
| Re-inspect | `node get` again (same predicate) | `node_value_codec.gd:72` | ✅ |
| Render QA | `screenshot --runtime --analyze` | `game_image_analyzer.gd:10` | ⚠️ sanity only |

**Enforcement detail** — `theme_override` value types split by
`node_value_codec.gd:8-43`:

| Token | Type | Path | Note |
|---|---|---|---|
| `theme_override_constants/separation`, `.../margin_*` | INT | `node set` | direct, undoable |
| `theme_override_font_sizes/font_size` | INT | `node set` | direct, undoable |
| `theme_override_colors/font_color` | COLOR | `node set --value "Color(r,g,b,a)"` | float components only; hex is rejected |
| `theme_override_styles/panel` (StyleBox) | OBJECT | `node set-resource --resource res://*.tres` | needs a saved `.tres` first |
| `theme` (whole Theme) | OBJECT | `node set-resource --resource res://ui/theme.tres` | needs the resource |

The shipped MVP stays inside the INT/COLOR rows — scalar tokens with no
StyleBox or Theme-resource prerequisite.

> **Reading tokens is type-scoped.** A `theme_override_*` property exists only
> on classes that define that theme item, and `node get --props` fails the whole
> read if any requested property is missing rather than returning the subset that
> exists. Inspectors must enumerate per class (`node find --type`) and request
> only that class's tokens; a single token list swept over a mixed tree is a hard
> error, not a partial result.

> **Verified against the live editor:** `node set --value` accepts only float
> variant text `Color(r, g, b, a)`. Both a bare `#hex` and `Color("#hex")` are
> rejected by the value coercion, even though `Color("#hex")` parses fine in
> GDScript and `eval`. Corpus hex must be converted (`channel = int(hh,16)/255`)
> before enforcement.

---

## 5. The Godot UI defect taxonomy

Net-new authored asset. Each area is named for the Godot construct it measures,
and each tell names a **mechanical trigger** — a single measurement, never
taste — plus a reference source.

### Area `decoration` — measure-free "delete on sight"
- **blob** — decorative `TextureRect`/`ColorRect` with no informational role
  (pure background fill) → remove.
- **glow** — gratuitous `modulate`/`self_modulate` repeated across ≥3 surfaces,
  or decorative `GradientTexture` fills → flatten.
- **emoji-icon** — emoji used as icons/bullets in `Label`/`Button` text → a real
  icon or a text label.
- **uniform-shadow** — decorative `StyleBox` `shadow_size` applied to most
  panels → remove; use figure-ground instead.

*Deletion-type tells borrow no replacement value.*

### Area `containers` — layout discipline
- **ghost-wrapper** — container-in-container contributing no layout (a
  `PanelContainer` wrapping a single `VBoxContainer` wrapping a single child) →
  flatten.
- **anchor-drift** — missing `size_flags`/anchor discipline that only fails at
  the viewport boundary → caught by `possible_clipping` in the render stage.
- **surface-per-item** — every item of a repeated series wrapped in its own
  `Panel` where a separator plus spacing would do.

### Area `spacing`
- **off-ladder** — `theme_override_constants/separation` and `margin_*` values
  that do not snap to a declared ladder. Trigger: the count of distinct spacing
  values exceeds the number of ladder rungs they map to.
- Reference: the corpus spacing ladder — multiples of the engine's 4px base
  unit. Snap to the nearest rung, ties down.
  Snapping is lateral — it never inflates macro whitespace.

### Area `type-scale`
- **unscaled** — distinct `theme_override_font_sizes/font_size` values outnumber
  the rungs of a declared type scale → collapse to the scale, preserving order
  so no two hierarchy levels merge into one rung. A collision is resolved only
  between the two levels that collide: a *peer group* that merely shares a size
  (all buttons in a row, all cells in a grid) is one level, and must not be
  pushed down to settle a collision belonging to a different pair.
- **role-confusion** — mixed `theme_override_fonts/font` with no role bijection.
- Reference: the corpus type ladder — the engine's `default_font_size` and
  heading sizes, continued above them at the engine's own 1.25 ratio.
  Hierarchy is size/weight/spacing,
  never a font swap.

### Area `color`
- **scattered-literals** — `theme_override_colors/*` set to ad-hoc `Color(...)`
  literals with no shared source → converge to one accent plus a neutral ramp.
  - **Escape (not a defect):** overrides that reference a **shared palette
    source** (named color constants or a project `Theme`) with a role bijection
    — one color ↔ one semantic role (title / body / accent / success / error) —
    are a *decided* palette. The trigger is *literal scatter with no single
    source*, never "uses overrides." See §5.1: the dock is exactly this case.
- Reference: none vendored. Convergence targets the project's *own* declared
  palette or `Theme`, never an imported one.

### Area `contrast`
- **below-wcag** — the effective `font_color` against its background `StyleBox`
  color fails WCAG body text (< 4.5:1; large text ≥ 24px or ≥ 18.66px bold uses
  3.0:1) → raise foreground lightness (same hue, higher ramp step). Never
  recolor the background.
- Objective — no taste escape.
- Reference: WCAG 2.1 SC 1.4.3.

> **Enforcement order** (dependency order): `decoration` → `containers` →
> `spacing` → `type-scale` → `color` → `contrast`. Upstream areas commit first
> so downstream conflicts self-resolve; contrast runs last because it depends on
> the final colors.

The taxonomy is deliberately small and should grow only from observed misses,
never speculatively — a rule added on a hunch produces false positives on the
first real UI it meets.

### 5.1 Validation against a real target — the Hera dock

Dry-run of the taxonomy against `main_screen_panel.gd` (the plugin's own editor
dock — an intentionally-styled, non-trivial Godot UI), before any code existed.
Values measured directly from source; contrast via WCAG relative-luminance math.

| Tell | Result | Measurement |
|---|---|---|
| `spacing/off-ladder` | 🔴 fires (true positive) | 10 distinct values `{3,4,6,10,12,14,16,22,24,28}`; 5/10 off the engine's 4px ladder (`3,6,10,14,22`). No declared scale → magic numbers. |
| `type-scale/unscaled` | 🔴 fires (true positive) | 4 `font_size` overrides `{12,17,20,32}`; ratios `1.42 / 1.18 / 1.60` (non-modular); `17,32` off-rung. |
| `contrast/below-wcag` | ✅ silent (true negative) | All text pairs pass: ICE/DEEP 15.0:1, MUTED/NIGHT 7.2:1, RED/NIGHT 4.9:1, GOLD 9.4:1, GREEN/NIGHT 11.3:1. A well-made dark UI — the objective check correctly does not fire. |
| `color/scattered-literals` | ✅ silent (correct — *refined the tell*) | 8 colors, but all are **named constants** (`HERA_ICE`, `HERA_WARM_GOLD`, …) with a role bijection (ice=title, muted=body, gold=accent, green=ok, red=error). The escape case above; a naive "uses overrides" trigger would false-positive here. |
| `decoration/blob` | ✅ silent (correct — *role qualifier*) | The 1px `ColorRect` divider has a functional role, so it is not a decorative blob. Confirms the "no informational role" qualifier is load-bearing. |
| `containers/ghost-wrapper` | 🟡 candidate | `MarginContainer → VBox(layout) → PanelContainer(shell) → …` — the `layout` VBox has one child, so its `separation:14` does nothing → foldable wrapper. |

**Second pass — the v1.1 areas.** The three areas added after the MVP were
dry-run against the same dock before shipping:

| Tell | Result | Measurement |
|---|---|---|
| `color/scattered-literals` | ✅ silent (true negative) | No near-duplicate pair among the 8 constants at the 0.04/channel threshold, *and* every one is a named constant with a single role — the trigger and the escape agree independently. |
| `containers/ghost-wrapper` | 🔴 fires on exactly 1 of 7 (true positive) | Seven single-child containers exist; only `layout` (VBox whose sole property is `separation:14`) has inert layout. `panel`/`shell_margin` apply margins, `shell` and both cards draw a `StyleBox` — correctly excluded. The "one child is not the tell" qualifier does the discriminating. |
| `decoration/blob` | ⚠️ **false positive — rule corrected** | The divider escaped as before, but the 86×86 logo `TextureRect` matched: no text, no layout dependents. The escape list had no entry for identity/branding, so it would have proposed deleting a brand mark. Escapes 2 (identity) and 3 (`tooltip_text` set) were added because of this. |

That last row is the taxonomy earning its keep a second time: the rule was wrong
in a way only a real UI exposed, and it was corrected before shipping rather
than after a user lost a logo.

**What validation changed:** two escape conditions were promoted from implicit
to explicit — `color`'s *shared-source* escape and `decoration`'s
*functional-role* escape — because the first real target is precisely the case
both must not false-positive on. The measurable tells (`spacing`, `type-scale`)
and the objective one (`contrast`) behaved correctly unchanged. The taxonomy
earned its keep against a real UI before any code existed.

---

## 6. Gaps (net-new engineering)

- **G1 — Project-wide `Theme` construction has no clean primitive.** A `Theme`'s
  data (`set_color`/`set_constant`/`set_font_size` on a type map) is
  method-based, so `resource set --prop` cannot reach it, and `eval` is a single
  non-undoable expression. The MVP avoids this by enforcing per-node overrides.
  A future `hera theme set <res://t.tres> --type Label --color font_color=…`
  would unlock palette convergence at the project level.
- **G2 — No visual regression / before-after pixel diff.** *(closed)* The
  analyzer remains a coarse whole-image heuristic (`game_image_analyzer.gd`):
  nonblank, unique colors, per-edge content ratio, `possible_clipping`,
  `low_detail`. `screenshot diff <before> <after>` now compares two captures
  directly and reports the changed pixel count, ratio, max per-channel delta and
  a bounding box locating the change. It runs locally on files already on disk,
  so it needs no editor. Note what it still is *not*: a confirmation that
  something changed and where, never a measurement of contrast, spacing or
  palette — measurement stays structural.
- **G3 — The theme-token read surface is split.** The edited scene uses
  `node get`; the running game uses `game node get` / `eval`. The checker works
  the edited scene statically and runs the game only for the render stage.
- **G4 — Effective vs override colors.** `node get theme_override_colors/...`
  returns the *override* (empty when unset), not the *rendered* color. Contrast
  checks must read the effective color via `eval get_theme_color`.

---

## 7. Pipeline

0. **Prep.** Confirm one live editor; enumerate `Control` nodes; establish the
   target spacing/type ladders once.
1. **Parallel static inspection.** One inspector per area reads only its own
   area's rules plus the live editor (`node get`, `eval`, `game ui tree`), and
   writes `findings-<area>.md` with `check` predicates. No mutation.
2. **Report.** Merge findings into one local HTML report served over
   `localhost`. Measured values are shown verbatim.
3. **Sequential enforcement** in area order. One area at a time re-measures each
   `check` from the live editor, applies the fix only where the predicate is
   currently false, and enforces with `node set` (undoable), snapping to the
   corpus. Each area commits before the next runs.
4. **Parallel re-inspection.** Fresh inspectors — not the enforcers — recompute
   the same predicates; anything still false re-enters step 3 for that area only.
5. **Render QA.** `run` + `screenshot --runtime --analyze` for a before/after
   and a clipping/blank sanity gate. The report is updated in place.

**Split of labor:** orchestration lives in the skill; measurement uses existing
CLI commands. No new CLI surface is required (see §12, open question 1).

---

## 8. Finding schema

Per area, `findings-<area>.md`, one entry each. The `check` is a Hera command
(or `eval`) returning a value a predicate can test — never a status word:

```
- id: <area>-<slug>            # e.g. spacing-off-ladder
  problem: <one line>
  evidence: <live measurement — node path + value(s)>
  fix: <mechanical change — which theme_override, snapped to which rung>
  check: <re-measurable predicate — e.g.
          `hera node get Panel/VBox --props "theme_override_constants/separation"`
          returns a value on the declared ladder>
  order: decoration|containers|spacing|type-scale|color|contrast
```

---

## 9. Reference corpus

Every value is either read from Godot's own default theme, derived from those
roots by a rule stated on the corpus page, or taken from a published
accessibility standard. Nothing is vendored from another project's design
system, so the corpus is self-contained and needs no toolchain:

- **Spacing:** multiples of the engine's 4px container base unit
  (`separation` on BoxContainer/Grid/Flow; SplitContainer contributes 12).
- **Type:** the engine's `default_font_size` (16) plus the heading sizes it
  defines (20/24/28), continued above 28 at the engine's own 1.25 step ratio so
  large sizes stay perceptually distinct.
- **Colour:** no palette is vendored. The engine defines no ramp to derive one
  from, and importing a foreign ramp would overwrite the project's design.
  Contrast repair keeps the project's hue and solves for the required
  lightness instead.
- **Contrast:** WCAG 2.1 SC 1.4.3 thresholds (4.5:1 body, 3:1 large/UI) and the
  relative-luminance formula. W3C — a human-vision standard, toolkit-independent.

A project's own theme or declared tokens win over the corpus when present.
Deletion-type tells (`decoration`, `containers`) borrow no value.

---

## 10. MVP (shipped)

Stays entirely inside the ✅ rows of §4 — no StyleBox, no Theme resource, no new
Go value coercion:

1. Enumerate `Control`s; read `theme_override_constants/separation`,
   `.../margin_*`, and `theme_override_font_sizes/font_size`; read `game ui tree`
   rects for actual sibling gaps.
2. Three checks, each a live predicate:
   - `spacing` — distinct separation/margin values vs ladder rungs.
   - `type-scale` — distinct `font_size` values vs type-scale rungs.
   - `contrast` — `eval get_theme_color('font_color')` vs background → WCAG.
3. Enforce with `node set --prop theme_override_*` (undoable).
4. Verify with a `node get` re-measure + `screenshot --runtime --analyze`.

Delivered shape: the `ui-theme-qa` skill (mirrored byte-identical into both the
Claude Code and Codex plugin trees) + a `references/ui-theme-areas.md` rule doc
+ the vendored corpus. No CLI change was required.

**Live verification.** Driven end-to-end against Godot 4.7: measure → snap
(separation 13→12, font_size 17→16, contrast 3.31→5.94:1) → enforce → re-verify
→ runtime geometry read, with `game ui tree` confirming the enforced separation
in real layout geometry. A GUI before/after render pass returned real pixels
(`nonblank`, no `possible_clipping`, unique colors 22→29) confirming token
writes reach the rendered frame.

---

## 11. Phasing

- **v1** *(done)* — MVP skill: `spacing`, `type-scale`, `contrast` on per-node
  overrides.
- **v1.1** *(done)* — `color` convergence at the node level (enforced), plus
  `containers` and `decoration` as report-only areas. Their fixes are structural
  and stay proposals; making them mutate would need an explicit opt-in flag.
- **v2** *(done)* — `hera theme get/set` (closed G1) makes project-wide `Theme`
  values reachable; `hera screenshot diff` (closed G2) compares two captures and
  locates the change.
- **later** — a wholesale restyle mode. **Closed as deliberately unbuilt**, not
  pending. See §11.1.

### 11.1 Why the restyle mode stays unbuilt

The phasing list once ended with "a wholesale restyle mode, if coherent
re-theming is wanted; needs a Godot reference matrix." Read as a backlog item it
looks like the last box to tick. It is not, and leaving it ambiguous invites
someone to build it without the context that argues against it.

**What it would be.** A canned visual contract — "Apple-like", "arcade", pick a
name — stored in this repo and applied to a project on command, replacing its
look wholesale.

**Why it does not get built:**

1. **§3 already excludes it, with the reason.** It "borrows a whole external
   contract for coherence and is a different mode; removing undisciplined values
   does not require it."
2. **It contradicts §13.** This capability "does not invent taste". Adopting a
   different visual language is exactly inventing taste, on the user's project.
3. **It reverses a decision this repo already made.** The corpus deliberately
   vendors no palette, because "importing another project's ramp would overwrite
   the project's own design". A restyle mode's whole premise is importing
   someone else's contract — the same move, much larger.
4. **The prerequisite does not exist.** A "Godot reference matrix" — a set of
   coherent visual contracts extracted from finished Godot UIs — would have to
   be built first, and building it is a separate project with its own sourcing
   and licensing questions.

**The capability is not missing.** An agent driving Hera can already restyle a
project: read the scene, decide the values, and apply them with `node set`,
`theme set` and `resource create Theme`. v2 made that materially better, since a
whole-project `Theme` can now be written instead of per-node overrides. What is
absent is only the *automation of the taste decision* — and §12's fourth open
question is precisely why: **values can be borrowed, decisions cannot.**

So the division of labour is deliberate. A human or an agent chooses the look;
`ui-theme-qa` then checks that the chosen look was applied consistently. Style
selection and style discipline are different jobs, and only the second one is
mechanical.

**If it is ever wanted anyway**, it needs, in order: the reference matrix; a
rewrite of §3 and §13 to drop "does not invent taste"; and a separate opt-in
mode, so a default run can never silently restyle a project.

---

## 12. Open questions

1. **Skill vs CLI boundary.** Should the measurement predicates live in a new
   `hera ui check` command (reusable, testable in Go) or stay in the skill /
   `eval`? A thin CLI is more durable but adds surface; the skill is faster to
   iterate. Skill-first, promote once the checks stabilize.
2. **Runtime vs edited-scene inspection.** Theme tokens read cleanly on the
   edited scene; rects read cleanly at runtime. Is a static rect estimate good
   enough to avoid a `run`, or is the render stage the only reliable geometry
   source?
3. **Corpus refresh.** Vendored static snapshot (current choice — Hera has no
   Node toolchain) vs generating at build time. The corpus doc carries a
   self-contained regeneration command for verification.
4. **Where design intent lives.** Contrast and clipping are objective; "which
   accent" and "is this density right" are decisions the tool cannot make.
   Values can be borrowed; decisions cannot. Surface those as proposals rather
   than silently picking.

---

## 13. Non-goals

- Replacing the developer's design decisions. This removes *statistical*
  defects and forces undecided values to be decided; it does not invent taste.
- Touching copy, information architecture, or node order.
- Importing web/CSS concepts. Every rule is grounded in a Godot construct —
  `Control`, `Theme`, `StyleBox`, `theme_override_*`.
