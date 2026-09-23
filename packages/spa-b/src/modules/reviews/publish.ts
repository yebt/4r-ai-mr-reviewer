/**
 * Pure gating helpers for the "publish to the live MR" feature — used by the
 * detail page to decide which findings still need a Publish control and
 * whether the "Publish all" action has anything to do. Kept dependency-free
 * (no store/component imports) so they're trivial to unit test, matching the
 * style of `findings.ts`.
 */
import type { Finding, Review } from './types'

/** Indices of findings that have not been published to the MR yet. */
export function unpublishedFindingIndices(findings: Finding[]): number[] {
  return findings.filter((finding) => !finding.published).map((finding) => finding.index)
}

/** True when the review has at least one unpublished finding or an unpublished summary. */
export function hasUnpublished(review: Pick<Review, 'findings' | 'summaryPublished'>): boolean {
  return !review.summaryPublished || review.findings.some((finding) => !finding.published)
}
