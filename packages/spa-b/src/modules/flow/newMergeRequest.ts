/**
 * Flow module — pure helpers behind `NewMergeRequestDialog.vue`: default
 * target-branch/voice-profile resolution, form validity, and dirty-draft
 * detection. Provider/model defaults are NOT duplicated here — the dialog
 * reuses `resolveDefaultProviderId`/`resolveDefaultModel` from
 * `./reviewLaunch` (same repo → provider/model precedence as launching a
 * review, see that file's doc).
 */
import type { Profile } from '@modules/profiles/types'
import type { Provider } from '@modules/providers/types'
import type { Repo } from '@modules/repos/types'

/** Conventional branch names to try, in priority order, when picking the
 * target branch's initial value. */
const TARGET_BRANCH_CANDIDATES = ['development', 'main', 'master']

/**
 * The target branch Combobox's initial value once the repo's branches load:
 * the first conventional name (`development` → `main` → `master`) that
 * actually exists on the repo, else its first branch, else `''` (nothing to
 * preselect — the picker starts unselected rather than guessing). Source
 * never gets a default: the user always picks the branch they're merging
 * (mirrors the old app's `CreateMergeRequestModal`, which only preselects
 * the target).
 */
export function resolveDefaultTargetBranch(branches: string[]): string {
  for (const candidate of TARGET_BRANCH_CANDIDATES) {
    if (branches.includes(candidate)) return candidate
  }
  return branches[0] ?? ''
}

/**
 * The voice-profile Select's initial value: the repo's own
 * `defaultProfileId` when it's still configured AND its style guide is
 * `ready` (an unready guide can't voice a draft — the backend rejects it
 * with `ErrStyleGuideNotReady`/409), else `''` ("No voice profile").
 */
export function resolveDefaultProfileId(repo: Repo, profiles: Profile[]): string {
  if (!repo.defaultProfileId) return ''
  const profile = profiles.find((p) => p.id === repo.defaultProfileId)
  return profile && profile.styleGuideStatus === 'ready' ? profile.id : ''
}

/**
 * Only profiles whose style guide has finished distilling can voice a
 * draft — the rest are hidden from the picker entirely (mirrors the old
 * app's `CreateMergeRequestModal#voiceProfiles` filter and the backend's
 * own 409 rejection for an unready profile).
 */
export function readyVoiceProfiles(profiles: Profile[]): Profile[] {
  return profiles.filter((profile) => profile.styleGuideStatus === 'ready')
}

export interface MergeRequestBranches {
  sourceBranch: string
  targetBranch: string
}

/** Both branches are picked and identical. */
export function isSameBranch(branches: MergeRequestBranches): boolean {
  return !!branches.sourceBranch && branches.sourceBranch === branches.targetBranch
}

export interface MergeRequestDraft extends MergeRequestBranches {
  title: string
}

/**
 * Whether the form has everything `createMergeRequest` requires: both
 * branches picked, different, and a non-blank title (the backend's own
 * `Create` validation — see `mergerequests/service.go`).
 */
export function isMergeRequestFormValid(draft: MergeRequestDraft): boolean {
  return (
    !!draft.sourceBranch.trim() &&
    !!draft.targetBranch.trim() &&
    !isSameBranch(draft) &&
    !!draft.title.trim()
  )
}

export interface DraftState {
  sourceBranch: string
  title: string
  description: string
}

/**
 * Whether the dialog holds work worth protecting from an accidental close.
 * The auto-preselected target branch alone never counts — only a source
 * pick, a title, or a description mean the user actually did something
 * (mirrors the old app's `CreateMergeRequestModal#isDirty`, which only
 * checks `source`/`title`/`description`, not the preselected `target`).
 * A source branch restored from the remembered setup (`baselineSource`) is
 * not the user's doing either, so it only counts once it changes.
 */
export function isDraftDirty(state: DraftState, baselineSource = ''): boolean {
  const source = state.sourceBranch.trim()
  return (!!source && source !== baselineSource) || !!state.title.trim() || !!state.description.trim()
}

/**
 * Whether "Generate with AI" would overwrite text the user already has —
 * either typed by hand or left over from an earlier generation. Drives the
 * overwrite confirm before a regeneration silently replaces it.
 */
export function hasDraftText(state: { title: string; description: string }): boolean {
  return !!state.title.trim() || !!state.description.trim()
}

/**
 * The dialog's setup choices (everything above the draft), remembered per
 * repo so reopening the dialog — or coming back to the workspace — picks up
 * where the user left off. The draft title/description are never remembered.
 */
export interface MergeRequestSetup {
  sourceBranch: string
  targetBranch: string
  profileId: string
  providerId: string
  model: string
}

const SETUP_STORAGE_PREFIX = '4r:new-mr-setup:'

function isSetup(value: unknown): value is MergeRequestSetup {
  if (!value || typeof value !== 'object') return false
  const record = value as Record<string, unknown>
  return ['sourceBranch', 'targetBranch', 'profileId', 'providerId', 'model'].every(
    (key) => typeof record[key] === 'string',
  )
}

/** The remembered setup for `repoId`, or `null` when absent, corrupt, or storage is unavailable. */
export function loadMergeRequestSetup(repoId: string): MergeRequestSetup | null {
  try {
    const raw = window.localStorage.getItem(SETUP_STORAGE_PREFIX + repoId)
    if (!raw) return null
    const parsed: unknown = JSON.parse(raw)
    return isSetup(parsed) ? parsed : null
  } catch {
    return null
  }
}

/** Remembers `setup` for `repoId`; silently a no-op when storage is unavailable. */
export function saveMergeRequestSetup(repoId: string, setup: MergeRequestSetup): void {
  try {
    window.localStorage.setItem(SETUP_STORAGE_PREFIX + repoId, JSON.stringify(setup))
  } catch {
    // Remembering the setup is a convenience; never break the dialog over it.
  }
}

/** A remembered branch, only while it still exists on the repo; else `''`. */
export function restoreBranch(saved: string, branches: string[]): string {
  return saved && branches.includes(saved) ? saved : ''
}

/**
 * The remembered profile/provider/model, validated against what exists now.
 * The profile falls back to `''` ("No voice profile") when it's gone or no
 * longer ready. `providerId`/`model` are `null` when the provider is gone,
 * so the caller falls back to the repo defaults; a model the provider no
 * longer lists falls back to `''` (provider default).
 */
export function restoreSetupChoices(
  saved: MergeRequestSetup,
  profiles: Profile[],
  providers: Provider[],
): { profileId: string; providerId: string | null; model: string | null } {
  const profileId = readyVoiceProfiles(profiles).some((p) => p.id === saved.profileId) ? saved.profileId : ''
  const provider = providers.find((p) => p.id === saved.providerId)
  if (!provider) return { profileId, providerId: null, model: null }
  return { profileId, providerId: provider.id, model: provider.models.includes(saved.model) ? saved.model : '' }
}
