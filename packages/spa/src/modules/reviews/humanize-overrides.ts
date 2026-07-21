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

// buildFindingMarkdown assembles the WHOLE finding as standalone Markdown for the
// copy-to-clipboard action: the dimension/severity header, the location (file:line
// — which buildFindingBody omits because inline MR comments already sit on the
// file), the issue, why and fix. `parts` carries the active tab's text (Original
// or a humanize run), so a copy respects the tab the user is viewing.
export function buildFindingMarkdown(
  finding: Finding,
  parts: Pick<FindingHumanized, 'issue' | 'why' | 'fix'>,
): string {
  let md = `**[${dimensionLabel[finding.dimension]} · ${finding.severity.toUpperCase()}]** ${parts.issue}\n`
  if (finding.file) md += `\n\`${finding.file}${finding.line > 0 ? `:${finding.line}` : ''}\`\n`
  if (parts.why) md += `\n**Why:** ${parts.why}\n`
  if (parts.fix) md += `\n**Suggested fix:** ${parts.fix}\n`
  if (finding.blocking) md += `\n_Blocking._\n`
  return md
}
