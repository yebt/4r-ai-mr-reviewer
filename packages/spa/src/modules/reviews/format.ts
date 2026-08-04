import type { Dimension, Recommendation, ReviewStatus, Severity } from '@shared/api/types'

export const statusClass: Record<ReviewStatus, string> = {
  awaiting_approval: 'text-warn',
  pending: 'text-muted',
  running: 'text-accent',
  done: 'text-ok',
  error: 'text-danger',
  cancelled: 'text-muted',
}

// Human-readable label for a review status. Most statuses read fine as their raw
// key; awaiting_approval is spelled out so the chip never shows raw snake_case.
export const statusLabel: Record<ReviewStatus, string> = {
  awaiting_approval: 'Awaiting approval',
  pending: 'pending',
  running: 'running',
  done: 'done',
  error: 'error',
  cancelled: 'cancelled',
}

export const severityClass: Record<Severity, string> = {
  high: 'text-danger',
  medium: 'text-warn',
  low: 'text-muted',
}

// Sort/triage weight for a severity — higher is more severe. Used to order the
// triage list (most severe first) and to pick each finding's left-border color.
export const severityRank: Record<Severity, number> = {
  high: 3,
  medium: 2,
  low: 1,
}

export const dimensionLabel: Record<Dimension, string> = {
  risk: 'R1 Risk',
  readability: 'R2 Readability',
  reliability: 'R3 Reliability',
  resilience: 'R4 Resilience',
}

// Short display label for a captured-reasoning phase. Mirrors the 4R lens names
// used by the progress indicator; unknown phases fall back to a capitalized form
// so nothing renders as a raw lowercase key.
export function phaseLabel(phase: string): string {
  switch (phase) {
    case 'risk':
      return 'Risk'
    case 'readability':
      return 'Readability'
    case 'reliability':
      return 'Reliability'
    case 'resilience':
      return 'Resilience'
    default:
      return phase ? phase.charAt(0).toUpperCase() + phase.slice(1) : ''
  }
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
  return status === 'done' || status === 'error' || status === 'cancelled'
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
