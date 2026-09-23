import type { ReviewStatus, ReviewWithRepo } from './types'

export type ReviewFilterKey = 'all' | 'awaiting_approval' | 'running' | 'done' | 'error'

export interface ReviewFilterDef {
  key: ReviewFilterKey
  label: string
  /** `null` for the "All" tab, which matches every row regardless of status. */
  status: ReviewStatus | null
}

/**
 * The 5 filter tabs `ReviewsListSection` renders. `pending`/`cancelled` rows
 * have no dedicated tab (per the spec) — they're only ever visible under
 * "All".
 */
export const REVIEW_FILTERS: ReviewFilterDef[] = [
  { key: 'all', label: 'All', status: null },
  { key: 'awaiting_approval', label: 'Awaiting', status: 'awaiting_approval' },
  { key: 'running', label: 'Running', status: 'running' },
  { key: 'done', label: 'Done', status: 'done' },
  { key: 'error', label: 'Error', status: 'error' },
]

/** Pure client-side filter of an already fan-out'd review list by tab key. */
export function filterReviewsByKey(reviews: ReviewWithRepo[], key: ReviewFilterKey): ReviewWithRepo[] {
  const filter = REVIEW_FILTERS.find((f) => f.key === key)
  if (!filter || filter.status === null) return reviews
  return reviews.filter((review) => review.status === filter.status)
}
