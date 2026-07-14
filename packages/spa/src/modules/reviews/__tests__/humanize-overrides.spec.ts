import { describe, expect, it } from 'vitest'
import type { Finding, HumanizeVariant } from '@shared/api/types'
import { ORIGINAL, buildOverrides, variantFindingText } from '@modules/reviews/humanize-overrides'

const finding = (index: number): Finding => ({
  index,
  dimension: 'risk',
  severity: 'low',
  file: 'a.ts',
  line: index,
  issue: 'issue',
  why: 'why',
  fix: 'fix',
  blocking: false,
  published: false,
})

const variants: HumanizeVariant[] = [
  { summary: 'summary v1', findings: [{ index: 0, text: 'f0 v1' }, { index: 1, text: 'f1 v1' }] },
  { summary: 'summary v2', findings: [{ index: 0, text: 'f0 v2' }, { index: 1, text: 'f1 v2' }] },
]

const findings = [finding(0), finding(1)]

describe('variantFindingText', () => {
  it('returns the variant text matched by finding index', () => {
    expect(variantFindingText(variants, 1, 0)).toBe('f0 v2')
  })

  it('returns empty string for an unknown variant or finding index', () => {
    expect(variantFindingText(variants, 9, 0)).toBe('')
    expect(variantFindingText(variants, 0, 99)).toBe('')
  })
})

describe('buildOverrides', () => {
  it('returns an empty object when everything is Original', () => {
    expect(
      buildOverrides({ summary: ORIGINAL, findings: {} }, variants, findings),
    ).toEqual({})
  })

  it('includes only findings whose source is a variant', () => {
    const out = buildOverrides(
      { summary: ORIGINAL, findings: { 0: 1, 1: ORIGINAL } },
      variants,
      findings,
    )
    expect(out).toEqual({ findingOverrides: [{ index: 0, text: 'f0 v2' }] })
  })

  it('mixes summary and per-finding variant sources', () => {
    const out = buildOverrides(
      { summary: 0, findings: { 0: 0, 1: 1 } },
      variants,
      findings,
    )
    expect(out).toEqual({
      summaryOverride: 'summary v1',
      findingOverrides: [
        { index: 0, text: 'f0 v1' },
        { index: 1, text: 'f1 v2' },
      ],
    })
  })

  it('omits overrides that resolve to empty text', () => {
    const withBlank: HumanizeVariant[] = [{ summary: '', findings: [{ index: 0, text: '' }] }]
    const out = buildOverrides(
      { summary: 0, findings: { 0: 0 } },
      withBlank,
      [finding(0)],
    )
    expect(out).toEqual({})
  })

  it('ignores selections pointing at a missing variant', () => {
    const out = buildOverrides(
      { summary: 5, findings: { 0: 5 } },
      variants,
      [finding(0)],
    )
    expect(out).toEqual({})
  })
})
