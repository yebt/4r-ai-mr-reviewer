import { describe, expect, it } from 'vitest'
import { hasUnpublished, unpublishedFindingIndices } from './publish'
import type { Finding, Review } from './types'

function makeFinding(overrides: Partial<Finding> = {}): Finding {
  return {
    index: 0,
    dimension: 'risk',
    severity: 'high',
    file: 'src/foo.ts',
    line: 10,
    issue: 'issue',
    why: 'why',
    fix: 'fix',
    blocking: false,
    published: false,
    ...overrides,
  }
}

describe('unpublishedFindingIndices', () => {
  it('returns only the indices of unpublished findings, mixed with published ones', () => {
    const findings = [
      makeFinding({ index: 0, published: true }),
      makeFinding({ index: 1, published: false }),
      makeFinding({ index: 2, published: false }),
      makeFinding({ index: 3, published: true }),
    ]

    expect(unpublishedFindingIndices(findings)).toEqual([1, 2])
  })

  it('returns an empty array when every finding is published', () => {
    const findings = [makeFinding({ index: 0, published: true }), makeFinding({ index: 1, published: true })]
    expect(unpublishedFindingIndices(findings)).toEqual([])
  })
})

describe('hasUnpublished', () => {
  it('returns false when every finding and the summary are published', () => {
    const review: Pick<Review, 'findings' | 'summaryPublished'> = {
      findings: [makeFinding({ index: 0, published: true }), makeFinding({ index: 1, published: true })],
      summaryPublished: true,
    }

    expect(hasUnpublished(review)).toBe(false)
  })

  it('returns true when a finding is unpublished even if the summary is published', () => {
    const review: Pick<Review, 'findings' | 'summaryPublished'> = {
      findings: [makeFinding({ index: 0, published: true }), makeFinding({ index: 1, published: false })],
      summaryPublished: true,
    }

    expect(hasUnpublished(review)).toBe(true)
  })

  it('returns true when the summary is unpublished even if every finding is published', () => {
    const review: Pick<Review, 'findings' | 'summaryPublished'> = {
      findings: [makeFinding({ index: 0, published: true })],
      summaryPublished: false,
    }

    expect(hasUnpublished(review)).toBe(true)
  })

  it('returns true for an empty findings list with an unpublished summary', () => {
    const review: Pick<Review, 'findings' | 'summaryPublished'> = {
      findings: [],
      summaryPublished: false,
    }

    expect(hasUnpublished(review)).toBe(true)
  })
})
