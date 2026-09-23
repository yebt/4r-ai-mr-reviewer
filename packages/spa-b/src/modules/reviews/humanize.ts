/**
 * Pure helpers for the "humanize" feature (slice 2) — rewriting a review's
 * summary/findings in a profile's author voice, and reassembling a humanized
 * finding back into the exact markdown body the server would have generated
 * itself (`formatFinding` in `packages/server/internal/app/reviews/publish.go`),
 * so a finding override posted to the MR is byte-for-byte compatible with an
 * un-humanized publish. Also (U1) `buildFindingMarkdown`, a standalone
 * Markdown rendering of a finding for the per-card copy action. Kept
 * dependency-free (no store/component imports), matching `findings.ts`/
 * `publish.ts`.
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

/**
 * Assembles a finding as standalone Markdown for the copy-to-clipboard
 * action (U1) — unlike `buildFindingBody` (an MR comment body, posted
 * inline on the file so it omits the location), this includes the
 * `file:line` location since a copied snippet has no surrounding context.
 * Mirrors the old spa's `buildFindingMarkdown`
 * (`packages/spa/src/modules/reviews/humanize-overrides.ts`) byte-for-byte,
 * ported to this package's `DIMENSION_LABELS`/`Finding` shape. `parts` is
 * the active tab's text (Original or a humanize run) — same contract as
 * `buildFindingBody`.
 */
export function buildFindingMarkdown(finding: Finding, parts: { issue: string; why: string; fix: string }): string {
  let markdown = `**[${DIMENSION_LABELS[finding.dimension]} · ${finding.severity.toUpperCase()}]** ${parts.issue}\n`
  if (finding.file) {
    markdown += `\n\`${finding.file}${finding.line > 0 ? `:${finding.line}` : ''}\`\n`
  }
  if (parts.why) {
    markdown += `\n**Why:** ${parts.why}\n`
  }
  if (parts.fix) {
    markdown += `\n**Suggested fix:** ${parts.fix}\n`
  }
  if (finding.blocking) {
    markdown += `\n_Blocking._\n`
  }
  return markdown
}
