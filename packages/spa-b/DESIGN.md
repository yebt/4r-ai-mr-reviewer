# spa-b — Design system

Source of truth for spa-b's visual world. Read this before building any new
screen. It documents what F1+F2 established (foundation only); the real app
shell, atoms, and navigation ship in later tasks (F3, S1, S2).

## Philosophy

**Radix-minimal, dual-theme, usability-first.** 4R is a deeply usable dev
tool — config forms, dashboards, diffs, dense data — not a marketing surface.
Decoration never competes with the task. Structure comes from a disciplined
token scale (Radix Themes' approach, reimplemented, not copied), hairline
borders, soft radii, and restrained motion. Light and dark are both
first-class: neither is the "real" theme with the other bolted on.

Color strategy: **Restrained** — neutral gray carries the interface, one
accent (indigo) marks interactive/emphasis state. This is the correct
strategy for an Operate surface: the visitor came to configure and review,
not to be persuaded.

## Token system

All tokens live in `src/shared/assets/tokens.css` as CSS custom properties,
then get projected into Tailwind v4 utilities via one `@theme inline` block
in the same file. **Retune the whole app from one place**: edit the
`--accent-*` scale (indigo today) to change the accent everywhere.

### Scales

Two 12-step scales, gray and accent, each following a fixed Radix-style step
role regardless of hue — so "step 9" always means "solid/emphasis
background" whether you're reading gray or accent, light or dark:

| Step | Role                              |
| ---- | ---------------------------------- |
| 1    | App background                     |
| 2    | Subtle panel background            |
| 3    | UI element background              |
| 4    | UI element hover background        |
| 5    | UI element active/pressed background |
| 6    | Subtle border                      |
| 7    | Default border                     |
| 8    | Strong border / focus ring         |
| 9    | Solid background (accent/emphasis) |
| 10   | Solid background, hover            |
| 11   | Low-contrast text (AA on step 1/2) |
| 12   | High-contrast text                 |

Both scales are hand-tuned per theme (not derived from one another) to hold
WCAG AA contrast: step 11 on step 1/2 backgrounds, and step 12 everywhere.

### Semantic aliases

App code should reach for these, not the raw numbered steps, unless building
something that genuinely needs a specific step (e.g. a custom illustration):

| Token | Reads from | Use |
| --- | --- | --- |
| `--bg-app` | gray-1 | Page background |
| `--bg-panel` | gray-2 | Raised panel / card background |
| `--bg-panel-raised` | gray-1 (light) / gray-2 (dark) | Popovers, dialogs — sits above `--bg-panel` |
| `--bg-hover` | gray-3 | Hover background on interactive surfaces |
| `--bg-active` | gray-4 | Pressed/active background |
| `--line-subtle` | gray-6 | Dividers, faint separators |
| `--line` | gray-7 | Default component border |
| `--line-strong` | gray-8 | Emphasized border, input focus edge |
| `--text` | gray-12 | Primary text |
| `--text-muted` | gray-11 | Secondary text, captions, labels |
| `--text-placeholder` | gray-9 | Input placeholders |
| `--text-on-accent` | accent-contrast | Text/icons on solid accent fills |
| `--accent` | accent-9 | Primary action background |
| `--accent-hover` | accent-10 | Primary action hover |
| `--accent-text` | accent-11 | Accent-colored text (links, active state) |
| `--accent-text-strong` | accent-12 | High-contrast accent text |
| `--accent-subtle-bg` | accent-3 | Selected/highlighted row or menu-item background |
| `--focus-ring` | accent-8 | `:focus-visible` outline color |

### Status (success / warning / danger / info)

Each has three variants, both themes, all AA-checked:

- `--{status}-bg` — subtle background for banners/badges
- `--{status}-solid` — saturated background for solid buttons/badges
- `--{status}-text` — accessible text color, readable on `-bg` and on `--bg-app`

Tailwind utilities: `bg-success-bg`, `text-success-text`, `bg-success-solid`,
and the same for `warning`, `danger`, `info`.

### Type scale

Modest, deliberate, dense-tool sized (base is 14px, not 16px — this is a
config/data tool, not prose):

| Utility | Size / line-height | Use |
| --- | --- | --- |
| `text-xs` | 12px / 16px | Captions, timestamps |
| `text-sm` | 13px / 18px | Meta text, table cells |
| `text-base` | 14px / 20px | Body default |
| `text-md` | 15px / 22px | Emphasized body |
| `text-lg` | 17px / 24px | Subheading |
| `text-xl` | 20px / 28px | Section heading |
| `text-2xl` | 24px / 32px | Page title |
| `text-3xl` | 30px / 38px | Display (rare) |

Fonts: `--font-sans` = system-ui stack (no web-font download/FOUT cost for a
tool people reopen many times a day); `--font-mono` = ui-monospace /
JetBrains Mono stack for code, diffs, tokens, and any measured/data value.

### Radius, elevation, motion

- Radius: `radius-sm` 4px, `radius-md` 6px (default component radius),
  `radius-lg` 8px, `radius-xl` 12px, `radius-full` for pills/avatars.
- Elevation: `shadow-token-sm/md/lg` — soft, offset-and-blur shadows only
  (never a hard 0-blur block shadow; never a zero-offset colored halo — that
  is decoration, not depth). Panels mostly rely on the 1px border, not
  shadow; shadow is reserved for things that float above content (dialogs,
  popovers, menus).
- Motion: `--duration-fast` 120ms / `--duration-base` 180ms / `--duration-slow`
  280ms, eased with `--ease-standard` (standard ease-out curve). All three
  collapse to 0ms under `prefers-reduced-motion: reduce` — this is handled
  once, globally, in `tokens.css`; components never need their own
  reduced-motion branch for duration.

## Theme switching

`useColorScheme()` (`src/shared/composables/useColorScheme.ts`) wraps
VueUse's `useColorMode`, persisted under the `4r-color-scheme` key, default
`auto` (system). It toggles a `light`/`dark` class on `<html>`; `tokens.css`
keys both the raw scales and Tailwind's `dark:` variant off that class.

To avoid a flash of the wrong theme, `index.html` runs a small inline script
in `<head>` that reads the same storage key (or `prefers-color-scheme`) and
applies the class **before** Vue mounts — this is the one place theme logic
is duplicated outside the composable, deliberately, because it must run
before any JS module loads.

## Icons

**@iconify/vue** + **Lucide** (`@iconify-json/lucide`), on-demand
(`Icon icon="lucide:settings"`) rather than a bundled sprite — no build step,
tree-shaken per icon actually used. Draw icons in Lucide's stroke weight
consistently; never substitute an emoji or Unicode glyph for an icon.

## Code / diff highlighting

**Shiki**, via `src/shared/composables/useCodeHighlighter.ts`. Uses
`createHighlighter` (the fine-grained core entry, not `shiki/bundle/full`) so
only the requested themes (`github-light` / `github-dark`) and langs
(`diff`, `typescript`, `json`, `bash`, `vue`, `html`, `css`) are loaded.

Highlighting runs **once per snippet with both themes** (`defaultColor:
false`); the output carries `--shiki-light`/`--shiki-dark` CSS variables per
token. The `.shiki` rules in `main.css` select the active pair off the same
`.dark` class the rest of the token system uses — switching theme never
re-highlights a snippet. Diffs (`lang: 'diff'`) are first-class: reviewing
diffs is 4R's core job.

## Atomic-design layering

`src/shared/ui/design-system/{atoms,molecules,organisms,templates}` (F3+
builds into these; F1+F2 only established the folders and the tokens they'll
consume):

- **atoms** — smallest reusable pieces with no internal layout decisions:
  Button, Input, Field label, Badge, Icon wrapper, Avatar. Style Reka UI
  primitives here; an atom owns exactly one Reka primitive (or none).
- **molecules** — a small group of atoms with one job: a labeled form field
  (Field + Input + error text), a menu item row, a search input with a clear
  button. No page-level layout, no data fetching.
- **organisms** — a distinct, self-contained section of a screen composed
  from atoms/molecules: the settings section list, the command palette, a
  diff viewer panel, the app sidebar's nav list. May hold local UI state; no
  route awareness.
- **templates** — page-level layout skeletons (the responsive app shell,
  a settings-page layout) — regions and slots, not final content.

Pages (`src/pages/*.vue`) compose a template with real data/routing; they
should stay thin.

## What F1+F2 proved (see the preview page)

`src/pages/index.vue` is a **temporary** visual smoke test, not a real
screen — remove it once the app shell (S1) lands. It renders the gray and
accent scales, the status tokens, the type scale, a theme toggle
(light/dark/system), Reka UI `Tabs`, `Dialog`, and `DropdownMenu` styled
purely with the token layer, and a Shiki-highlighted unified diff.
