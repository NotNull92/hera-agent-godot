# UI Slop QA — design proposal

> A Godot-native design-QA capability for Hera that detects and removes
> "AI-slop" in Control-node UI: undisciplined spacing, unscaled type,
> ad-hoc colors, and low contrast. It adapts the *architecture* of the
> `slopslap` Claude Code plugin (checklist-as-eval-function, area isolation,
> snap-to-reference) to Godot, reusing Hera's existing inspect/mutate/QA
> primitives instead of a browser.

**Status:** design proposal — nothing here is implemented yet. This document
records the feasibility survey and the intended shape so Claude Code and Codex
share one plan. Capability claims cite code paths (`file:line`) that were read
during the survey; taxonomy and pipeline sections are proposals to critique.

---

## 1. Motivation

Agent-generated Godot UI drifts toward the same statistical defaults that
`slopslap` documents for the web: magic-number spacing, a random spread of font
sizes, per-node color overrides with no shared palette, and text that fails
contrast. Hera can already *see* and *change* a live editor, but it has no
notion of "this UI is undisciplined" — `game qa diagnose` only checks render
sanity (errors, truncation, clipping, blank frames;
`cmd/game_qa_diagnose_checks.go`).

`slopslap` (analysed separately) is the closest existing art, but it is
**web-bound**: it detects CSS gradients / glassmorphism / `background-clip:text`,
scans `.js/.css/.vue`, and enforces by editing CSS. None of that transfers to
Godot's `Control` + `Theme` model. What *does* transfer is its design
philosophy and its reference **values** (numbers and hex are domain-neutral).

This proposal keeps the transferable parts and rebuilds the domain-specific
parts natively.

---

## 2. Principles inherited from slopslap

These are the load-bearing ideas, restated for Godot:

1. **Checklist = eval function.** A finding is never a stored "done/not-done"
   status. It carries a `check` predicate that is **re-measured from the live
   editor every time** (inspect, enforce, re-inspect). This blocks the
   "trust the earlier note and silently skip" failure. Hera already applies
   this pattern in `game qa diagnose`, where each check re-evaluates live data.
2. **Snap, don't invent.** Replacement values (spacing rungs, font-size scale,
   palette, contrast target) come from a fixed reference corpus, not from the
   model's imagination. The corpus is the same real-source data slopslap
   generates — Tailwind spacing (px), Radix color ramps (hex), WCAG thresholds —
   because those are just numbers and apply to Godot unchanged.
3. **Inspect is static and parallel; enforce is sequential and undoable.**
   Reading Control geometry / theme tokens is side-effect-free and can run in
   any order. Mutations register with `EditorUndoRedoManager`
   (`node_tool.gd:150`) so the developer keeps Ctrl+Z.
4. **Thin orchestrator, isolated areas.** A Claude Code skill drives the
   pipeline and holds only orchestration; the per-area rules live in a
   reference doc, and per-area findings live in files. No single context holds
   all rules at once (the capacity-drop failure slopslap exists to prevent).
5. **Render is confirmation, not measurement.** Design properties are measured
   structurally (rects, theme tokens, contrast math). The screenshot analyzer
   is a sanity/clipping gate only — matching slopslap's "static inspection,
   browser only for the final before/after render."

---

## 3. Scope

**In scope (v1, reductive only):**

- Edited-scene `Control` trees (the primary, static surface).
- Four measurable defect areas: spacing, type scale, color/palette, contrast.
- Per-node `theme_override_*` enforcement (directly settable + undoable).

**Out of scope (v1):**

- `transform` mode (slopslap's Plan B). It borrows a whole external site's
  contract for a coherent restyle; it is a separate, later mode and is not
  needed to remove slop.
- Project-wide `Theme` resource construction (see §6 gap G1).
- 2D/3D "game feel" visuals — that is the existing Game Feel surface, not this.
- Copy/content/order changes — inviolable, exactly as in slopslap.

---

## 4. Capability map (code-grounded)

Every link in the loop resolves to an existing Hera primitive. Gaps are called
out in §6.

| Stage | slopslap (web) | Hera primitive | Evidence | Status |
|---|---|---|---|---|
| Enumerate UI | DOM walk | `scene tree`, `node find --type Control` | `node_tool.gd:59` | ✅ |
| Read layout geometry | `getBoundingClientRect` | `game ui tree` rect x/y/w/h | `game_ui_inspector.gd:112` | ✅ (runtime) |
| Read design tokens | `getComputedStyle` | `node get --props theme_override_*` | `node_value_codec.gd:45` | ✅ (edited scene) |
| Read *effective* color | computed color | `eval "get_node(p).get_theme_color('font_color')"` | `eval_tool.gd:33` | ✅ read-only |
| Contrast math | JS on hex | CLI/skill-side WCAG on read colors | — | ✅ values only |
| Enforce token | edit CSS | `node set --prop theme_override_* ` (undoable) | `node_tool.gd:134` | ✅ scalars |
| Re-inspect | re-read source | `node get` again (same predicate) | `node_value_codec.gd:72` | ✅ |
| Render QA | Playwright before/after | `screenshot --runtime --analyze` | `game_image_analyzer.gd:10` | ⚠️ sanity only |

**Enforcement detail** — `theme_override` value types split by
`node_value_codec.gd:8-43`:

| Token | Type | Path | Note |
|---|---|---|---|
| `theme_override_constants/separation`, `.../margin_*` | INT | `node set` | direct, undoable |
| `theme_override_font_sizes/font_size` | INT | `node set` | direct, undoable |
| `theme_override_colors/font_color` | COLOR | `node set --value "Color(r,g,b,a)"` | direct (str_to_var), undoable |
| `theme_override_styles/panel` (StyleBox) | OBJECT | `node set-resource --resource res://*.tres` | needs a saved `.tres` first |
| `theme` (whole Theme) | OBJECT | `node set-resource --resource res://ui/theme.tres` | needs the resource |

v1 stays inside the INT/COLOR rows — the scalar tokens that map 1:1 to
slopslap's "replace this value" enforcement, with no StyleBox/Theme resource
prerequisite.

---

## 5. The Godot UI slop taxonomy (v0 DRAFT)

Net-new asset — this is what has to be *authored*, not ported. Areas mirror
slopslap's A–E but are re-grounded in Godot constructs. Each item names a
mechanical trigger (single measurement, not taste) and a reference source.

### Area A — decorative removal (measure-free "delete on sight")
Godot's decorative slop is thinner than the web's (no CSS gradients/glass), but
the reflexive-decoration pattern recurs:
- **A1** decorative `TextureRect`/`ColorRect` with no informational role
  (pure background blobs) → remove.
- **A2** gratuitous `modulate`/`self_modulate` "glow" repeated across ≥3
  surfaces, or decorative `GradientTexture` fills → flatten.
- **A3** emoji used as icons/bullets in `Label`/`Button` text → real icon or
  text label.
- **A4** decorative `StyleBox` shadow (`shadow_size`) applied uniformly to most
  panels → remove; use figure-ground instead.

*Deletion-type tells: no borrowed value (like slopslap's `applies:false`).*

### Area B — layout / container discipline
- **B1** container-in-container with no layout contribution (a `PanelContainer`
  wrapping a single `VBoxContainer` wrapping a single child) → flatten.
  Mirrors slopslap's ghost-wrapper / box-in-box rule.
- **B2** missing `size_flags`/anchor discipline that only fails at the viewport
  boundary → caught by `possible_clipping` in the render stage.
- **B3** surface-per-item overuse: each item of a repeated series wrapped in its
  own `Panel` when a separator + spacing would do.

### Area C — spacing
- **C1** `theme_override_constants/separation` and `margin_*` values that do not
  snap to a declared ladder. Trigger: count of distinct spacing values used
  exceeds the number of ladder rungs they map to.
- Reference: Tailwind spacing scale (px). Snap each value to the nearest rung.

### Area D — type scale
- **D1** distinct `font_size` values (via `theme_override_font_sizes/font_size`)
  outnumber the rungs of a declared type scale → collapse to the scale.
- **D2** decorative font-family role confusion — a family used for random roles
  (Godot: mixed `theme_override_fonts/font` with no role bijection).
- Reference: Tailwind fontSize scale (px).

### Area E — color / contrast
- **E1** *scattered literal* colors — `theme_override_colors/*` set to ad-hoc
  `Color(...)` literals with no shared source → converge to one accent + a
  neutral ramp.
  - **Escape (not slop):** overrides that reference a **shared palette source**
    (named color constants or a project `Theme`) with a role bijection
    (one color ↔ one semantic role: title / body / accent / success / error)
    are a *decided* palette, not slop — exactly slopslap's "a decided value is
    not slop even if common." The trigger is *literal scatter / no single
    source*, never "uses overrides." (See §5.1 — the dock is the escape case.)
- **E2** effective `font_color` vs its background `StyleBox` color fails
  WCAG body text (< 4.5:1) → re-map lightness. Objective, no escape.
- Reference: Radix color ramps (hex) for E1; WCAG 2.1 SC 1.4.3 for E2.

> **Enforcement order** (from slopslap, dependency order): A → B → C → D → E.
> Upstream commits first so downstream conflicts self-resolve.

This taxonomy is deliberately small and should grow only from observed misses,
never speculatively (slopslap's overfit guard).

### 5.1 Validation against a real target — the Hera dock

Dry-run of the v0 taxonomy against `main_screen_panel.gd` (the plugin's own
editor dock — an intentionally-styled, non-trivial Godot UI). Values measured
directly from source; contrast via WCAG relative-luminance math.

| Tell | Result | Measurement |
|---|---|---|
| **C1** spacing | 🔴 fires (true positive) | 10 distinct spacing values `{3,4,6,10,12,14,16,22,24,28}`; 5/10 off a Tailwind ladder (`3,6,10,14,22`). No declared scale → magic numbers. |
| **D1** type | 🔴 fires (true positive) | 4 `font_size` overrides `{12,17,20,32}`; ratios `1.42 / 1.18 / 1.60` (non-modular); `17,32` off-rung. |
| **E2** contrast | ✅ silent (true negative) | All text pairs pass: ICE/DEEP 15.0:1, MUTED/NIGHT 7.2:1, RED/NIGHT 4.9:1, GOLD 9.4:1, GREEN/NIGHT 11.3:1. A well-made dark UI — objective check correctly does not fire. |
| **E1** palette | ✅ silent (correct — *refined the tell*) | 8 colors, but all are **named constants** (`HERA_ICE`, `HERA_WARM_GOLD`, …) with a role bijection (ice=title, muted=body, gold=accent, green=ok, red=error). This is the escape case above; naive "uses overrides" would false-positive here. |
| **A1** decoration | ✅ silent (correct — *role qualifier*) | The 1px `ColorRect` divider has a functional role, so it is not a decorative blob. Confirms the "no informational role" qualifier is load-bearing. |
| **B1** nesting | 🟡 candidate | `MarginContainer → VBox(layout) → PanelContainer(shell) → …` — the `layout` VBox has one child, so its `separation:14` does nothing → foldable wrapper. |

**What validation changed:** two escape conditions were promoted from implicit
to explicit — E1's *shared-source* escape and A's *functional-role* escape —
because the first real target is precisely a case both must not false-positive
on. The core measurable tells (C1, D1) and the objective one (E2) behaved
correctly with no change. This is the taxonomy earning its keep against a real
UI before any code exists.

---

## 6. Gaps (net-new engineering, not port)

- **G1 — Project-wide `Theme` construction has no clean primitive.**
  A `Theme`'s data (`set_color`/`set_constant`/`set_font_size` on a type map) is
  method-based, so `resource set --prop` cannot reach it, and `eval` is a single
  non-undoable expression. v1 avoids this by enforcing per-node overrides.
  A future `hera theme set <res://t.tres> --type Label --color font_color=...`
  tool would unlock Area-E palette convergence at the project level.
- **G2 — No visual regression / before-after pixel diff.** The analyzer is a
  coarse whole-image heuristic (`game_image_analyzer.gd`): nonblank, unique
  colors, per-edge content ratio, `possible_clipping`, `low_detail`. Good as a
  render/clipping gate; it does not measure contrast, spacing, or palette
  in-image. Measurement stays structural.
- **G3 — Theme-token read surface is split.** Edited scene uses `node get`;
  running game uses `game node get` / `eval`. A checker mostly works the edited
  scene (static, like slopslap's static inspection), running the game only for
  the render QA stage.
- **G4 — Effective vs override colors.** `node get theme_override_colors/...`
  returns the *override* (empty if unset), not the *rendered* color. Contrast
  checks must read the effective color via `eval get_theme_color`.

---

## 7. Pipeline (adapted, live-editor variant)

slopslap runs 5 stages against static files + one browser render. Hera's live
editor lets the loop tighten, but the stage roles are preserved:

0. **Prep + single upstream judgment.** Enumerate `Control` nodes; abstract the
   content into repeated-series vs one-off (slopslap's content-constantization,
   feeds Area B). Establish the target spacing/type ladders once.
1. **Parallel static inspection.** Per-area subagents read only their area's
   rules + the live editor (`node get`, `game ui tree`), each writing
   `findings-<X>.md` with `check` predicates. No mutation.
2. **Report.** Merge findings into one local HTML report served over
   `localhost` (identical to slopslap stage 2). Values shown verbatim.
3. **Sequential enforcement A→B→C→D→E.** One area agent at a time re-measures
   each `check` from the live editor; applies the fix only where the predicate
   is false; uses `node set` (undoable). Snap replacement values to the corpus.
4. **Parallel re-inspection.** Fresh agents re-evaluate the same predicates
   (`node get`); any still-false item re-enters stage 3 for that area only.
5. **Render QA.** `run` + `screenshot --runtime --analyze` for a before/after and
   a clipping/blank sanity gate. Report is updated in place.

**Split of labor:** the orchestration lives in a Claude Code **skill**; the
only *new CLI surface* needed is thin measurement help (see §9). This mirrors
slopslap's "skill orchestrates, scripts measure" division.

---

## 8. Finding schema

Per area, `findings-<area>.md`, one entry each — the `check` is a Hera command
(or `eval`) that returns a value a predicate can test, never a status word:

```
- id: <area>-<slug>            # e.g. C-unscaled-separation
  problem: <one line>
  evidence: <live measurement — node path + value(s)>
  fix: <mechanical change — which theme_override, snapped to which rung>
  check: <re-measurable predicate — e.g.
          `hera node get Panel/VBox --props "theme_override_constants/separation"`
          returns a value on the declared ladder>
  order: A|B|C|D|E
```

---

## 9. Reference corpus

Reuse slopslap's generator idea: values come from real packages, not
hand-typed constants.

- **Spacing / type:** Tailwind `defaultTheme` spacing + fontSize (px). MIT.
- **Palette:** Radix color ramps (hex), accent + neutral. MIT.
- **Contrast:** WCAG 2.1 SC 1.4.3 thresholds (4.5:1 body, 3:1 large/UI). W3C.

These are numeric/hex and apply to Godot unchanged. The corpus ships vendored
(offline). Deletion-type tells (Area A, B) borrow no value.

---

## 10. MVP (what to build first)

Stays entirely inside the ✅ rows of §4 — no StyleBox, no Theme resource, no new
Go value-coercion:

1. Enumerate `Control`s; read `theme_override_constants/separation`,
   `.../margin_*`, `theme_override_font_sizes/font_size`; read `game ui tree`
   rects for actual sibling gaps.
2. Three checks, each a live predicate:
   - **C**: distinct separation/margin values vs ladder rungs.
   - **D**: distinct `font_size` values vs type-scale rungs.
   - **E2**: `eval get_theme_color('font_color')` vs background color → WCAG.
3. Enforce with `node set --prop theme_override_*` (undoable).
4. Verify with `node get` re-measure + `screenshot --runtime --analyze`.

Deliverable shape: one Claude Code skill (`ui-slop-qa`) + a
`references/ui-slop-areas.md` rule doc + the vendored corpus. No CLI change is
strictly required for the MVP; a later `hera ui check` could fold the
measurement helpers in.

---

## 11. Phasing

- **v1** — MVP skill (areas C, D, E2) on per-node overrides.
- **v1.1** — areas A, B (deletion/flatten) + Area-E1 palette convergence at the
  node level.
- **v2** — `hera theme set` tool (closes G1) → project-wide `Theme` convergence;
  optional before/after pixel diff (closes G2).
- **later** — `transform` mode, if a coherent restyle (not just de-slop) is
  wanted; requires a Godot reference matrix analogous to slopslap's Plan B.

---

## 12. Open questions

1. **Skill vs CLI boundary.** Should the measurement predicates live in a new
   `hera ui check` command (reusable, testable in Go) or stay in the skill /
   `eval`? A thin CLI is more durable but adds surface; the skill is faster to
   iterate. Leaning skill-first, promote to CLI once the checks stabilize.
2. **Runtime vs edited-scene inspection.** Theme tokens read cleanly on the
   edited scene; rects read cleanly at runtime. Is a static rect estimate from
   the edited scene good enough to avoid a `run`, or is the render stage the
   only reliable geometry source?
3. **Corpus vendoring.** Generate at build time (like slopslap's
   `gen-reference-data.mjs`) or vendor a static snapshot? Hera has no Node
   toolchain today; a static JSON snapshot avoids adding one.
4. **Where design intent lives.** Contrast and clipping are objective; "which
   accent" and "is this the right density" are decisions the tool cannot make
   (slopslap's own ceiling: "values can be borrowed, decisions cannot"). The
   tool should surface those as proposals, not silently pick.

---

## 13. Non-goals

- Replacing the developer's design decisions. This removes *statistical*
  defects and forces the undecided values to be decided; it does not invent
  taste.
- Touching copy, information architecture, or node order.
- Any web/CSS concept. This is a Godot-native rebuild, not a slopslap port.
