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
