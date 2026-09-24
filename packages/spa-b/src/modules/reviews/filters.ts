/**
 * Reviews list filters — repo + status `Select` options and a pure
 * client-side filter, mirroring `modules/runs/filters.ts`'s pure-function
 * shape so the filtering logic is unit-testable without driving Reka UI's
 * `Select` primitive through jsdom (portals + internal state make that
 * brittle).
 *
 * Replaces the earlier tab-style `REVIEW_FILTERS`/`filterReviewsByKey`
 * (5 status tabs, no repo axis) with a repo + status `Select` bar — the same
 * shape `RunsListSection` uses — so reviews can be filtered by the repo
 * they belong to as well as by any `ReviewStatus`, not just the subset the
 * tabs covered.
 */
import type { ReviewStatus, ReviewWithRepo } from './types'

/** Reka UI's `Select` reserves the empty string for its own internal state, so both sentinels use a non-empty value — same convention as `modules/runs/filters.ts`. */
export const ALL_REPOS_VALUE = 'all'
export const ALL_STATUSES_VALUE = 'all'

export interface ReviewFilterOption {
  label: string
  value: string
}

/** Human labels for every `ReviewStatus` — mirrors `ReviewStatusChip`'s `labelMap`. */
const REVIEW_STATUS_LABELS: Record<ReviewStatus, string> = {
  awaiting_approval: 'Awaiting approval',
  pending: 'Pending',
  running: 'Running',
  done: 'Done',
  error: 'Error',
  cancelled: 'Cancelled',
}

/** Every `ReviewStatus`, in `types.ts` declaration order, as Select items — "All statuses" first. */
export const REVIEW_STATUS_FILTER_OPTIONS: ReviewFilterOption[] = [
  { label: 'All statuses', value: ALL_STATUSES_VALUE },
  ...(Object.keys(REVIEW_STATUS_LABELS) as ReviewStatus[]).map((status) => ({
    label: REVIEW_STATUS_LABELS[status],
    value: status,
  })),
]

/**
 * Repo `Select` options, "All repositories" first, sorted by name. Built
 * from the distinct repos actually present in the (already fan-out'd)
 * reviews list — unlike runs' `repoFilterOptions`, reviews has no
 * `useReposStore` fallback list to prefer, since every review already
 * carries its owning repo's id + name off `ReviewWithRepo`.
 */
export function repoFilterOptions(reviews: ReviewWithRepo[]): ReviewFilterOption[] {
  const seen = new Map<string, string>()
  for (const review of reviews) {
    if (!seen.has(review.repoId)) seen.set(review.repoId, review.repoName)
  }
  const repos = [...seen.entries()]
    .map(([value, label]) => ({ value, label }))
    .sort((a, b) => a.label.localeCompare(b.label))
  return [{ label: 'All repositories', value: ALL_REPOS_VALUE }, ...repos]
}

export interface ReviewFilters {
  repoId: string
  status: string
}

/** Pure client-side filter of the (already fan-out'd) reviews list by repo + status. The `ALL_*` sentinel on either matches everything. */
export function filterReviews(reviews: ReviewWithRepo[], filters: ReviewFilters): ReviewWithRepo[] {
  return reviews.filter((review) => {
    if (filters.repoId !== ALL_REPOS_VALUE && review.repoId !== filters.repoId) return false
    if (filters.status !== ALL_STATUSES_VALUE && review.status !== filters.status) return false
    return true
  })
}
