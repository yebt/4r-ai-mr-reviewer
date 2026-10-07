import type { ReviewStatus } from './types'

/**
 * What the Findings section says when a review has no findings. "No findings."
 * is a result, so it is only true once the review is done; every other state
 * explains why nothing is listed yet (or at all).
 */
export function emptyFindingsMessage(status: ReviewStatus): string {
  switch (status) {
    case 'done':
      return 'No findings.'
    case 'error':
      return 'The review failed before it produced findings.'
    case 'cancelled':
      return 'The review was cancelled before it produced findings.'
    case 'awaiting_approval':
      return 'This review is waiting for approval to start. Findings appear once it has run.'
    case 'running':
    case 'pending':
      return 'The review is still in progress. Findings appear when it finishes.'
  }
}
