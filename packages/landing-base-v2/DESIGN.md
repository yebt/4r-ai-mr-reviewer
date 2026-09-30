---
name: 4R Landing (landing-base-v2)
description: The path of one MR across the operator's infrastructure boundary — a Persuade surface built from spa-b's own token world.
colors:
  accent: "hsl(233 55% 52%)"
  accent-hover: "hsl(234 58% 46%)"
  accent-text: "hsl(235 58% 45%)"
  bg-app: "hsl(230 20% 99%)"
  bg-panel: "hsl(230 20% 97.5%)"
  line: "hsl(224 12% 83%)"
  line-strong: "hsl(222 11% 74%)"
  text: "hsl(224 22% 13%)"
  text-muted: "hsl(222 10% 39%)"
  success-solid: "hsl(152 55% 33%)"
  warning-solid: "hsl(36 92% 45%)"
  danger-solid: "hsl(356 72% 48%)"
  info-solid: "hsl(200 80% 38%)"
typography:
  h1:
    fontFamily: "system-ui, -apple-system, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif"
    fontSize: "clamp(2.125rem, 1.4rem + 3.2vw, 3.5rem)"
    fontWeight: 600
    lineHeight: 1.08
    letterSpacing: "-0.03em"
  h2:
    fontFamily: "{typography.h1.fontFamily}"
    fontSize: "clamp(1.5rem, 1.1rem + 1.6vw, 2.125rem)"
    fontWeight: 600
    lineHeight: 1.18
    letterSpacing: "-0.02em"
  h3:
    fontFamily: "{typography.h1.fontFamily}"
    fontSize: "1.0625rem"
    fontWeight: 600
    lineHeight: 1.4
  lede:
    fontFamily: "{typography.h1.fontFamily}"
    fontSize: "1.1875rem"
    lineHeight: 1.55
  body:
    fontFamily: "{typography.h1.fontFamily}"
    fontSize: "1rem"
    lineHeight: 1.6
  small:
    fontFamily: "{typography.h1.fontFamily}"
    fontSize: "0.8125rem"
  label:
    fontFamily: "ui-monospace, 'JetBrains Mono', 'SFMono-Regular', Menlo, Consolas, monospace"
    fontSize: "0.75rem"
rounded:
  sm: "2px"
  md: "4px"
  lg: "6px"
  xl: "8px"
  full: "9999px"
spacing:
  gutter-mobile: "16px"
  gutter-desktop: "32px"
  rail-mobile: "2.25rem"
  rail-desktop: "12rem"
  page-max: "76rem"
  section-gap: "clamp(4.5rem, 3rem + 6vw, 8rem)"
components:
  button-primary:
    backgroundColor: "{colors.accent}"
    textColor: "#ffffff"
    rounded: "{rounded.md}"
    padding: "0 1.125rem"
    height: "2.75rem"
  button-primary-hover:
    backgroundColor: "{colors.accent-hover}"
    textColor: "#ffffff"
  button-secondary:
    backgroundColor: "transparent"
    textColor: "{colors.text}"
    rounded: "{rounded.md}"
    padding: "0 1.125rem"
    height: "2.75rem"
  badge-success:
    backgroundColor: "{colors.success-solid}"
    rounded: "{rounded.sm}"
    padding: "0.0625rem 0.4375rem"
  badge-warning:
    backgroundColor: "{colors.warning-solid}"
    rounded: "{rounded.sm}"
    padding: "0.0625rem 0.4375rem"
  badge-danger:
    backgroundColor: "{colors.danger-solid}"
    rounded: "{rounded.sm}"
    padding: "0.0625rem 0.4375rem"
---

# Design System: 4R Landing (landing-base-v2)

## Overview

**Creative North Star: "The path of one MR."**

This is a Persuade surface, not the Operate surface spa-b's own DESIGN.md
describes — its job is to make one visitor believe the product works, once,
top to bottom, not to be scanned all day. It runs on spa-b's exact token
world (`src/styles/tokens.css` is spa-b's `tokens.css` ported verbatim, minus
the Tailwind `@theme` projection — see
[`packages/spa-b/DESIGN.md`](../spa-b/DESIGN.md) for the full gray/accent
scale, semantic aliases, and the rationale behind them). What this document
records is only what the landing adds on top: a persuasive narrative device
(the boundary rail), a larger type scale built for a once-through read, and a
handful of stop-layout patterns that did not exist in spa-b.

The whole page is built to prosecute one argument, captured verbatim as an
HTML comment that is the first child of `<body>` in
[`src/layouts/Layout.astro`](src/layouts/Layout.astro) (direction seed
`110d183e`, structure 7 — "the path of one MR across the infrastructure
boundary"): THESIS (4R is a review service you run, proven by following one
MR through it), OWN-WORLD (spa-b's own world: Radix-style gray, one flat
indigo, hairlines, 2–8px radii, system sans — the product's real screens are
the imagery), STORY (`acme/payments-api` MR `!142` enters from GitLab,
crosses into your infrastructure, gets four passes, a computed score, your
selection, a release that waits for you, a Telegram ping, and ends at the
commands to run it), FIRST VIEWPORT (left-aligned mechanism headline, two
actions, the GitLab → 4R → provider diagram with `!142` entering), and FORM
(the boundary-rail structure described below). Read that comment for the
literal text; every rule below is how it actually landed in code.

**Key Characteristics:**
- One token world, two surfaces: spa-b owns the palette, this package owns
  the narrative shell built from it.
- A single running example (`acme/payments-api` MR `!142`) carries every
  screenshot and every data table; nothing on the page is a distinct,
  uncorrelated case study.
- The boundary between "GitLab" and "your infrastructure" is drawn once, as
  a literal rail down the page, and every stop declares whether it sits
  inside it or crosses it.
- Every screenshot is real spa-b UI at true pixel scale, never a stretched
  or fabricated mockup, and is labelled as example data.

## Colors

The landing does not define its own palette — it `@import`s
`./tokens.css` (spa-b's scales, aliases, and status colors, unchanged) and
spends it far more sparingly than the app does, because a Persuade surface
earns trust by restraint, not by decoration. The frontmatter above lists only
the primitives this package's own CSS/components actually reach for; treat
spa-b's DESIGN.md as the normative source for the full 12-step gray/accent
scales and the light/dark contrast rationale.

### Primary
- **Indigo accent** (`--accent` / `--accent-hover`, `hsl(233 55% 52%)` /
  `hsl(234 58% 46%)`): the boundary-rail marker square, the rail node once a
  stop is "current," the primary CTA, and link color. Nowhere else.

### Neutral
- **Gray scale** (`--bg-app`, `--bg-panel`, `--line`, `--line-strong`,
  `--text`, `--text-muted`): page background, panel/card backgrounds inside
  stop content, hairline borders, and body/muted text. Identical roles to
  spa-b.

### Status
- **Success / warning / danger / info** (`--success-*`, `--warning-*`,
  `--danger-*`, `--info-*`): reused verbatim from spa-b's status vocabulary,
  applied only to the `.badge` variants in the rail log (`Done`,
  `Awaiting confirmation`, `Published`, `Request changes`) — never invented
  new status colors for the landing.

### Named Rules
**The One Indigo Rule.** The accent appears in exactly four places on the
whole page: the rail/boundary marker squares, the current-stop rail node,
the primary button, and link color. It never appears as a background wash,
gradient stop, or decorative fill.

## Typography

**Body Font:** system sans stack (`--font-sans`, inherited from spa-b) —
the same stack the app uses, so the landing never introduces a second
typeface.
**Label/Mono Font:** `--font-mono` (inherited from spa-b) — used for the MR
id, file:line locations, commands, and path-log text.

**Character:** The app's dense 14px-base scale is replaced by a scale built
for a once-through read: bigger display sizes, a longer lede, and the same
system sans doing the work instead of a second display face.

### Hierarchy
- **H1** (600, `clamp(2.125rem, 1.4rem + 3.2vw, 3.5rem)`, line-height 1.08,
  −0.03em): the hero mechanism headline only. Capped at `17ch` so it never
  wraps past two lines.
- **H2** (600, `clamp(1.5rem, 1.1rem + 1.6vw, 2.125rem)`, line-height 1.18,
  −0.02em): every stop's section title.
- **Lede** (400, `1.1875rem`, line-height 1.55): the hero's one supporting
  paragraph, capped at `52ch`.
- **H3** (600, `1.0625rem`, line-height 1.4): sub-headings inside a stop
  (e.g. "Providers", "The rules", self-host step titles).
- **Body** (400, `1rem`, line-height 1.6): prose inside stops.
- **Small** (`0.8125rem`): captions, source line, provider list meta.
- **Label** (mono, `0.75rem`): path-log parts, table captions, `where`
  columns — always monospace because it is data, not prose.

### Named Rules
**The Once-Through Rule.** Type sizes scale up from spa-b's dense UI scale
because this page is read once, start to finish, not scanned repeatedly —
the opposite optimization from the app it demonstrates.

## Layout

**Container:** `.wrap` caps content at `--page-max: 76rem`, centered, with
`--gutter: 16px` under 60rem and `32px` at/above it.

**The boundary rail.** `#path` (the wrapper around every stop after the
hero) draws one continuous hairline (`.path::before`, 1px, `--line`) at
`--line-x = gutter + rail`. `--rail` is the column reserved for the rail
itself: `2.25rem` under 60rem (a thin left margin, log text moves under each
stop's heading) and `12rem` at/above it (a real log column to the line's
left, where the path-log text sits). A `.boundary-label` ("your
infrastructure", with the same 11px accent square used by every rail node)
marks where the boundary starts, immediately under the hero.

**Stop grammar.** Each `.stop` is a two-column grid (`rail` + content).
Every stop draws an 11px square `.rail-node` vertically centered on its H2
(`--head-mid`, derived from `--fs-h2`):
- **Inside stops** (`Passes`, `Score`, `Flow`, `Secrets`, `Self-host`): the
  node sits directly on the line, hairline border, filled with `--bg-app` at
  rest.
- **Crossing stops** (`Publish`, `Release` when awaiting Telegram, `Notify`)
  carry `crosses` and set `--out: 28px`: the node moves that far outside the
  line, gets a `--bg-panel` fill, and a 1px `.rail-tick` joins it back to the
  line. `crosses` marks an action that leaves the operator's infrastructure
  (to GitLab, the AI provider, or Telegram) — it is not decorative, it is
  read from the direction contract's STORY.
- The stop currently in the viewport (`IntersectionObserver`, progressive
  enhancement only) gets `.is-current`: its rail node fills with the accent
  instead of staying a neutral hairline square. Without JS every node stays
  neutral — the page never depends on this for meaning.

**The path log.** Each stop's `<p class="stop-log">` renders its `log` prop
("a · b" splits into stacked lines) plus an optional status `.badge`, reusing
spa-b's status vocabulary and colors (`Done`/success, `Awaiting
confirmation`/warning, `Published`/success, `Request changes`/danger). On
desktop it sits in the rail column to the left of the heading; on mobile it
follows the heading, joined by middle-dots instead of stacking.

**Rhythm:** `--section-gap: clamp(4.5rem, 3rem + 6vw, 8rem)` between stops;
a stop right after the boundary label gets a shorter, fixed gap
(`clamp(2.5rem, 2rem + 2vw, 3.5rem)`) since the label already read as a
separator.

**Responsive breakpoint:** `60rem` is the one structural breakpoint (rail
width, stop grid, two-column prose panes). A second breakpoint at `75rem`
governs when a side-crop stop's prose column (`Notify`, `Publish`,
`Release`) sits beside its screenshot instead of stacking above it — chosen
so the prose keeps ~16–18rem next to the crop at the crop's true pixel size,
never squeezed.

## Elevation & Depth

Same as spa-b: flat at rest, hairline-first. The only shadow in use is
`--shadow-sm` on `.shot-frame` (the screenshot frame) — a near-invisible lift
so the crop reads as a discrete object on the page, not a decorative card
effect. No shadow is used for hover states, buttons, or the rail; depth
elsewhere comes from a 1px border and a one-step background shift
(`--bg-app` → `--bg-panel`), exactly as in spa-b.

## Shapes

Inherited from spa-b unchanged: `--radius-sm` 2px / `--radius-md` 4px /
`--radius-lg` 6px / `--radius-xl` 8px, `--radius-full` reserved for
genuinely circular/pill UI. The landing's one addition is the Telegram
message bubble in `Notify` (`.bubble`), which squares one corner
(`xl xl xl sm`) to read as a chat bubble without adding a decorative tail.

## Components

### Header
Sticky (`position: sticky; top: 0`), `--bg-app` background, a `--line-subtle`
bottom hairline. Brand mark (`Mark.astro`, four 8×8 rounded squares, one lit
in accent — the 4R logo, echoing the four lenses) + nav links + a theme
toggle button (sun/moon SVG icons, swapped by an `html.dark`/`html:not(.dark)`
selector, not JS re-render). The toggle writes `localStorage["4r-color-scheme"]`
— the exact same key spa-b's own toggle uses, so a visitor's choice is
shared in spirit with the app they're about to self-host. A pre-paint
inline script in `Layout.astro` applies the stored or OS-preferred theme
class before first paint to avoid a flash.

### Hero
Two-column at ≥60rem (copy left, diagram right; stacked below it). The
diagram is the persuasive core: three `.node` boxes (GitLab → your 4R server
→ your AI provider) joined by two `.edge`s, each edge rendered as a pair of
opposing SVG arrows with a two-line label (`down: ...` / `up: ...`) stating
exactly what crosses that edge in each direction. The 4R server node sits
inside a `.boundary` box (its own hairline + accent-marked "your
infrastructure" label, the same device as the page-level rail, in miniature)
and lists all four lenses as done. This is the only place the whole
GitLab→4R→provider path is shown as a single diagram; every stop below it
narrates one piece of that same path in UI, not diagram, form.

### Stop layouts
Four recorded patterns, all built from the shared `<Stop>` shell (title, rail
node/tick, path log, optional status badge):
- **Text-led** (`Secrets`, `Self-host`): two-column prose/list at ≥60rem, no
  screenshot. `Self-host` additionally renders a `<pre><code>` block per
  step — the only place code blocks appear on the page.
- **Table** (`Score`): a horizontally-scrollable `<table>` (findings →
  points, with a `<tfoot>` total row) beside a short rules list. Collapses
  to stacked per-row cards under 48rem (`thead` visually hidden, each `<tr>`
  becomes a mini grid) rather than forcing horizontal scroll on mobile.
- **Side crop** (`Notify`, `Publish`, `Release`): prose (or a small mocked
  artifact — the Telegram bubble, the release step ledger) in one column,
  a `<Shot layout="side">` in the other, side-by-side only ≥75rem.
- **Full crop below** (`Flow`, `Passes`): two-column prose above, a
  full-width `<Shot>` (no `layout` prop, so it renders its `focus` crop)
  below.
`full` on `<Shot>` is an independent flag (not a layout): it adds a
desktop-only "View full screen" link to the untouched full capture, offered
only where the crop is legible enough on its own that zooming in adds real
value (`Flow`, `Passes`, `Publish`, `Release`).

### Shot
Every screenshot is a real spa-b screen, never a fabricated mockup. Each
`<Shot>` renders **both themes** as separate `<picture>`s (`is-light` /
`is-dark`, toggled by the same `html.dark` class the rest of the page uses)
so the correct theme's image is what paints — the other is `display:none`
and never fetched. Inside a `<picture>`, a `<source>` swaps in the `mobile`
crop under `47.99rem`; otherwise the image serves `side` (if the stop asked
for `layout="side"` and a side crop exists) or `focus`. Every crop has an
explicit CSS-pixel `width`/`height` (from `shots.ts`) plus a `2x` density
descriptor — it always renders at its true size, never upscaled, and
`max-width: 100%` is the only thing that can shrink it. **Every single
`<figcaption>` carries an "Example data" tag** (`.tag`), with no exception —
this is the page's one non-negotiable truth label, since every screen shows
the synthetic `acme/payments-api` MR `!142`, never a real operator's data.

## Do's and Don'ts

### Do:
- **Do** keep the accent to exactly the rail markers, the current-stop rail
  node, the primary button, and link color — restraint is the persuasive
  device here, not a missed opportunity to decorate.
- **Do** show real spa-b UI at true pixel scale (via `<Shot>`) as the proof
  for every claim; the diagram in `Hero` is the only invented illustration
  on the page, and it is explicitly a diagram, not a screenshot.
- **Do** label every screenshot "Example data" — the running example
  (`acme/payments-api` MR `!142`) is synthetic and must never be mistaken
  for a real operator's private repository.
- **Do** state the truth rule plainly where it's said at all: "the diff
  goes to the provider account you configured, under your key and that
  provider's terms" (`Secrets` stop) — never a stronger, false claim.
- **Do** carry the same status badge vocabulary and colors as spa-b
  (`Done`/success, `Awaiting confirmation`/warning, `Published`/success,
  `Request changes`/danger) rather than inventing new ones for marketing.
- **Do** collapse a data table to stacked cards on mobile (`Score`) instead
  of leaving it to horizontal-scroll as the only option.

### Don't:
- **Don't** use a gradient, glow, or glassmorphism anywhere — every surface
  is a flat fill (`--bg-app`/`--bg-panel`) with a 1px hairline border; the
  only shadow in the whole page is the near-invisible `--shadow-sm` lift
  under a screenshot frame.
- **Don't** claim "the code never leaves your servers" or any stronger
  privacy claim than is true — the diff is sent to the operator's
  *configured AI provider account*, and the page says so.
- **Don't** build a centered hero with a rounded pill badge above the
  headline, a 3-icon feature grid, or a vague bento layout — the hero is
  left-aligned copy beside a literal system diagram, and every feature claim
  below it is proven by a `<Shot>`, not an icon tile.
- **Don't** invent customers, testimonials, logos, benchmarks, adoption
  numbers, or pricing — none exist yet, and `PRODUCT.md` records that
  explicitly as a constraint.
- **Don't** add a second display typeface, emoji bullets, all-caps
  micro-label spam, or a pulsing/animated badge — the system sans stack and
  the rail's static geometry carry the whole page.
- **Don't** let a stop invent its own status-color vocabulary; reuse
  spa-b's four semantic variants (`success`/`warning`/`danger`/`info`) or
  `neutral`, never a fifth marketing-only color.
