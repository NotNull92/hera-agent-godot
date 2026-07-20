# UI Theme QA — area rules (SSOT, MVP: spacing · type-scale · contrast)

Per-area detection + enforcement contract for Godot `Control` UI. Each inspector
/ enforcer reads **only its own area section** (context isolation — no single
context holds every rule). What counts as a replacement value comes from
[reference-corpus.md](reference-corpus.md); this file is the judgement layer on
top.

Common principle — **an undecided value is a defect; a decided one is not.**
Every tell fires on a single measurement, never on taste, and every replacement
is snapped to the corpus rather than invented.

All measurement uses existing `hera` commands (no custom tooling):
- Enumerate: `hera --ids scene tree` / `hera node find --type <Class>`
- Read tokens: `hera node get <path> --props "<a>,<b>"`
  (**type-aware — see below**)
- Read effective color: `hera eval "get_node('<path>').get_theme_color('font_color')"`
- Read geometry (runtime): `hera game ui tree --fields rect,type,path`
- Enforce (undoable): `hera node set <path> --prop "<token>" --value <v>`
- Verify render: `hera screenshot --runtime --analyze`

Godot theme-token property paths used below:
`theme_override_constants/separation`, `theme_override_constants/margin_left`
(`margin_top|right|bottom`), `theme_override_font_sizes/font_size`,
`theme_override_colors/font_color`.

> **Read tokens by node type — never sweep one token list over the whole tree.**
> A theme-override property only exists on nodes whose class defines that theme
> item, and `node get --props` **fails the entire read** if *any* requested
> property is missing on that node — it does not return the subset that exists.
> Asking a `Label` for `.../separation`, or a `HBoxContainer` for `.../margin_*`,
> is a hard error, so a naive whole-tree sweep breaks on any real mixed UI.
>
> Enumerate per class first, then request only the tokens that class defines:
>
> | token | classes that define it |
> |---|---|
> | `constants/separation` | `BoxContainer` (`HBoxContainer`, `VBoxContainer`), `SplitContainer`, `FlowContainer` |
> | `constants/h_separation`, `constants/v_separation` | `GridContainer`, and `h_separation` on `Button`/`Tree`/`ItemList` |
> | `constants/margin_*` | `MarginContainer` only |
> | `font_sizes/font_size`, `colors/font_color` | text controls — `Label`, `Button`, `LineEdit`, … |
>
> ```bash
> hera node find --type VBoxContainer     # then --props ".../separation"
> hera node find --type GridContainer     # then --props ".../h_separation,.../v_separation"
> ```
>
> A bare `hera node get <path>` (no `--props`) returns every property and is
> failure-proof, but costs ~100 properties per node — use it only to discover
> what a class supports, never for the sweep itself.

> **Enforcement order** (dependency order): `spacing` → `type-scale` →
> `contrast`. Spacing commits before type-scale, and contrast runs last because
> it depends on the final colors. (Areas `decoration`, `containers`, and
> `color` from the design doc are out of MVP scope.)

---

## Area `spacing` (enforcement order 1)

**Statistical default:** spacing constants are magic numbers with no declared
ladder — a spread of near-but-unequal values (`3, 6, 10, 14, 22 …`) instead of
a scale.

**Mechanical trigger:** collect every `separation` and `margin_*` override,
reading **per container class** (see the type table above — `separation` from
`BoxContainer`s, `h/v_separation` from `GridContainer`s, `margin_*` from
`MarginContainer`s only). If the count of **distinct** spacing values exceeds the
count of corpus rungs they map to — i.e. two or more distinct values snap to the
same rung, or values sit off-ladder — the ladder is undisciplined.

**Fix:** snap each spacing value to the nearest corpus rung (ties → smaller).
Do not scale macro whitespace up; snapping is lateral, not inflation.

**Escape (not a defect):** values that are already all on the corpus ladder and
self-consistent. A single off-ladder value that is a deliberate optical tweak on
one focal element is a proposal, not an automatic fix.

**check:** for each changed node, `hera node get <path> --props
"theme_override_constants/separation"` (and `margin_*`) returns a value present
in the corpus spacing ladder. Predicate = *every distinct spacing token ∈
ladder*.

## Area `type-scale` (enforcement order 2)

**Statistical default:** `font_size` overrides are a random spread with no
modular relationship (ratios like `1.42 / 1.18 / 1.60`) and off-rung values
(`17`, odd sizes).

**Mechanical trigger:** collect every `theme_override_font_sizes/font_size`.
If distinct sizes are not all on the corpus type scale, or two hierarchy levels
would collapse to one rung, the scale is undisciplined.

**Fix:** snap each size to the nearest corpus rung, **preserving order** — if two
sizes that express a hierarchy round to the same rung, push the smaller one down
a rung so the hierarchy survives. Hierarchy is size/weight/spacing, never font
swaps.

> **Resolve a collision only between the two levels that collide.** Group the
> nodes before snapping: a *hierarchy chain* is a set of text nodes whose sizes
> rank them against each other (title vs body vs caption in one panel), while a
> *peer group* is a set of controls that intentionally share one size (every
> button in a row, every cell in a grid). Snap each group to the nearest rung;
> when a collision forces one level down, move **only the colliding hierarchy
> level**, never a peer group that merely happened to share that size.
>
> Worked example — sizes `{42 cells, 22 title, 17 score, 15 status, 15 buttons}`:
> `17` and `15` both snap to `16`, so the *status* level drops to `12`. The
> buttons are a peer group, not the level below `score`, so they stay at `16`.
> Dragging them down too would shrink interactive labels for a collision that
> was never theirs.

**Escape:** sizes already all on the scale with a monotonic hierarchy.

**check:** `hera node get <path> --props
"theme_override_font_sizes/font_size"` returns a value in the corpus type scale;
distinct sizes remain strictly ordered by their prior hierarchy.

## Area `contrast` (enforcement order 3)

**Statistical default:** text color chosen for looks, contrast against its
surface never checked.

**Mechanical trigger:** for each text Control, read the **effective** font color
and its background surface color, compute WCAG contrast. Fail if below the
corpus threshold for that text's size (body 4.5:1; large ≥24px or ≥18.66px bold
3.0:1).

- Effective font color: `hera eval "get_node('<path>').get_theme_color('font_color')"`
  (the override property alone can be empty while the theme still paints it).
- Background: the nearest ancestor with a painted `StyleBox` — read via
  `hera eval "get_node('<panel>').get_theme_stylebox('panel').bg_color"`, else
  the effective panel/root background.

**Fix:** keep the text colour's hue and saturation and **solve** for the
lightness that meets the threshold — the corpus gives the exact bound
(`L_text >= T*(L_bg+0.05)-0.05` to lighten). Do not guess a colour and do not
recolour the background. Enforce with `node set --value "Color(r, g, b, a)"`
(float 0..1) — the CLI rejects bare `#hex` and `Color("#hex")`.

**Escape:** pair already ≥ threshold. This check is objective — no taste escape.

**check:** recomputed WCAG ratio for the pair ≥ the corpus threshold for that
text size.

---

## Finding schema (all areas)

Each inspector writes `findings-<area>.md`, one entry per finding. The `check`
is a **re-measurable predicate** (a `hera` command + comparison), never a status
word — enforcers and re-inspectors recompute it from the live editor every time.

```
- id: <area>-<slug>            # e.g. spacing-off-ladder
  problem: <one line>
  evidence: <live measurement — node path + value(s) read via hera>
  fix: <mechanical change — which theme_override token, snapped to which rung/hex>
  check: <hera command + predicate that returns true/false from source>
  order: spacing|type-scale|contrast
```

**A check is a predicate, not a status.** Never record "done" as text. An
enforcer applies the fix only where `check` is currently false; a fresh
re-inspector recomputes the same `check`. This is what blocks the "trust the
earlier note and silently skip" failure.

## Inviolable

Copy, information architecture, and node order are never changed — only theme
tokens. Snapping spacing/type/color is not a redesign.
