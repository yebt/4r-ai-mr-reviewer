/**
 * Flow module — pure helper mapping a repo's reviews to each merge
 * request's most recent review, keyed by `mrIid`. Used by
 * `MergeRequestListSection` to show a best-effort review status per MR row
 * (ported from the old app's `packages/spa/src/pages/flow/[repoId].vue`'s
 * `reviewByMr`).
 */
import type { Review } from '@modules/reviews'

/**
 * Query key for a repo's open merge requests list (`GET
 * /repos/{id}/merge-requests`) — exported so anything that mutates that
 * list (`NewMergeRequestDialog`, after creating one) invalidates the exact
 * same @pinia/colada cache entry `MergeRequestListSection` reads, instead
 * of duplicating the literal key and risking drift.
 */
export function repoMergeRequestsQueryKey(repoId: string): readonly [string, string] {
  return ['flow-merge-requests', repoId] as const
}

export function latestReviewByMr(reviews: Review[]): Map<number, Review> {
  const latest = new Map<number, Review>()
  for (const review of reviews) {
    const current = latest.get(review.mrIid)
    if (!current || review.createdAt > current.createdAt) {
      latest.set(review.mrIid, review)
    }
  }
  return latest
}
