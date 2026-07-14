// Pure helpers that turn per-finding / summary source selections into the
// override payload the publish endpoint accepts. Kept framework-free so it can
// be unit-tested in isolation (see __tests__/humanize-overrides.spec.ts).
import type { Finding, HumanizeFindingText, HumanizeVariant } from '@shared/api/types'

// Sentinel meaning "post the generated text" (no override). Any value >= 0 is an
// index into the variants array.
export const ORIGINAL = -1

export interface OverrideSelections {
  // Selected source for the summary: ORIGINAL or a variant index.
  summary: number
  // Selected source per finding index: ORIGINAL or a variant index.
  findings: Record<number, number>
}

export interface PublishOverrides {
  summaryOverride?: string
  findingOverrides?: HumanizeFindingText[]
}

// Looks up a variant's rewritten text for a given finding index.
export function variantFindingText(
  variants: HumanizeVariant[],
  variantIndex: number,
  findingIndex: number,
): string {
  const variant = variants[variantIndex]
  if (!variant) return ''
  return variant.findings.find((f) => f.index === findingIndex)?.text ?? ''
}

// Builds the { summaryOverride?, findingOverrides? } payload. Sources left on
// ORIGINAL are omitted so those parts publish with the generated text. Empty
// variant text is treated as "no override" to avoid posting blank comments.
export function buildOverrides(
  selections: OverrideSelections,
  variants: HumanizeVariant[],
  findings: Finding[],
): PublishOverrides {
  const out: PublishOverrides = {}

  if (selections.summary !== ORIGINAL) {
    const summary = variants[selections.summary]?.summary ?? ''
    if (summary) out.summaryOverride = summary
  }

  const findingOverrides: HumanizeFindingText[] = []
  for (const f of findings) {
    const source = selections.findings[f.index] ?? ORIGINAL
    if (source === ORIGINAL) continue
    const text = variantFindingText(variants, source, f.index)
    if (text) findingOverrides.push({ index: f.index, text })
  }
  if (findingOverrides.length) out.findingOverrides = findingOverrides

  return out
}
