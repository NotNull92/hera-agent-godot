# UI Slop QA — reference corpus (vendored snapshot)

Replacement values are **snapped to real design-system constants**, not
invented. This file is a vendored, offline snapshot of published values with
provenance, so the skill needs no toolchain at run time. Values are numeric /
hex and apply to Godot `Control` theme tokens unchanged.

## Sources

| key | source | license | provenance |
|---|---|---|---|
| `tailwind` | tailwindcss@3.4.19 `defaultTheme` | MIT | npm package `spacing` / `fontSize` (rem→px ×16) |
| `radix` | @radix-ui/colors@3.0.0 | MIT | npm package 12-step ramps |
| `wcag` | WCAG 2.1 SC 1.4.3 | W3C standard | spec constants |

**Regenerate / verify** (no repo toolchain; run anywhere with Node):

```bash
npm i tailwindcss@3.4.19 @radix-ui/colors@3.0.0
node -e 'const t=require("tailwindcss/defaultTheme");
  console.log([...new Set(Object.values(t.spacing).map(v=>{const m=/^([0-9.]+)rem$/.exec(v);return m?+(parseFloat(m[1])*16).toFixed(2):null}).filter(x=>x&&x<=256))].sort((a,b)=>a-b));
  console.log(Object.entries(t.fontSize).map(([k,v])=>[k,+(parseFloat(Array.isArray(v)?v[0]:v)*16)]));'
```

If a snapshot value ever disagrees with the packages above, the packages win —
re-vendor from them.

---

## C — spacing ladder (px)

`tailwindcss@3.4.19` spacing scale (rem→px, ≤256). Snap each spacing token
(`theme_override_constants/separation`, `.../margin_*`) to the nearest rung
(ties → smaller). The UI-relevant band is roughly `[2 … 96]`.

```
2, 4, 6, 8, 10, 12, 14, 16, 20, 24, 28, 32, 36, 40, 44, 48,
56, 64, 80, 96, 112, 128, 144, 160, 176, 192, 208, 224, 240, 256
```

## D — type scale (px)

`tailwindcss@3.4.19` fontSize scale (rem→px). Snap each
`theme_override_font_sizes/font_size` to the nearest rung; preserve ordering
(never collapse two hierarchy levels into one rung).

| name | px |
|---|---|
| xs | 12 |
| sm | 14 |
| base | 16 |
| lg | 18 |
| xl | 20 |
| 2xl | 24 |
| 3xl | 30 |
| 4xl | 36 |
| 5xl | 48 |
| 6xl | 60 |
| 7xl | 72 |
| 8xl | 96 |
| 9xl | 128 |

## E — palette (hex) + contrast

`@radix-ui/colors@3.0.0`, 12-step ramps. Converge a scattered palette to **one
accent + the neutral ramp**.

**Applying a hex to a Godot token.** In GDScript / `hera eval`, `Color("#0090ff")`
parses hex directly. The CLI `node set --value` coercion does **not**: it rejects
both `#0090ff` and `Color("#0090ff")` and accepts only float variant text
`Color(r, g, b, a)` (0..1). So when enforcing E2 via `node set`, convert the
corpus hex to floats first — each channel `= int(hh, 16) / 255` — e.g.
`#0090ff` → `Color(0, 0.565, 1, 1)`.

**Step convention (Radix):** 1–2 = app/subtle background · 3–5 = component
background · 6–8 = borders · **9–10 = solid accent fill** · **11 = low-contrast
text** · **12 = high-contrast text**. Radix guarantees step-11 text on a step-2
background ≥ 4.5:1.

**Neutral ramp — `slate` (steps 1→12):**

```
#fcfcfd #f9f9fb #f0f0f3 #e8e8ec #e0e1e6 #d9d9e0
#cdced6 #b9bbc6 #8b8d98 #80838d #60646c #1c2024
```

**Accent ramps (pick ONE; steps 1→12):**

- `blue`  — `#fbfdff #f4faff #e6f4fe #d5efff #c2e5ff #acd8fc #8ec8f6 #5eb1ef #0090ff #0588f0 #0d74ce #113264`
- `green` — `#fbfefc #f4fbf6 #e6f6eb #d6f1df #c4e8d1 #adddc0 #8eceaa #5bb98b #30a46c #2b9a66 #218358 #193b2d`
- `amber` — `#fefdfb #fefbe9 #fff7c2 #ffee9c #fbe577 #f3d673 #e9c162 #e2a336 #ffc53d #ffba18 #ab6400 #4f3422`
- `red`   — `#fffcfc #fff7f7 #feebec #ffdbdc #ffcdce #fdbdbe #f4a9aa #eb8e90 #e5484d #dc3e42 #ce2c31 #641723`

Reserve accent hues for real semantic roles only (brand/CTA, and existing
error/warning/success states). Everything else → the neutral ramp.

## WCAG thresholds (contrast)

WCAG 2.1 SC 1.4.3. Contrast ratio `(L1+0.05)/(L2+0.05)`, L = relative luminance.

| target | min ratio |
|---|---|
| body text | 4.5 : 1 |
| large text (≥ 24px, or ≥ 18.66px bold) | 3.0 : 1 |
| UI component / graphical object | 3.0 : 1 |

Fix a failing pair by adjusting **foreground lightness first** (same hue → a
higher Radix step), not by recoloring the background.
