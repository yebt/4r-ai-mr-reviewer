/**
 * Pure helpers for rendering a `Preflight` result (see `./types`) —
 * extracted from `PreflightDialog.vue` so the status → chip mapping and the
 * ok/fail/unknown summary counts stay unit-testable without mounting a
 * component. Mirrors the old app's `modules/repos/preflight.ts`
 * (`checkStatusUi`), adapted to this app's `Badge` status vocabulary
 * (`success`/`warning`/`danger` instead of custom `text-ok`/`text-warn`/
 * `text-danger` classes).
 */
import type { BadgeStatus } from '@shared/ui/design-system'
import type { PreflightCheck } from './types'

/** Visual mapping for one preflight check status. */
export interface CheckStatusUi {
  badgeStatus: BadgeStatus
  icon: string
  srLabel: string
}

export const checkStatusUi: Record<PreflightCheck['status'], CheckStatusUi> = {
  ok: { badgeStatus: 'success', icon: 'check', srLabel: 'Allowed' },
  fail: { badgeStatus: 'danger', icon: 'x', srLabel: 'Blocked' },
  unknown: { badgeStatus: 'warning', icon: 'triangle-alert', srLabel: 'Unknown' },
}

/** Count of checks per status, for a one-line summary above the check list. */
export interface PreflightSummary {
  ok: number
  fail: number
  unknown: number
}

export function summarizePreflightChecks(checks: PreflightCheck[]): PreflightSummary {
  const summary: PreflightSummary = { ok: 0, fail: 0, unknown: 0 }
  for (const check of checks) {
    summary[check.status] += 1
  }
  return summary
}
