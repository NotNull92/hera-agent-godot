# UI Slop QA — area rules (SSOT, MVP: C · D · E2)

Per-area detection + enforcement contract for Godot `Control` UI. Each inspector
/ enforcer reads **only its own area section** (context isolation — no single
context holds every rule). What counts as a replacement value comes from
[reference-corpus.md](reference-corpus.md); this file is the judgement layer on
top.

Common principle — **"agents statistically default like this → force it
mechanically like that."** A single measurement triggers each tell, never taste.
Values are snapped to the corpus, never invented.

All measurement uses existing `hera` commands (no custom tooling):
- Enumerate: `hera --ids scene tree` / `hera node find --type <Class>`
- Read tokens: `hera node get <path> --props "<a>,<b>"`
- Read effective color: `hera eval "get_node('<path>').get_theme_color('font_color')"`
- Read geometry (runtime): `hera game ui tree --fields rect,type,path`
- Enforce (undoable): `hera node set <path> --prop "<token>" --value <v>`
- Verify render: `hera screenshot --runtime --analyze`

Godot theme-token property paths used below:
`theme_override_constants/separation`, `theme_override_constants/margin_left`
(`margin_top|right|bottom`), `theme_override_font_sizes/font_size`,
`theme_override_colors/font_color`.

> **Enforcement order** (dependency order): C → D → E. Spacing commits before
> type before color so downstream reads the already-changed state. (Areas A/B
> from the design doc are out of MVP scope.)

---

## Area C · spacing (enforcement order 1)

**Agent statistic:** spacing constants are magic numbers with no declared
ladder — a spread of near-but-unequal values (`3, 6, 10, 14, 22 …`) instead of
a scale.

**Mechanical trigger:** collect every `separation` and `margin_*` override
across the Control tree. If the count of **distinct** spacing values exceeds the
count of corpus rungs they map to — i.e. two or more distinct values snap to the
same rung, or values sit off-ladder — the ladder is undisciplined.

**Fix:** snap each spacing value to the nearest corpus rung (ties → smaller).
Do not scale macro whitespace up; snapping is lateral, not inflation.

**Escape (not slop):** values that are already all on the corpus ladder and
self-consistent. A single off-ladder value that is a deliberate optical tweak on
one focal element is a proposal, not an automatic fix.

**check:** for each changed node, `hera node get <path> --props
"theme_override_constants/separation"` (and `margin_*`) returns a value present
in the corpus spacing ladder. Predicate = *every distinct spacing token ∈
ladder*.

## Area D · type scale (enforcement order 2)

**Agent statistic:** `font_size` overrides are a random spread with no modular
relationship (ratios like `1.42 / 1.18 / 1.60`) and off-rung values (`17`, odd
sizes).

**Mechanical trigger:** collect every `theme_override_font_sizes/font_size`.
If distinct sizes are not all on the corpus type scale, or two hierarchy levels
would collapse to one rung, the scale is undisciplined.

**Fix:** snap each size to the nearest corpus rung, **preserving order** — if two
adjacent levels round to the same rung, push the smaller one down a rung so the
hierarchy survives. Hierarchy is size/weight/spacing, never font swaps.

**Escape:** sizes already all on the scale with a monotonic hierarchy.

**check:** `hera node get <path> --props
"theme_override_font_sizes/font_size"` returns a value in the corpus type scale;
distinct sizes remain strictly ordered by their prior hierarchy.

## Area E2 · contrast (enforcement order 3)

**Agent statistic:** text color chosen for looks, contrast against its surface
ignored.

**Mechanical trigger:** for each text Control, read the **effective** font color
and its background surface color, compute WCAG contrast. Fail if below the
corpus threshold for that text's size (body 4.5:1; large ≥24px or ≥18.66px bold
3.0:1).

- Effective font color: `hera eval "get_node('<path>').get_theme_color('font_color')"`
  (the override property alone can be empty while the theme still paints it).
- Background: the nearest ancestor with a painted `StyleBox` — read via
  `hera eval "get_node('<panel>').get_theme_stylebox('panel').bg_color"`, else
  the effective panel/root background.

**Fix:** raise foreground lightness first (same hue → a higher Radix step) until
the pair passes. Do not recolor the background.

**Escape:** pair already ≥ threshold. This check is objective — no taste escape.

**check:** recomputed WCAG ratio for the pair ≥ the corpus threshold for that
text size.

---

## Finding schema (all areas)

Each inspector writes `findings-<area>.md`, one entry per finding. The `check`
is a **re-measurable predicate** (a `hera` command + comparison), never a status
word — enforcers and re-inspectors recompute it from the live editor every time.

```
- id: <area>-<slug>            # e.g. C-unscaled-separation
  problem: <one line>
  evidence: <live measurement — node path + value(s) read via hera>
  fix: <mechanical change — which theme_override token, snapped to which rung/hex>
  check: <hera command + predicate that returns true/false from source>
  order: C|D|E
```

**Checklist = eval function.** Never record "done" as text. An enforcer applies
the fix only where `check` is currently false; a fresh re-inspector recomputes
the same `check`. This is what blocks the "trust the earlier note and silently
skip" failure.

## Inviolable

Copy, information architecture, and node order are never changed — only theme
tokens. Snapping spacing/type/color is not a redesign.
