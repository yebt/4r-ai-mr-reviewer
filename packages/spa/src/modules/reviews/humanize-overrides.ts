// Pure helper that assembles a finding's humanized parts into the Markdown body
// posted to the MR. Kept framework-free so it can be unit-tested in isolation
// (see __tests__/humanize-overrides.spec.ts).
//
// The structure mirrors the server's formatFinding (packages/server .../publish.go)
// so humanized comments read consistently with generated ones — just in the
// user's voice. The frontend still owns the finding, so we can restore the
// dimension/severity header tag that a raw text override would otherwise drop.
import type { Finding, FindingHumanized } from '@shared/api/types'
import { dimensionLabel } from '@modules/reviews/format'

// Sentinel meaning "Original" — post the generated text (no humanized override).
// Any value >= 0 is an index into a card's humanize tabs.
export const ORIGINAL = -1

export function buildFindingBody(finding: Finding, humanized: FindingHumanized): string {
  let body = `**[${dimensionLabel[finding.dimension]} · ${finding.severity.toUpperCase()}]** ${humanized.issue}\n\n`
  if (humanized.why) body += `**Why:** ${humanized.why}\n\n`
  if (humanized.fix) body += `**Suggested fix:** ${humanized.fix}\n`
  if (finding.blocking) body += `\n_Blocking._`
  return body
}
