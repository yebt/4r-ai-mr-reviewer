/**
 * Home dashboard — pure selectors over the global runs list. Kept
 * Vue/store-free (no Pinia/@pinia/colada import) so they can be unit-tested
 * directly and reused from `src/pages/index.vue` without a store context.
 */
import { isRunActive } from './format'
import type { RoutineRun, RoutineRunStatus } from './types'

/**
 * Statuses that put a run in the "needs a human" bucket — the same two
 * statuses the run detail page (`src/pages/runs/[id].vue`) gates its
 * Resume/Skip (`blocked`) and Merge/Wait (`awaiting_confirmation`) actions
 * on.
 */
const ATTENTION_STATUSES: readonly RoutineRunStatus[] = ['blocked', 'awaiting_confirmation']

function needsAttention(status: RoutineRunStatus): boolean {
  return ATTENTION_STATUSES.includes(status)
}

export interface RunStats {
  total: number
  active: number
  attention: number
}

/**
 * Dashboard stat-row counts. Only the tiles the dashboard actually renders
 * (Total / Active / Needs attention) are computed here — a "Completed" or
 * "Cancelled" tile would be added with its own correct semantics when the
 * design calls for it, rather than mislabelling `cancelled` as "failed"
 * (`RoutineRunStatus` has no dedicated failed run status).
 *
 * `active` mirrors `isRunActive` (pending/running/blocked/
 * awaiting_confirmation — i.e. still progressing on the server *or* resting
 * on a human), so it overlaps with `attention` by design: these are
 * independent tile counts, not a partition of `total`.
 */
export function runStats(runs: RoutineRun[]): RunStats {
  let active = 0
  let attention = 0

  for (const run of runs) {
    if (isRunActive(run.status)) active++
    if (needsAttention(run.status)) attention++
  }

  return { total: runs.length, active, attention }
}

/** Runs currently waiting on a human — `blocked` or `awaiting_confirmation`. */
export function attentionRuns(runs: RoutineRun[]): RoutineRun[] {
  return runs.filter((run) => needsAttention(run.status))
}

/**
 * First `n` runs for the "Recent activity" section. The `['routines']`
 * query already returns the list newest-first (see
 * `api.ts#listRecentRoutines`), so this only slices — it never re-sorts.
 */
export function recentRuns(runs: RoutineRun[], n: number): RoutineRun[] {
  return runs.slice(0, n)
}
