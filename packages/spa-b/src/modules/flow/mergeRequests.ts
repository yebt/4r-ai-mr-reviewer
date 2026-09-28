/**
 * Flow module — pure helper mapping a repo's reviews to each merge
 * request's most recent review, keyed by `mrIid`. Used by
 * `MergeRequestListSection` to show a best-effort review status per MR row
 * (ported from the old app's `packages/spa/src/pages/flow/[repoId].vue`'s
 * `reviewByMr`).
 */
import type { Review } from '@modules/reviews'

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
