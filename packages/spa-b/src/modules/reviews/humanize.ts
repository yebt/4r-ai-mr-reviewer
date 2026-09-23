/**
 * Pure helpers for the "humanize" feature (slice 2) — rewriting a review's
 * summary/findings in a profile's author voice, and reassembling a humanized
 * finding back into the exact markdown body the server would have generated
 * itself (`formatFinding` in `packages/server/internal/app/reviews/publish.go`),
 * so a finding override posted to the MR is byte-for-byte compatible with an
 * un-humanized publish. Kept dependency-free (no store/component imports),
 * matching `findings.ts`/`publish.ts`.
 */
import type { Finding, ReviewFindingDimension } from './types'

/** Sentinel active-tab value meaning "the generated original, no override". */
export const ORIGINAL = -1

/** Per-dimension header label, matching the server's `dimensionLabel` map. */
export const DIMENSION_LABELS: Record<ReviewFindingDimension, string> = {
  risk: 'R1 Risk',
  readability: 'R2 Readability',
  reliability: 'R3 Reliability',
  resilience: 'R4 Resilience',
}

/**
 * Rebuilds a finding's MR comment body from humanized parts, using the exact
 * structure of the server's `formatFinding` — dimension/severity/blocking
 * come from the original `finding` (never humanized), issue/why/fix come
 * from `parts` (the active humanized tab, or the finding's own text for the
 * "Original" tab). Must stay byte-for-byte identical to the Go source: any
 * drift here changes what gets posted to the live MR.
 */
export function buildFindingBody(finding: Finding, parts: { issue: string; why: string; fix: string }): string {
  let body = `**[${DIMENSION_LABELS[finding.dimension]} · ${finding.severity.toUpperCase()}]** ${parts.issue}\n\n`
  if (parts.why) {
    body += `**Why:** ${parts.why}\n\n`
  }
  if (parts.fix) {
    body += `**Suggested fix:** ${parts.fix}\n`
  }
  if (finding.blocking) {
    body += `\n_Blocking._`
  }
  return body
}
