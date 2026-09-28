/**
 * Flow module — pure helpers behind `ReleaseDialog.vue` (dev + main release
 * flows). Ported from the old app's `packages/spa/src/modules/routines/
 * components/ReleaseModal.vue` (bump defaults, emoji handling, branch
 * defaults/warnings), reshaped into standalone functions the dialog composes
 * instead of inline refs/watchers, so the payload-building and emoji-name
 * rules are unit-testable independent of Reka/@pinia-colada wiring.
 */
import type { CreateMainReleaseInput, CreateReleaseInput, Bump } from '@modules/runs/types'

export type ReleaseFlow = 'dev' | 'main'

/** Curated quick-pick reaction emojis — GitLab award-emoji NAMES (not
 * unicode), ported verbatim from the old app's `EMOJI_CHOICES`. */
export const EMOJI_CHOICES: { name: string; glyph: string }[] = [
  { name: 'thumbsup', glyph: '👍' },
  { name: 'seedling', glyph: '🌱' },
  { name: 'rocket', glyph: '🚀' },
  { name: 'tada', glyph: '🎉' },
  { name: 'fire', glyph: '🔥' },
  { name: 'white_check_mark', glyph: '✅' },
  { name: 'eyes', glyph: '👀' },
  { name: 'heart', glyph: '❤️' },
  { name: 'sparkles', glyph: '✨' },
  { name: 'raised_hands', glyph: '🙌' },
]

/** Default selection when a release dialog opens — matches the old app and
 * the backend's own fallback (`routines/service.go`'s `CreateRelease`/
 * `CreateMainRelease` apply this exact pair when `emojis` is omitted/empty). */
export const DEFAULT_EMOJIS = ['thumbsup', 'seedling']

/** Matches the backend's `maxEmojis` bound in `packages/server/internal/app/
 * routines/service.go` (`CreateRelease`/`CreateMainRelease`/`ApproveAndReact`
 * all reject `len(in.Emojis) > maxEmojis` with a 400) — kept in sync here so
 * the dialog refuses a selection the backend would reject anyway, instead of
 * only surfacing it as a submit error. */
export const MAX_RELEASE_EMOJIS = 10

/** GitLab award-emoji names are lowercase snake/kebab tokens — reject
 * anything else before it ever reaches the API. */
const EMOJI_NAME_PATTERN = /^[a-z0-9_+-]+$/

export function isValidEmojiName(name: string): boolean {
  return EMOJI_NAME_PATTERN.test(name)
}

/** Strips surrounding whitespace and `:colon:` wrapping a user might paste
 * (e.g. `:rocket:` → `rocket`) — ported from the old app's `addCustomEmoji`. */
export function normalizeCustomEmojiName(raw: string): string {
  return raw.trim().replace(/^:|:$/g, '')
}

/** Toggles one emoji name in/out of a selection (chip click) — returns a new
 * array; never mutates `selected`. Deselecting always works, even at the
 * cap; selecting a new (not-yet-selected) name past `MAX_RELEASE_EMOJIS`
 * returns `selected` unchanged instead of growing past what the backend
 * would accept. */
export function toggleEmoji(selected: string[], name: string): string[] {
  if (selected.includes(name)) return selected.filter((n) => n !== name)
  if (selected.length >= MAX_RELEASE_EMOJIS) return selected
  return [...selected, name]
}

/** Adds one emoji name to a selection if not already present (custom-name
 * input's "Add" action — never removes, unlike `toggleEmoji`). Refuses to
 * grow the selection past `MAX_RELEASE_EMOJIS`, returning `selected`
 * unchanged instead. */
export function addEmoji(selected: string[], name: string): string[] {
  if (selected.includes(name)) return selected
  if (selected.length >= MAX_RELEASE_EMOJIS) return selected
  return [...selected, name]
}

/** The selected names that aren't one of the curated `EMOJI_CHOICES` — these
 * render as their own removable chips. */
export function customEmojis(selected: string[]): string[] {
  return selected.filter((name) => !EMOJI_CHOICES.some((choice) => choice.name === name))
}

export interface ReleaseFormState {
  bump: Bump
  /** Main flow only — base on the highest tag INCLUDING "-dev" prereleases. */
  includeDev: boolean
  /** Main flow only — empty means "use the backend default" (development). */
  sourceBranch: string
  /** Main flow only — empty means "use the backend default" (main). */
  targetBranch: string
  emojis: string[]
  removeSourceBranch: boolean
  mergeWhenPipelineSucceeds: boolean
}

/**
 * `mergeWhenPipelineSucceeds`'s per-flow default: `true` for dev (mirrors
 * the old app and the backend's own `ReleaseInput` default), `false` for
 * main (the main flow already gates on a green pipeline in `wait_pipeline`,
 * so an immediate all-or-nothing merge of the pinned head is safer than
 * scheduling another async MWPS merge — see `routines/service.go`'s
 * `createMainReleaseRoutine` doc comment).
 */
export function defaultMergeWhenPipelineSucceeds(flow: ReleaseFlow): boolean {
  return flow === 'dev'
}

/** Fresh per-flow defaults, applied every time the dialog opens (or targets
 * a different flow) so a previous edit never leaks into the next release. */
export function defaultReleaseFormState(flow: ReleaseFlow): ReleaseFormState {
  return {
    bump: 'minor',
    includeDev: false,
    sourceBranch: '',
    targetBranch: '',
    emojis: [...DEFAULT_EMOJIS],
    removeSourceBranch: false,
    mergeWhenPipelineSucceeds: defaultMergeWhenPipelineSucceeds(flow),
  }
}

/** The old app's `missingConventional` — which of `development`/`main` the
 * repo's branch list is missing (main flow only warns; the routine assumes
 * both exist and its picker should offer alternatives when they don't). */
export function missingConventionalBranches(branches: string[]): string[] {
  return ['development', 'main'].filter((branch) => !branches.includes(branch))
}

/** The main flow's initial source/target picks once branches load — the
 * conventional name when present, else empty (the picker starts unselected
 * rather than guessing a branch that doesn't exist). */
export function resolveDefaultBranches(branches: string[]): { source: string; target: string } {
  return {
    source: branches.includes('development') ? 'development' : '',
    target: branches.includes('main') ? 'main' : '',
  }
}

/**
 * Builds the dev-flow release request body from form state. `emojis` is
 * only included when non-empty — an empty selection omits the field so the
 * backend applies its own default (`thumbsup`, `seedling`) rather than the
 * dialog sending an explicit empty array (see `CreateRelease` in
 * `routines/service.go`).
 */
export function buildDevReleasePayload(form: ReleaseFormState, mrIid: number): CreateReleaseInput {
  return {
    mrIid,
    bump: form.bump,
    ...(form.emojis.length > 0 ? { emojis: form.emojis } : {}),
    removeSourceBranch: form.removeSourceBranch,
    mergeWhenPipelineSucceeds: form.mergeWhenPipelineSucceeds,
  }
}

/** Builds the main-flow release request body from form state. Branch names
 * are trimmed and sent empty when blank, so the backend's own
 * development/main defaults apply (`CreateMainRelease`). */
export function buildMainReleasePayload(form: ReleaseFormState): CreateMainReleaseInput {
  return {
    bump: form.bump,
    includeDev: form.includeDev,
    sourceBranch: form.sourceBranch.trim(),
    targetBranch: form.targetBranch.trim(),
    ...(form.emojis.length > 0 ? { emojis: form.emojis } : {}),
    removeSourceBranch: form.removeSourceBranch,
    mergeWhenPipelineSucceeds: form.mergeWhenPipelineSucceeds,
  }
}
