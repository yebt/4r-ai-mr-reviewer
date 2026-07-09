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

/** First 8 chars of a review id, to tell same-MR reviews apart. */
export function shortId(id: string): string {
  return id.slice(0, 8)
}

export function formatDateTime(iso: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  return d.toLocaleString(undefined, {
    month: 'short',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}
