import { describe, expect, it } from 'vitest'
import { FINDING_SEVERITY_BADGE, groupFindingsByDimension } from './findings'
import type { Finding } from './types'

function makeFinding(overrides: Partial<Finding> = {}): Finding {
  return {
    index: 0,
    dimension: 'risk',
    severity: 'medium',
    file: 'src/foo.ts',
    line: 10,
    issue: 'issue',
    why: 'why',
    fix: 'fix',
    blocking: false,
    published: true,
    ...overrides,
  }
}

describe('groupFindingsByDimension', () => {
  it('buckets findings under their dimension, in risk/readability/reliability/resilience order', () => {
    const findings = [
      makeFinding({ index: 0, dimension: 'resilience' }),
      makeFinding({ index: 1, dimension: 'risk' }),
      makeFinding({ index: 2, dimension: 'risk' }),
      makeFinding({ index: 3, dimension: 'readability' }),
    ]

    const grouped = groupFindingsByDimension(findings)

    expect(Object.keys(grouped)).toEqual(['risk', 'readability', 'reliability', 'resilience'])
    expect(grouped.risk.map((f) => f.index)).toEqual([1, 2])
    expect(grouped.readability.map((f) => f.index)).toEqual([3])
    expect(grouped.reliability).toEqual([])
    expect(grouped.resilience.map((f) => f.index)).toEqual([0])
  })

  it('returns every dimension key, empty, for an empty findings list', () => {
    expect(groupFindingsByDimension([])).toEqual({
      risk: [],
      readability: [],
      reliability: [],
      resilience: [],
    })
  })
})

describe('FINDING_SEVERITY_BADGE', () => {
  it.each([
    ['high', 'danger'],
    ['medium', 'warning'],
    ['low', 'neutral'],
  ] as const)('maps severity "%s" to Badge status "%s"', (severity, badgeStatus) => {
    expect(FINDING_SEVERITY_BADGE[severity]).toBe(badgeStatus)
  })
})
