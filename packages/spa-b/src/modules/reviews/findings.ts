/**
 * Pure display helpers for a `Review`'s `findings[]` — used by the detail
 * page to group findings by dimension and map severity to a `Badge` status.
 * Kept dependency-free (no store/component imports) so they're trivial to
 * unit test.
 */
import type { BadgeStatus } from '@shared/ui/design-system'
import type { Finding, ReviewFindingDimension, ReviewFindingSeverity } from './types'

/** Render order for the detail page's findings-by-dimension sections. */
export const FINDING_DIMENSIONS: ReviewFindingDimension[] = ['risk', 'readability', 'reliability', 'resilience']

/** Groups a flat `findings[]` list into one bucket per dimension, in `FINDING_DIMENSIONS` order. */
export function groupFindingsByDimension(findings: Finding[]): Record<ReviewFindingDimension, Finding[]> {
  const grouped: Record<ReviewFindingDimension, Finding[]> = {
    risk: [],
    readability: [],
    reliability: [],
    resilience: [],
  }
  for (const finding of findings) {
    grouped[finding.dimension].push(finding)
  }
  return grouped
}

/** high -> danger, medium -> warning, low -> neutral. */
export const FINDING_SEVERITY_BADGE: Record<ReviewFindingSeverity, BadgeStatus> = {
  high: 'danger',
  medium: 'warning',
  low: 'neutral',
}
