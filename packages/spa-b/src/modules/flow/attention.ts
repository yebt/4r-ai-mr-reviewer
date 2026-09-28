/**
 * Flow module — pure "needs a human" attention-count helper, computed over
 * the app's already-loaded global lists (`useReviewsStore().reviews` /
 * `useRunsStore().runs`, both auto-fetch their first recent-first page on
 * store instantiation — see those stores' module docs). A repo needs
 * attention when it has a review sitting `awaiting_approval`/`error` or a
 * run sitting `awaiting_confirmation`/`blocked`, the same statuses the home
 * dashboard's `modules/runs/dashboard.ts#attentionRuns` and the old app's
 * Flow picker (`packages/spa/src/pages/flow/index.vue#repoAttention`) use.
 *
 * LIMITATION: both source lists are the global *recent* page (30 rows each,
 * cursor-paginated), not each repo's full history — an attention-worthy row
 * buried past the first page won't be counted here. Good enough for a
 * cheap triage badge on the repo picker; the repo's own Reviews/Actions
 * tabs (`modules/flow/components/RepoReviewsSection.vue`,
 * `RepoActionsSection.vue`) fetch the complete per-repo list instead.
 */
import type { Review, ReviewStatus } from '@modules/reviews'
import type { RoutineRun, RoutineRunStatus } from '@modules/runs'

const ATTENTION_REVIEW_STATUSES: ReadonlySet<ReviewStatus> = new Set(['awaiting_approval', 'error'])
const ATTENTION_RUN_STATUSES: ReadonlySet<RoutineRunStatus> = new Set(['awaiting_confirmation', 'blocked'])

/**
 * Attention count per repo id, built in one pass over each list — prefer
 * this over calling a per-repo count in a loop over every repo (an
 * O(repos × rows) scan), which is exactly what the repo picker page would
 * otherwise do in its `v-for`.
 */
export function attentionCountsByRepo(
  reviews: Pick<Review, 'repoId' | 'status'>[],
  runs: Pick<RoutineRun, 'repoId' | 'status'>[],
): Map<string, number> {
  const counts = new Map<string, number>()
  const bump = (repoId: string) => counts.set(repoId, (counts.get(repoId) ?? 0) + 1)

  for (const review of reviews) {
    if (ATTENTION_REVIEW_STATUSES.has(review.status)) bump(review.repoId)
  }
  for (const run of runs) {
    if (ATTENTION_RUN_STATUSES.has(run.status)) bump(run.repoId)
  }

  return counts
}
