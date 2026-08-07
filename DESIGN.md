---
name: 4R — AI Merge Request Reviewer
description: A borderless, near-black instrument panel for AI code review — hairlines, mono readouts, one lime signal.
colors:
  canvas: "#0a0a0b"
  surface: "#111113"
  line: "#2a2a30"
  ink: "#ededf0"
  muted: "#7c7c85"
  accent: "#d6ff3f"
  accent-ink: "#0a0a0b"
  flame: "#ff5a1f"
  danger: "#ff5a5a"
  warn: "#ffb84d"
  ok: "#8bef8b"
typography:
  display:
    fontFamily: "ui-sans-serif, system-ui, -apple-system, 'Segoe UI', sans-serif"
    fontSize: "1.5rem"
    fontWeight: 600
    lineHeight: 1.2
    letterSpacing: "-0.01em"
  title:
    fontFamily: "ui-sans-serif, system-ui, -apple-system, 'Segoe UI', sans-serif"
    fontSize: "0.95rem"
    fontWeight: 600
    lineHeight: 1.3
    letterSpacing: "-0.01em"
  body:
    fontFamily: "ui-sans-serif, system-ui, -apple-system, 'Segoe UI', sans-serif"
    fontSize: "0.875rem"
    fontWeight: 400
    lineHeight: 1.5
    letterSpacing: "normal"
  label:
    fontFamily: "ui-monospace, SFMono-Regular, Menlo, Consolas, monospace"
    fontSize: "0.68rem"
    fontWeight: 500
    lineHeight: 1.4
    letterSpacing: "0.14em"
rounded:
  none: "0"
spacing:
  xs: "4px"
  sm: "8px"
  md: "16px"
  lg: "20px"
  xl: "32px"
components:
  button-accent:
    backgroundColor: "{colors.accent}"
    textColor: "{colors.accent-ink}"
    rounded: "{rounded.none}"
    padding: "8px 16px"
    typography: "{typography.body}"
  button-line:
    backgroundColor: "transparent"
    textColor: "{colors.ink}"
    rounded: "{rounded.none}"
    padding: "8px 16px"
    typography: "{typography.body}"
  button-ghost:
    backgroundColor: "transparent"
    textColor: "{colors.muted}"
    rounded: "{rounded.none}"
    padding: "4px 8px"
    typography: "{typography.body}"
  button-danger-solid:
    backgroundColor: "{colors.danger}"
    textColor: "{colors.canvas}"
    rounded: "{rounded.none}"
    padding: "8px 16px"
    typography: "{typography.body}"
  input-underline:
    backgroundColor: "transparent"
    textColor: "{colors.ink}"
    rounded: "{rounded.none}"
    padding: "8px 0"
    typography: "{typography.body}"
  chip:
    backgroundColor: "transparent"
    textColor: "{colors.muted}"
    typography: "{typography.label}"
  card-boxed:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.ink}"
    rounded: "{rounded.none}"
    padding: "16px"
---

# Design System: 4R — AI Merge Request Reviewer

## Overview

**Creative North Star: "The Instrument Panel"**

4R looks like a precision diagnostic instrument for reading code, not a
consumer app. The screen is a near-black field (`#0a0a0b`) on which thin
`#2a2a30` hairlines act as gauge markings, monospace uppercase micro-labels act
as instrument readouts, and a single lime signal (`#d6ff3f`) marks whatever is
live, primary, or on. Nothing is boxed unless the content genuinely needs to be
isolated; structure is drawn with lines and whitespace the way a well-designed
panel separates dials without cramming each one in a bezel.

The register is engineered, exact, and calm under load. Type is the default
system sans for prose and the system monospace for anything that is *data* —
a filename, a line number, a status, a count, a label. Corners are square
everywhere; there is not a single rounded rectangle in the system. There are no
shadows: depth comes only from a three-step tonal stack (canvas → surface →
hairline), the way a matte panel reads as flat but layered. This is a dark-only
system by design (`color-scheme: dark`); there is no light theme to honor.

The two color voices never blur. Lime is the *live signal* — active nav, the
primary action, the score fill, the focus ring. Flame (`#ff5a1f`) is the
*alarm* — reserved for blocking findings and danger emphasis. Because both are
rationed, a screen with one lime element and one flame element reads instantly:
here is what's active, here is what's wrong.

**Key Characteristics:**
- Near-black canvas, hairline dividers, zero boxes by default.
- Monospace uppercase micro-labels as the signature detail.
- One lime "live signal" accent; one flame "alarm" accent; both rationed.
- Square corners everywhere (`border-radius: 0`).
- Flat — no shadows; depth is a canvas → surface → line tonal stack.
- Dark-only; system font stacks (no bundled web fonts).

## Colors

A near-black instrument field with hairline structure, a single lime live-signal,
a flame alarm, and a small semantic status set. Everything else is one of three
neutrals.

### Primary
- **Live Signal Lime** (`#d6ff3f`): The one brand voice. Marks what is *active or
  primary*: the current nav item (as a left-edge rule), the primary action button,
  the score-meter fill, input focus underlines, the `running` status, and the
  `:line` location pill. High-chroma against the black field so a single touch
  carries the whole screen. Text on lime uses **Accent Ink** (`#0a0a0b`).

### Secondary
- **Alarm Flame** (`#ff5a1f`): The danger-emphasis voice. Reserved for *blocking*
  findings and their severity edge. Deliberately distinct from Danger Red so
  "blocking" reads as hotter than a merely high-severity issue.

### Neutral
- **Canvas Black** (`#0a0a0b`): The page field and the base tonal layer.
- **Surface** (`#111113`): The one step up from canvas — used only where content
  is genuinely boxed (triage finding cards, mono file badges).
- **Hairline** (`#2a2a30`): Every divider, border, and gauge marking. Almost
  always used at reduced alpha (`/50`, `/60`, `/70`).
- **Ink** (`#ededf0`): Primary text and headings.
- **Muted** (`#7c7c85`): Secondary text, all mono labels, inactive nav, counts.

### Status (semantic — always paired with a label, never color-only)
- **OK Green** (`#8bef8b`): `approve` / `done` / `published`; published state as a
  faint `bg-ok/5` tint.
- **Warn Amber** (`#ffb84d`): `comment` recommendation, medium severity,
  `awaiting_approval`.
- **Danger Red** (`#ff5a5a`): `request_changes`, high severity, `error`.

### Named Rules
**The Two-Voice Rule.** There are exactly two accents and they are not
interchangeable: **lime = live signal** (active/primary/on), **flame = alarm**
(blocking/danger emphasis). Never use lime to warn; never use flame to indicate
primary action.

**The One Signal Rule.** Lime appears on a small fraction of any screen — one
primary action, one active nav item, one score fill. Its rarity is what makes it
legible. If two things are lime, one of them is wrong.

**The Labelled-Status Rule.** Status color is always shipped with a text label
(the recommendation bar, severity chips, status chips). Color is reinforcement,
never the sole signal — CVD- and contrast-safe by construction.

## Typography

**Display / Body Font:** system sans (`ui-sans-serif, system-ui, -apple-system,
'Segoe UI', sans-serif`)
**Data / Label Font:** system monospace (`ui-monospace, SFMono-Regular, Menlo,
Consolas, monospace`)

**Character:** No bundled web fonts — the system leans entirely on native OS
stacks, which keeps the instrument feeling fast and native to the developer's
machine. The expressive contrast is not two display faces but the sans/mono
split: prose is sans, *data* is mono. That single decision carries the whole
personality.

### Hierarchy
- **Display** (600, 1.5rem/`text-2xl`, tight tracking): Page titles in
  `PageHeader`. One per screen, paired above with a mono eyebrow label.
- **Title** (600, 0.95rem, tight tracking): Section titles (`section-title`) —
  the dominant heading inside a form or panel, clearly above the muted labels.
- **Body** (400, 0.875rem/`text-sm`): Default prose. Finding headlines step up
  to 1rem/`text-base`, medium weight, to lead each card.
- **Label** (500, 0.68rem, uppercase, `0.14em` tracking, muted): The signature
  `label-mono` — field labels, eyebrows, section readouts. Monospace and letter-
  spaced so metadata reads as instrumentation, not prose.
- **Chip** (mono, 0.66rem, uppercase, wide tracking): Inline status/metadata
  tags (dimension, severity, blocking, published), colored by role.

### Named Rules
**The Sans-Prose / Mono-Data Rule.** If it is language, it is sans. If it is a
datum — filename, line number, count, status, label — it is monospace, uppercase,
and letter-spaced. This split *is* the brand; do not introduce a decorative
display face to replace it.

## Layout

A borderless, hairline-and-whitespace grammar. On desktop, a fixed **13rem
(`w-52`) left sidebar** in canvas black carries primary nav; content sits to its
right. On mobile (`< md`) the sidebar is replaced by a fixed **bottom nav** of
four items (Home / Repos / Reviews / More), and secondary destinations move to a
"More" screen.

Structure is drawn, not boxed: the `row` idiom (`flex items-center gap-4`,
`border-b border-line/50`, `py-3`) turns lists into ledgers of hairline-separated
rows; the `divider` is a single `1px` `bg-line/50` rule. Page headers use a
bottom hairline (`border-b border-line/50`, `pb-4`, `mb-8`) rather than a filled
bar. Rhythm is generous and consistent: `gap-2`/`gap-4` between related items,
`p-4` inside the rare boxed card, `px-5 py-2.5` for nav rows, `mb-8` below a page
header.

### Named Rules
**The No-Box-By-Default Rule.** Content is separated with hairlines and space,
not containers. A boxed surface (`border-line` + `bg-surface`) is the deliberate
exception for content that must be isolated (a triage finding card), never the
default wrapper.

## Elevation & Depth

Flat. There are **no drop shadows** anywhere in the system. Depth is expressed
purely as a three-step tonal stack — **canvas** (`#0a0a0b`) → **surface**
(`#111113`) → **hairline** (`#2a2a30`) — the way a matte instrument panel reads
as layered without ever casting a shadow. The single exception is the route-change
progress bar's peg glow, which is a *lime signal* effect
(`box-shadow: 0 0 10px accent`), not a depth cue.

### Named Rules
**The Flat Panel Rule.** Never add a `box-shadow` to convey elevation. If a
surface needs to lift, step it up the tonal stack (canvas → surface) and/or add a
hairline — do not float it.

## Shapes

Uniformly square. **`border-radius: 0` everywhere** — buttons, inputs, cards,
chips, badges. The `btn` base explicitly sets `rounded-none`; there is no radius
scale to speak of. Borders are hairlines (`border-line`, typically at reduced
alpha). The one signature geometry is the **severity edge**: a finding card
carries a 2px colored rule on one edge — top on phone (`border-t-2`), left on
desktop (`border-l-2`) — colored flame for blocking, else danger/warn/muted by
severity. It is the system's one piece of structural color.

### Named Rules
**The Square-Corner Rule.** Nothing rounds. If a component needs to feel softer,
solve it with spacing or tone, never with a radius.

## Components

For each component: character first, then shape, color, states.

### Buttons
Square, flat, text-driven. All share the `btn` base
(`inline-flex`, `rounded-none`, `text-sm`, `font-medium`, disabled → `opacity-40`
+ no pointer events).
- **Shape:** square (`border-radius: 0`); default padding `8px 16px` (`px-4 py-2`).
- **Accent (primary):** lime fill, accent-ink text, `hover:opacity-90`. The one
  primary action per view.
- **Line (secondary):** transparent with a `border-line`, ink text; hover deepens
  the border to `ink` (`hover:border-ink`). No fill.
- **Ghost (tertiary):** muted text, tight padding (`4px 8px`); `hover:text-ink` +
  faint `bg-muted/20`. For inline/row actions.
- **Danger-solid:** danger fill, canvas text — destructive confirmations only.

### Inputs / Fields
- **Style:** the signature **underline field** (`field-underline`) — no box. A
  bottom hairline only (`border-0 border-b border-line`), transparent background,
  zero horizontal padding, placeholder at `muted/50`.
- **Focus:** the underline turns **lime** (`focus:border-accent`) — the only focus
  affordance, and a live-signal use of the accent.
- **Label:** paired above via `field-label` (the mono `label-mono`, block, `mb-1.5`).

### Chips
- **Style:** monospace, `0.66rem`, uppercase, wide tracking; no background, no
  border — just colored text.
- **State:** colored by role via text tokens — `text-muted` (dimension),
  severity classes (`text-danger`/`text-warn`/`text-muted`), `text-flame`
  (blocking), `text-ok` (published). Each pairs an inline Lucide glyph with a word.

### Cards / Containers
- **Default:** none — content is borderless (see Layout). There is deliberately no
  `card` shortcut.
- **Boxed variant (triage finding card):** built inline from palette idioms —
  `border-line` + `bg-surface`, `p-4`, square. Carries the **severity edge** (2px
  colored top on phone / left on desktop). Published state is a faint `bg-ok/5`
  **tint**, never dimming.

### Navigation
- **Desktop sidebar:** canvas field, `border-r border-line/60`. Items are muted
  text; the active item gets a **lime left-edge rule** (`border-l-2 border-accent`)
  and ink text. Each item pairs a Lucide icon with a sans label.
- **Mobile bottom nav:** fixed, hairline top border; four items, icon over a tiny
  (`0.6rem`) uppercase mono-ish label; active item turns **lime** (`text-accent`).

### Signature: the 4R readouts
- **Score meter:** a `2px`-tall `bg-line/40` track with a **lime** fill scaled to
  the 0–100 score. Thin, exact, gauge-like.
- **Recommendation bar:** a `2px` stacked bar (ok / warn / danger segments,
  2px gaps) over a labelled legend with mono counts — a distribution readout, not
  a chart.
- **File badge + line pill:** a mono `border-line`/`bg-surface` filename badge,
  followed by a `:NN` line pill in lime (`border-accent/40 bg-accent/10
  text-accent`) — location as instrumentation.

### Named Rules
**The Tint-Not-Dim Rule.** Represent a positive/done state (e.g. published) with a
faint semantic **tint** (`bg-ok/5`), never with reduced `opacity`. Opacity is
reserved for the disabled state (`opacity-40`) — dimming must always mean
"unavailable," never "complete," so actions on a completed item stay fully legible.

## Do's and Don'ts

### Do:
- **Do** separate content with hairlines (`border-line` at `/50`–`/70`) and
  whitespace; reach for a boxed `bg-surface` card only when content must be
  isolated.
- **Do** set `border-radius: 0` on every new component. Square is the system.
- **Do** put every datum — filename, line, count, status, label — in monospace,
  uppercase, letter-spaced; keep prose in the sans stack.
- **Do** use lime for exactly one "live/primary" thing per view (primary action,
  active nav, score fill, focus), and reserve flame for blocking/danger emphasis.
- **Do** ship status color with a text label (chips, bars) so meaning never
  depends on color alone.
- **Do** convey a done/positive state with a faint semantic tint (`bg-ok/5`).

### Don't:
- **Don't** add `box-shadow` for elevation — step the tonal stack instead.
- **Don't** round corners, anywhere.
- **Don't** use lime as a warning or flame as a primary action — the two voices
  don't cross.
- **Don't** wrap default content in bordered boxes; the borderless ledger is the
  point.
- **Don't** dim a completed item with `opacity` — dimming means disabled only.
- **Don't** introduce a bundled/decorative display font; the sans-prose /
  mono-data split carries the identity.
