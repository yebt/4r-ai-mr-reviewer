/**
 * Runs feature — DTO types for the global routine-run list, verified
 * against the real backend contract (`GET /routines?limit=N[&archived=1]`,
 * confirmed with live data on :8082 — see store.ts / api.ts module notes).
 */

export type RoutineKind = 'release' | 'approve_and_tag'

export type RoutineRunStatus = 'pending' | 'running' | 'blocked' | 'awaiting_confirmation' | 'done' | 'cancelled'

export type RoutineStepStatus = 'pending' | 'running' | 'done' | 'failed' | 'skipped'

export interface RoutineStep {
  name: string
  status: RoutineStepStatus
  detail: string
  updatedAt: string
}

export type RoutineFlow = 'development' | 'main'

export interface RoutineRun {
  id: string
  kind: RoutineKind
  repoId: string
  mrIid: number
  status: RoutineRunStatus
  steps: RoutineStep[]
  /**
   * Free-form per-run scratch state. `mrTitle`/`lastTag`/`nextTag` are the
   * fields this module reads (see `format.ts#runTitle`); other keys are
   * kind/flow-specific and left untyped here.
   */
  state: Record<string, unknown>
  lastError: string
  createdAt: string
  updatedAt: string
  archived: boolean
  /** Present on the global list endpoint (each run carries its repo's name). */
  repoName?: string
  /** `release`-kind runs only. */
  flow?: RoutineFlow
  sourceBranch?: string
  targetBranch?: string
}

/** `POST /routines/{id}/confirm` body's `decision` field. */
export type RoutineConfirmDecision = 'merge' | 'wait'

/**
 * Release routines — DTO/payload types for the Flow workspace's Release (dev)
 * and Release to main dialogs, verified against
 * `server/internal/http/handlers_routines.go` and
 * `server/internal/app/routines/service.go`.
 */

export type Bump = 'major' | 'minor' | 'patch'

/**
 * `POST /repos/{id}/routines/release` body (dev flow — `createReleaseRoutine`
 * / `CreateRelease`). `mrIid` must target an existing MR whose target branch
 * is `development` server-side (any other target is a 400). `bump` defaults
 * to `minor`; `emojis` defaults to `["thumbsup","seedling"]` when omitted/
 * empty; `removeSourceBranch` defaults to `false`, `mergeWhenPipelineSucceeds`
 * to `true` when omitted.
 */
export interface CreateReleaseInput {
  mrIid: number
  bump?: Bump
  emojis?: string[]
  removeSourceBranch?: boolean
  mergeWhenPipelineSucceeds?: boolean
}

/**
 * `POST /repos/{id}/routines/release-main` body (main flow —
 * `createMainReleaseRoutine` / `CreateMainRelease`). No MR needs to exist yet
 * — the routine creates one from `sourceBranch` into `targetBranch` (both
 * default to `development`/`main` server-side when omitted/blank; the two
 * must differ, or the request is a 400). `removeSourceBranch` and
 * `mergeWhenPipelineSucceeds` both default to `false` when omitted (unlike
 * the dev flow's MWPS default).
 */
export interface CreateMainReleaseInput {
  bump?: Bump
  includeDev?: boolean
  sourceBranch?: string
  targetBranch?: string
  emojis?: string[]
  removeSourceBranch?: boolean
  mergeWhenPipelineSucceeds?: boolean
}

/**
 * `GET /repos/{id}/routines/preview-tag` query (`previewRoutineTag` /
 * `PreviewNextTag`) — a dry-run of the `compute_tag` step, computing the
 * exact next tag without creating a run. `flow` is `'dev'` or `'main'`
 * (anything other than `'main'` is treated as dev server-side); `mrIid` is
 * required for dev, `source`/`target` apply to main only.
 */
export interface PreviewTagQuery {
  flow: 'dev' | 'main'
  bump: Bump
  mrIid?: number
  source?: string
  target?: string
  includeDev?: boolean
}

export interface PreviewTagResult {
  nextTag: string
  lastTag: string
  featCount: number
  fixCount: number
}
