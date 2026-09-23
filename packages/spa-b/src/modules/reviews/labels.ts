/** Shared display labels, used by both `ReviewsListSection` and the detail page. */
import type { ReviewRecommendation } from './types'

export const RECOMMENDATION_LABELS: Record<ReviewRecommendation, string> = {
  approve: 'Approve',
  request_changes: 'Request changes',
  comment: 'Comment',
}
