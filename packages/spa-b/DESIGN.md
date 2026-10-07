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
| `--line-control` | gray-9 | Border of interactive controls (input, textarea, select, combobox, checkbox) and the switch's off track. Use this, not `--line`, for anything the user must perceive as a control |
| `--text` | gray-12 | Primary text |
| `--text-muted` | gray-11 | Secondary text, captions, labels |
| `--text-placeholder` | gray-10 | Input placeholders |
| `--text-on-accent` | accent-contrast | Text/icons on solid accent fills |
| `--accent` | accent-9 | Primary action background |
| `--accent-hover` | accent-10 | Primary action hover |
| `--accent-text` | accent-11 | Accent-colored text (links, active state) |
| `--accent-text-strong` | accent-12 | High-contrast accent text |
| `--accent-subtle-bg` | accent-3 | Selected/highlighted row or menu-item background |
| `--focus-ring` | accent-8 (light) / accent-10 (dark) | `:focus-visible` outline color |
| `--control-on` | accent-9 (light) / accent-10 (dark) | Switch "on" track |

### Non-text contrast (WCAG 1.4.11) and placeholder text

Controls must be perceivable without relying on a subtle fill: borders, the
switch track and the focus ring need **>= 3:1** against the surfaces they sit
on (`--bg-app`, `--bg-panel`, `--bg-panel-raised`); placeholder text is text, so
**>= 4.5:1**. `--line` / `--line-subtle` are decorative (cards, dividers) and
are intentionally low-contrast — never use them as a control's only boundary.
Measured with the WCAG 2.x relative-luminance formula on the resolved colors:

| Token | Light (app / panel / raised) | Dark (app / panel / raised) | Target |
| --- | --- | --- | --- |
| `--line-control` (input border, switch off track) | 3.43 / 3.31 / 3.43 | 3.96 / 3.76 / 3.76 | 3:1 |
| `--control-on` (switch on track) | 6.33 / 6.11 / 6.33 | 4.00 / 3.80 / 3.80 | 3:1 |
| `--focus-ring` | 3.52 / 3.40 / 3.52 | 4.00 / 3.80 / 3.80 | 3:1 |
| `--text-placeholder` | 4.93 / 4.75 / 4.93 | 4.89 / 4.64 / 4.64 | 4.5:1 |

Before: control border (`--line`) 1.49 / 1.44 light and 1.74 / 1.65 dark; switch
off track about 1.11 light; dark switch on 2.76; dark focus ring 2.55-2.69;
light placeholder 3.31. To keep the placeholder at 4.5:1, light `--gray-10` was
darkened from 49% to 45% lightness. Re-measure when any of these steps change.

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

- Radius: crisp/technical, not soft — `radius-sm` 2px, `radius-md` 4px
  (default component radius — buttons, inputs, cards), `radius-lg` 6px,
  `radius-xl` 8px (dialogs/large panels), `radius-full` reserved for
  genuinely circular/pill UI only (switch track/thumb, avatars, badges, the
  mobile drawer handle) — never for panels or dialogs.
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

## Overlays (dialogs, alert dialogs, drawers)

Every Reka `DialogOverlay` / `AlertDialogOverlay` / `DrawerOverlay` in the
app uses one shared `.overlay` utility class (`src/shared/assets/main.css`,
`@layer components`) instead of a one-off `bg-*` class per call site:
darkened (`--overlay`, true black at 60% opacity — not the gray scale, which
flips light/dark and would brighten rather than darken a dark-theme dialog's
backdrop) **and** blurred (`backdrop-filter: blur(6px)`), with a `data-state`
driven opacity fade using the existing `--duration-base`/`--ease-standard`
motion tokens. `z-index` stays a per-call-site utility (`class="overlay
z-20"`) since stacking differs by surface (command palette vs. a settings
dialog vs. the mobile drawer) — everything else about the backdrop is
centralized and retuned from one place. Apply `.overlay` to any new
Dialog/AlertDialog/Drawer overlay so it matches automatically.

## Icons

**lucide-vue-next**, the official Lucide package for Vue. `atoms/Icon.vue`
wraps it behind a simple kebab-case `name` prop (`<Icon name="settings" />`,
no `lucide:` prefix) so call sites stay simple and the icon set can change
without touching every usage.

Internally, `Icon.vue` keeps a small `Record<string, Component>` map from
kebab-case name to a **named import** of that icon (`import { Settings } from
'lucide-vue-next'`) — add an icon to the map the first time it's used
elsewhere. Named imports are tree-shaken (`sideEffects: false`), so only
icons actually referenced ship in the bundle; an unrecognized `name` renders
nothing instead of throwing. (`lucide-vue-next`'s older `dynamicIconImports`
lazy-loading helper isn't shipped in the installed 1.x line, so this static,
still-tree-shaken map is the equivalent for this package version.)

Some Lucide names changed upstream — reach for the *current* kebab name, not
the old alias: `home` → `house`, `more-horizontal` → `ellipsis`,
`check-circle-2` → `circle-check`, `alert-triangle` → `triangle-alert`,
`x-circle` → `circle-x`. Draw icons in Lucide's stroke weight consistently;
never substitute an emoji or Unicode glyph for an icon.

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

The app shell (S1) has landed: `src/pages/index.vue` is now the real Home
screen. The original token/component smoke test moved to
`src/pages/design.vue` — a **dev-only** reference route (gated out of
production by the router guard in `src/core/router/index.ts`; unreachable
once built). It renders the gray and accent scales, the status tokens, the
type scale, a theme toggle (light/dark/system), Reka UI `Tabs`, `Dialog`, and
`DropdownMenu` styled purely with the token layer, and a Shiki-highlighted
unified diff.
