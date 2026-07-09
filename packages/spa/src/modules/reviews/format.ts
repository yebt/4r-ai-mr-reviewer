import type { Dimension, Recommendation, ReviewStatus, Severity } from '@shared/api/types'

export const statusClass: Record<ReviewStatus, string> = {
  pending: 'text-muted',
  running: 'text-accent',
  done: 'text-ok',
  error: 'text-danger',
}

export const severityClass: Record<Severity, string> = {
  high: 'text-danger',
  medium: 'text-warn',
  low: 'text-muted',
}

export const dimensionLabel: Record<Dimension, string> = {
  risk: 'R1 Risk',
  readability: 'R2 Readability',
  reliability: 'R3 Reliability',
  resilience: 'R4 Resilience',
}

export const recommendationClass: Record<Recommendation, string> = {
  approve: 'text-ok',
  request_changes: 'text-danger',
  comment: 'text-warn',
}

export function recommendationLabel(r: Recommendation): string {
  switch (r) {
    case 'approve':
      return 'Approve'
    case 'request_changes':
      return 'Request changes'
    default:
      return 'Comment'
  }
}

export function isTerminal(status: ReviewStatus): boolean {
  return status === 'done' || status === 'error'
}
