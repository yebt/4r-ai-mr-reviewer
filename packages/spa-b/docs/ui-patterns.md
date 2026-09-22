# spa-b UI/UX Pattern Playbook

Reusable, actionable reference for building spa-b screens. Read this **before**
building a component that touches navigation, overlays, forms, lists, or
theming. Cite the pattern number in PR/task descriptions instead of
re-deriving the decision.

Stack: Vue 3.5 + **Reka UI** (headless, `reka-ui@2.10.5`, the Radix port for
Vue) + Tailwind v4 + dual light/dark tokens (see `DESIGN.md`). Reka UI's full
primitive export list (verified against `packages/core/src/index.ts`,
2026-09): `Accordion, AlertDialog, AspectRatio, Autocomplete, Avatar,
Calendar, Checkbox, Collapsible, ColorArea/Field/Slider/Swatch(Picker),
Combobox, ConfigProvider, ContextMenu, Date*, Dialog, Drawer, DropdownMenu,
Editable, FocusScope, HoverCard, Label, Listbox, Menubar, NumberField,
Pagination, PinInput, Popover, Presence, Primitive, Progress, RadioGroup,
RangeCalendar, Rating, RovingFocus, ScrollArea, Select, Separator, Slider,
Splitter, Stepper, Switch, TagsInput, Tabs, Time*, Toast, Toggle(Group),
Toolbar, Tooltip, Tree, Viewport, VisuallyHidden`. Notably **Reka UI ships a
native `Drawer` primitive with snap points** — this is not a Radix-only gap,
so we build the mobile bottom sheet on `Drawer`, not on `Dialog`.

This is a **dev tool that centralizes UX**: config, forms, dashboards, code
diffs, one coherent product — not a marketing site. Every pattern below is
chosen for density, keyboard efficiency, and predictability over decoration.

---

## 1. App navigation

**Desktop:** persistent left sidebar (sections/projects) is the default for a
centralized dev tool — it keeps navigation visible while the user works
dense forms/diffs in the main pane, unlike a top nav which competes for
vertical space. Reserve a top bar for global actions (search/command
palette trigger, account, environment switcher), not primary nav.

**Mobile:** **bottom tab bar** for the 3–5 top-level destinations (e.g.
Dashboard, Config, Runs, Diffs, Settings-as-5th-or-collapsed). Evidence:
"the tab bar wins on engagement and feature discovery for most apps... use
tab bars for 3–5 most important destinations and reserve drawer navigation
for secondary options" — [NN/g mobile nav patterns](https://www.nngroup.com/articles/mobile-navigation-patterns/),
[designstudiouiux mobile nav 2026](https://www.designstudiouiux.com/blog/mobile-navigation-ux/).
Use a **hamburger → Drawer** only for secondary/overflow items (settings,
help, account, less-used sections) — never for primary navigation. 2–5 tabs
is the sweet spot; 6+ becomes cramped on small phones.

**When NOT to use bottom tabs:** more than ~5 top-level sections, or a
section whose own screen already needs the bottom safe area for a primary
action bar — in that case fold it under an "More" tab that opens a Drawer.

- (b) Primitive: no single "AppShell" primitive exists in Reka UI — this is
  composed from layout + `NavigationMenuRoot`/`NavigationMenuList` (desktop
  sidebar sections with sub-content) and `DrawerRoot` (mobile hamburger
  drawer / overflow sheet). The bottom tab bar itself is plain markup (a
  `nav` with `role="tablist"`-like semantics only if it behaves as tabs;
  usually it's just a list of router links, so plain `<nav><ul>` is correct,
  not `TabsRoot`).
- (c) A11y: sidebar/nav landmark (`<nav aria-label="Primary">`), current
  page indicated with `aria-current="page"`, drawer traps focus and returns
  it to the trigger on close (Reka `Drawer` `modal` prop handles this),
  bottom tab bar items are real links/buttons ≥44×44px targets.
- (d) Mobile vs desktop: **structural change**, not reflow — sidebar
  disappears entirely below the breakpoint; bottom tab bar and hamburger
  drawer are mounted instead (don't just hide/collapse the sidebar into an
  icon rail on mobile, that fails touch-target and thumb-reach needs).
- (e) Checklist: ☐ ≤5 primary destinations on mobile tabs ☐ secondary items
  live in the overflow Drawer, not crammed into tabs ☐ `aria-current`on
  active item ☐ 44×44 min touch targets ☐ safe-area padding under tab bar
  (`env(safe-area-inset-bottom)`).

## 2. Command palette (⌘K / Ctrl-K)

**When to use:** power-user shortcut for search + actions across a
centralized tool with many config screens (exactly this app's shape).
**When NOT:** don't use it to paper over confusing/undiscoverable primary
nav — "you should already have visible navigation, and the palette should be
the shortcut, not the app" — [uxpatterns.dev command palette](https://uxpatterns.dev/patterns/advanced/command-palette).

- (b) Primitive: compose **`ComboboxRoot` + `ComboboxInput` + `ComboboxContent`
  + `ComboboxViewport` + `ComboboxItem`**, wrapped in a **`DialogRoot`** (or
  Reka's dialog-flavored combobox pattern) for the modal overlay + focus
  trap + `Escape`-to-close. Use `useFilter()` (from `reka-ui`) for
  fuzzy/contains filtering of actions/recents; group results with
  `ComboboxGroup`/`ComboboxLabel` (Actions / Recent / Navigate). For >1k
  items, use `ComboboxVirtualizer` (verified in docs — virtualized combobox
  with manual pre-filtering).
- (c) A11y: input keeps real DOM focus the whole time; a virtual highlight
  moves through the listbox rather than moving focus into list items
  (standard combobox pattern, confirmed by Reka's Combobox keyboard docs:
  `ArrowDown/ArrowUp` navigate, `Enter` selects, `Esc` closes and restores
  input). Add an `aria-live="polite"` result-count region so screen reader
  users get "12 results" on filter, not silence. Global hotkey (`⌘K`/`Ctrl+K`)
  must not conflict with browser/OS bindings; always also expose a visible
  trigger button (search icon) — never keyboard-only.
- (d) Mobile vs desktop: desktop = centered modal dialog, ⌘K bound globally.
  Mobile = full-screen takeover (not a small centered modal — no room), open
  via a visible search icon in the top bar or bottom-tab "Search" slot, not
  a keyboard shortcut (no hardware keyboard). Font size on the input ≥16px
  to prevent iOS auto-zoom.
- (e) Checklist: ☐ visible trigger button in addition to hotkey ☐ recents
  group when query is empty ☐ `aria-live` result count ☐ mobile = full-screen
  not small modal ☐ virtualize if list >~200 items.

## 3. Bottom sheet / mobile drawer

**When to use:** mobile filters, item detail, contextual forms, secondary
actions — "almost every secondary interaction (filters, details, forms) has
moved to a bottom sheet" for thumb reach — [designstudiouiux 2026](https://www.designstudiouiux.com/blog/mobile-navigation-ux/),
[LogRocket bottom sheets](https://blog.logrocket.com/ux-design/bottom-sheets-optimized-ux/).
**When NOT:** destructive/blocking confirmations that must not be
swipe-dismissed accidentally — use `AlertDialogRoot` instead (no swipe-to-
dismiss ambiguity). Also skip it for content that genuinely needs the full
viewport height on open — use a full-screen route/dialog instead.

- (b) Primitive: **Reka UI has a native `Drawer` primitive** — do **not**
  hand-roll this on `Dialog`. Verified props: `snapPoints` (fractions 0–1,
  px, or `'30rem'` strings), `snapPoint` / `defaultSnapPoint` (controlled/
  uncontrolled active snap), `snapToSequentialPoints` (step-by-step vs
  nearest-by-distance), `swipeDirection` (defaults `"down"`), `modal`
  (`true` = focus trap + scroll lock + outside-dismiss, `'trap-focus'` =
  focus trap without outside-pointer blocking, `false` = non-modal). Default
  anchor is the bottom edge. Source: reka-ui Drawer docs & drawer-design spec.
- (c) A11y: `modal: true` (default) gives focus trap + background scroll
  lock automatically — keep it `true` for anything that isn't a persistent
  side panel. Provide a visible drag handle **and** a close button (don't
  rely on gesture-only dismissal — motor-impaired/mouse users need a
  target). `Escape` should close (Dialog-family convention Reka follows).
- (d) Mobile vs desktop: **this is the "two lines" pattern itself** — on
  mobile, use `Drawer` anchored bottom with snap points (e.g. `[0.4, 0.9]`
  for peek/expanded); on desktop, the equivalent UI is a **side Drawer**
  (`swipeDirection: "right"`/anchor right) or a **`Popover`** for a small,
  anchored contextual panel — never a bottom sheet on desktop (no thumb
  ergonomics reason for it, and it blocks horizontal reading flow).
- (e) Checklist: ☐ built on `Drawer`, not `Dialog` ☐ `modal: true` unless
  explicitly a non-blocking side panel ☐ visible drag handle + close button
  ☐ snap points chosen deliberately (peek + full, not just full) ☐ desktop
  equivalent is side `Drawer`/`Popover`, not the same bottom sheet stretched
  wide ☐ `padding-bottom: env(safe-area-inset-bottom)` on content.

## 4. Dialog vs Drawer vs Sheet vs Popover — decision matrix

| Need | Use | Reka primitive |
|---|---|---|
| Must act before continuing (destructive/irreversible, <6s task) | **Dialog / AlertDialog** | `DialogRoot` (generic), `AlertDialogRoot` (destructive confirm — non-dismissable by outside click, forces explicit choice) |
| Underlying context should stay visible/usable; longer or revisited task | **Drawer** (side panel, desktop) | `DrawerRoot` with `swipeDirection: "right"`/side anchor, `modal: 'trap-focus'` if non-blocking |
| Mobile secondary interaction, thumb reach, variable height | **Bottom sheet** | `DrawerRoot`, bottom anchor + `snapPoints` |
| Quick, low-friction, anchored to a trigger, single focused task, no submit button | **Popover** | `PopoverRoot` |
| Non-interactive supplementary text on hover/focus | **Tooltip** | `TooltipRoot` |

Rule of thumb from research: *"if a popover has a submit button, it wanted
to be a dialog"* — [Eleken popover UX guide](https://www.eleken.co/blog-posts/popover-ux).
*"Use a modal if the action is risky/hard to undo... if users spend more
than 6 seconds in it, it probably shouldn't be a dialog"* — [Userpilot modal UX](https://userpilot.com/blog/modal-ux-design/).
*"Use a drawer if the user benefits from keeping the underlying interface
visible"* — [onething.design modal vs drawer](https://www.onething.design/post/modal-vs-drawer).

- (c) A11y common to all: `Dialog`/`AlertDialog`/modal `Drawer` trap focus
  and return it to the trigger on close; `Popover`/`Tooltip` never trap
  focus. `AlertDialogRoot` specifically: `Enter`/`Space` open/close, `Tab`/
  `Shift+Tab` cycle within, `Esc` closes and restores trigger focus
  (verified in Reka AlertDialog keyboard docs).
- (d) Mobile vs desktop: `Dialog` stays a dialog on both (confirmations
  don't change shape) but should go near-full-width on mobile with safe
  margins, not a tiny centered box. `Popover` on mobile should widen/reflow
  to avoid edge clipping — consider swapping to a `Drawer` if content is
  non-trivial on small screens (Reka `Popover`'s `collisionPadding`/
  `avoidCollisions` props help but don't solve a fundamentally too-wide
  popover).
- (e) Checklist: ☐ picked row in the matrix, not "whatever's easiest to
  code" ☐ `AlertDialogRoot` (not `DialogRoot`) for destructive confirms ☐
  popovers have no submit/save action inside.

## 5. Toast / notifications

**When to use:** transient, non-blocking status ("Saved", "Sync failed") the
user doesn't need to act on. **When NOT:** anything requiring a decision
(use Dialog) or content the user must be able to re-read later (use an
in-page log/history instead of stacking toasts).

- (b) Primitive: **`ToastProvider` (root, place once near app root) +
  `ToastViewport` + `ToastRoot` + `ToastTitle`/`ToastDescription` +
  `ToastAction`/`ToastClose`**. Verified props: `ToastProvider.duration`
  (default `5000`ms), `swipeDirection` (default `"right"`), `label`
  (default `"Notification"`, screen-reader context string); `ToastViewport`
  renders as an `<ol>` by default with a `hotkey` (default `["F8"]`) to jump
  focus to the notifications landmark; per-toast `duration` overrides the
  provider default.
- (c) A11y: the viewport is an ARIA landmark reachable via the `hotkey` —
  keep the default `F8` or document a custom one. Status updates must be
  announced without moving focus (WCAG 4.1.3 Status Messages) — Reka's
  Toast handles the live-region semantics; don't wrap toast content in
  anything that steals focus. **Never auto-dismiss error toasts** — errors
  need explicit dismissal (WCAG 2.2.1 Timing Adjustable concern) — [accessibility.build notifications guide](https://accessibility.build/guides/accessible-notifications),
  [LogRocket toast best practices](https://blog.logrocket.com/ux-design/toast-notifications/).
  Pause the auto-dismiss timer on hover/touch focus.
- (d) Mobile vs desktop: desktop toasts stack bottom-right or top-right;
  mobile toasts stack bottom-center **above** the tab bar and above
  `safe-area-inset-bottom`, full-width minus gutters (thumb-width margins
  make small side-stacked toasts hard to read/dismiss on a phone).
- (e) Checklist: ☐ 5–7s default duration, 8–10s+ if it has an action button
  ☐ errors don't auto-dismiss ☐ max 3–5 visible at once, queue the rest ☐
  positioned clear of the mobile tab bar and safe-area ☐ swipe-to-dismiss
  matches `swipeDirection` used elsewhere in the app for consistency.

## 6. Forms & fields

Applies to every config form in the app — this is the highest-reuse pattern.

- (a) When to use inline vs summary errors: inline per-field error is the
  default for any form with more than 1–2 fields (immediate, localized
  correction). Add a **summary** only for long/multi-section forms so
  keyboard/screen-reader users get an overview before diving in — link each
  summary item to its field via `href="#field-id"`.
- (b) Primitive: **`Label`** (associates via `for`/nesting — verified
  pattern: wrap the control, or `for` + matching `id` on the trigger),
  `Checkbox`/`Switch`/`RadioGroup`/`Select`/`Combobox`/`NumberField`/
  `TagsInput` for controls. There's no dedicated "FormField" primitive in
  Reka UI — build a small `FormField.vue` wrapper (label + control slot +
  description + error) once and reuse it everywhere.
- (c) A11y (verified against current WCAG guidance, not memory):
  - Every control has a programmatically associated `<label>`.
  - Description text: `aria-describedby` pointing at the description
    element id (don't put clarifying instructions only in a placeholder).
  - Error text: same `aria-describedby` (can reference multiple ids
    space-separated: description + error), plus `aria-invalid="true"` on
    the control while invalid.
  - Announce validation errors with `role="alert"` or `aria-live="polite"`
    on the error container; **move focus to the first invalid field on
    failed submit** — [508blueprint form checklist](https://508blueprint.com/html-form-accessibility-checklist-wcag/),
    [thewcag.com accessible forms](https://www.thewcag.com/examples/forms).
  - Mark required fields with `required`/`aria-required` **and** a visible
    "Required" cue (color alone is not enough); mark optional fields
    explicitly if the form is majority-required (state the minority case).
  - Touch targets ≥44×44px (Checkbox/Switch/Select triggers) per WCAG 2.5.5;
    Reka primitives don't enforce sizing, own it in your Tailwind classes.
- (d) Validation timing: validate on blur for first pass (not on every
  keystroke — too noisy), then switch to on-input once a field has been
  touched-and-invalid (so the error clears as soon as it's fixed). Validate
  on submit regardless as the final gate.
- (d) Mobile vs desktop: desktop can use multi-column field grids; mobile
  forces single-column (structural, not just narrower columns) so labels
  stay left-aligned/stacked and error text doesn't get squeezed. Numeric/
  email/url inputs get the matching `inputmode`/`type` for the right mobile
  keyboard.
- (e) Checklist: ☐ label associated ☐ `aria-describedby` for description+
  error ☐ `aria-invalid` while invalid ☐ focus moves to first error on
  submit ☐ required marked visibly, not color-only ☐ single column on
  mobile ☐ correct `inputmode`/`autocomplete` tokens.

## 7. Data lists/tables on mobile

- (a) When to use table reflow vs cards: reflow (horizontal scroll or
  column-hiding) for dense operational tables where users need to scan
  many rows and compare columns (keep the table, don't lose the grid
  relationship). Switch to a **card list** when a table's columns don't
  form a meaningful grid at a glance on a small screen — e.g. each row is
  really "one record with several attributes", not being cross-compared.
  Given spa-b centralizes config/dashboards/diffs, prefer: dashboards/diff
  lists → cards on mobile; dense config/run tables → horizontal-scroll
  table with a pinned first column (identity), not full card conversion,
  so column relationships survive.
- (b) Primitive: no Reka UI table/list primitive — build on plain
  semantic `<table>`/`<ul>` markup; use `ScrollArea` (verified export) to
  give the horizontal-scroll table a styled, cross-platform scrollbar
  instead of relying on native overflow scroll only.
- (c) A11y: table keeps `<th scope="col">`/`<th scope="row">` even when
  visually reflowed; card view for the same data needs each card's fields
  labeled (don't rely on visual position alone — a screen reader user
  loses the "this value is the Status column" context that a table gives
  implicitly).
- (d) Mobile vs desktop states — same four states, different layout:
  - **Loading**: desktop = table skeleton rows matching column widths;
    mobile = card skeletons (3–5 shimmering blocks).
  - **Empty**: centered icon + one-line explanation + primary action,
    same shape both platforms — this doesn't need two lines.
  - **Error**: inline banner above the list/table (not a toast — errors on
    the primary content need to persist until dismissed/retried).
  - **Skeleton**: match the real content's approximate line count/widths
    so layout doesn't jump on load.
- (e) Checklist: ☐ decided reflow vs cards per this section's rule, not by
  default ☐ `scope` attributes kept in reflowed tables ☐ card fields
  explicitly labeled ☐ all 4 states (loading/empty/error/loaded) designed,
  not just the happy path.

## 8. Responsive strategy — the two UI lines

- **Breakpoint:** single primary breakpoint at **768px** (mobile vs
  desktop line) is enough for a two-line strategy — add a secondary
  `1024px` breakpoint only if the desktop layout itself needs a
  narrow/wide split (e.g. sidebar collapses to icon rail between 768–1024).
  Set breakpoints where *this app's* content actually breaks, not by
  copying generic device widths — [thefix.it 2025 breakpoint guide](https://thefix.it.com/what-breakpoints-should-i-use-in-2025-the-ultimate-layout-guide/).
- **What changes STRUCTURALLY (component swap, not CSS reflow):** app
  navigation (sidebar → bottom tabs + drawer, pattern 1), bottom sheet vs
  side Drawer/Popover (pattern 3), command palette modal vs full-screen
  (pattern 2), table vs card list (pattern 7). These need **two actual
  render branches** (`v-if`/separate components), not just Tailwind
  responsive classes — the interaction model itself differs, not just
  spacing.
- **What just REFLOWS (same component, CSS-only):** form field columns
  (pattern 6), toast stack position, dialog width/margins, type scale
  (already defined in `DESIGN.md`'s dense token scale).
- **Container queries:** use them for reusable components dropped into
  varying contexts (e.g. a card that appears both in a sidebar and a full
  page); keep `@media` for page-level structural swaps (nav, table↔cards) —
  [medium.com container style queries 2026](https://medium.com/@alekswebnet/modern-css-responsive-breakpoints-using-container-style-queries-64c9bea81ad6).
- **Touch targets (WCAG 2.5.5 / 2.5.8):** 44×44 CSS px minimum for primary
  interactive controls everywhere (not just "mobile" — desktop touchscreens
  and trackpads benefit too); 24×24 px is the WCAG 2.2 AA floor (2.5.8),
  44×44 is the safer cross-platform target used across Apple/Material/WCAG
  AAA guidance — [testparty.ai WCAG 2.5.5 guide](https://testparty.ai/blog/wcag-2-5-5-target-size-2025-guide).
- **Safe areas:** apply `env(safe-area-inset-*)` as padding (not margin) on
  the outermost mobile shell — bottom tab bar, bottom sheets, and any
  fixed-position toast stack — [Polypane safe-area guide](https://polypane.app/blog/using-safe-area-inset-to-build-mobile-safe-layouts/).
- (e) Checklist: ☐ used the right branch (structural swap vs reflow) per
  the lists above ☐ 768px primary breakpoint documented in the component
  ☐ 44×44 min targets ☐ safe-area padding on fixed mobile chrome ☐
  container query only for context-reused components.

## 9. Theming & motion

- **Light/dark parity:** spa-b already has a full semantic token system in
  `src/shared/assets/tokens.css`/`DESIGN.md` (12-step gray + accent scales,
  hand-tuned per theme to hold WCAG AA, plus `--focus-ring` = accent-8).
  Reuse those tokens in every new component — never hardcode a hex/rgb
  value or assume one theme is "default" and the other an afterthought.
  Both scales are independently tuned, not derived, so contrast holds in
  both directions — this is already the correct 2025 practice: "the best
  products treat dark mode as a tested, tokenized system" — [muz.li dark mode systems guide](https://muz.li/blog/dark-mode-design-systems-a-complete-guide-to-patterns-tokens-and-hierarchy/).
- **Motion:** wrap all non-essential transitions in
  `@media (prefers-reduced-motion: no-preference)`; keep durations short
  and purposeful (150–250ms, opacity/transform only) — avoid animating
  blur/glow, which can trigger discomfort for vestibular-sensitive users —
  [numberanalytics reduced-motion notes](https://www.numberanalytics.com/blog/toast-notifications-best-practices-web-design).
  Reka's overlay primitives (`Dialog`/`Drawer`/`Popover`) use `Presence`
  internally for enter/exit — hook CSS transitions off the data-state
  attributes they expose (`data-state="open|closed"`) rather than
  reimplementing show/hide logic.
- **Focus-visible:** every interactive element gets a visible focus ring
  using `--focus-ring` (accent-8, already AA-tuned per `DESIGN.md`) via
  `:focus-visible`, not `:focus` (avoid showing rings on mouse click).
  Never remove the ring without replacing it with an equally visible
  alternative.
- (e) Checklist: ☐ zero hardcoded colors — semantic tokens only ☐
  `prefers-reduced-motion` respected ☐ `:focus-visible` ring uses
  `--focus-ring` token ☐ transitions keyed off Reka's `data-state`, not
  custom show/hide state.

---

## Summary table

| Pattern | Reka UI primitive | Mobile treatment | Desktop treatment |
|---|---|---|---|
| 1. App navigation | `NavigationMenuRoot` (desktop) + `DrawerRoot` (overflow) | Bottom tab bar (3–5 items) + hamburger→Drawer for overflow | Persistent left sidebar via `NavigationMenu` |
| 2. Command palette | `ComboboxRoot`+`ComboboxInput`+`ComboboxContent`+`useFilter`, in `DialogRoot` | Full-screen takeover, visible search icon trigger | Centered modal, `⌘K`/`Ctrl+K` global hotkey |
| 3. Bottom sheet / drawer | `DrawerRoot` (native, has `snapPoints`) | Bottom-anchored, snap points (peek/expand), drag handle + close btn | Side-anchored `Drawer` or `Popover` |
| 4. Dialog/Drawer/Sheet/Popover | `DialogRoot`/`AlertDialogRoot`/`DrawerRoot`/`PopoverRoot` | Near-full-width dialogs; popovers widen or become Drawer | Centered dialog; anchored popover |
| 5. Toast | `ToastProvider`+`ToastViewport`+`ToastRoot` | Bottom-center, above tab bar + safe-area, full-width minus gutters | Bottom/top-right stack |
| 6. Forms & fields | `Label`+`Checkbox`/`Switch`/`Select`/`Combobox`+custom `FormField` wrapper | Single column, `inputmode`-tuned keyboards | Multi-column grids allowed |
| 7. Data lists/tables | None — plain `<table>`/`<ul>` + `ScrollArea` | Cards (dashboards/diffs) or horizontal-scroll table w/ pinned column (dense config) | Full table |
| 8. Responsive strategy | n/a (layout-level) | Structural swap <768px: nav, sheets, palette, lists | Structural desktop branch ≥768px |
| 9. Theming & motion | `Presence`/`data-state` (internal to overlay primitives) | Same tokens, same reduced-motion rule | Same tokens, same reduced-motion rule |

## Gaps: patterns with NO Reka UI primitive

Build these from scratch on plain markup + existing primitives:

- **Bottom tab bar** (pattern 1) — plain `<nav><ul>` of router links, not a
  Reka component (it isn't a `Tabs` widget semantically, it's navigation).
- **App shell / sidebar layout** (pattern 1) — layout composition only;
  `NavigationMenu` covers the interactive nav parts, not the shell grid.
- **`FormField` wrapper** (pattern 6) — Reka has the controls (`Label`,
  `Checkbox`, `Select`, etc.) but no combined label+description+error
  field component; build one shared wrapper, reuse everywhere.
- **Data table / card list** (pattern 7) — no table or list primitive at
  all; only `ScrollArea` helps (styled scrolling container). Table
  semantics, reflow logic, and card conversion are fully custom.
- **Skeleton loaders** (pattern 7) — no skeleton primitive; build simple
  shimmer placeholders matching each layout.

Everything else (dialogs, drawers/sheets, popovers, toasts, tooltips, combobox/
command-palette, tabs, forms controls, nav menu) has a verified native Reka UI
primitive — confirmed directly against `reka-ui@2.10.5`'s docs and its
`packages/core/src/index.ts` barrel export, not from memory.
