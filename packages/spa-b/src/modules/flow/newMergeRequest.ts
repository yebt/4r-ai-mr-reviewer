/**
 * Flow module — pure helpers behind `NewMergeRequestDialog.vue`: default
 * target-branch/voice-profile resolution, form validity, and dirty-draft
 * detection. Provider/model defaults are NOT duplicated here — the dialog
 * reuses `resolveDefaultProviderId`/`resolveDefaultModel` from
 * `./reviewLaunch` (same repo → provider/model precedence as launching a
 * review, see that file's doc).
 */
import type { Profile } from '@modules/profiles/types'
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
 */
export function isDraftDirty(state: DraftState): boolean {
  return !!state.sourceBranch.trim() || !!state.title.trim() || !!state.description.trim()
}

/**
 * Whether "Generate with AI" would overwrite text the user already has —
 * either typed by hand or left over from an earlier generation. Drives the
 * overwrite confirm before a regeneration silently replaces it.
 */
export function hasDraftText(state: { title: string; description: string }): boolean {
  return !!state.title.trim() || !!state.description.trim()
}
