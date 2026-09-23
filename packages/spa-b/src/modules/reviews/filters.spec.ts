import { describe, expect, it } from 'vitest'
import { filterReviewsByKey } from './filters'
import type { ReviewWithRepo } from './types'

function makeReview(overrides: Partial<ReviewWithRepo> = {}): ReviewWithRepo {
  return {
    id: 'r1',
    repoId: 'repo-1',
    repoName: 'repo-one',
    mrIid: 1,
    contextMode: 'fast',
    status: 'pending',
    phase: 'queued',
    archived: false,
    summaryPublished: false,
    summary: '',
    recommendation: 'comment',
    score: 0,
    inputTokens: 0,
    outputTokens: 0,
    findings: [],
    reasonings: [],
    createdAt: '2026-01-01T00:00:00.000Z',
    updatedAt: '2026-01-01T00:00:00.000Z',
    ...overrides,
  }
}

describe('filterReviewsByKey', () => {
  const reviews = [
    makeReview({ id: 'r1', status: 'awaiting_approval' }),
    makeReview({ id: 'r2', status: 'running' }),
    makeReview({ id: 'r3', status: 'done' }),
    makeReview({ id: 'r4', status: 'error' }),
    makeReview({ id: 'r5', status: 'pending' }),
    makeReview({ id: 'r6', status: 'cancelled' }),
  ]

  it('"all" returns every row unfiltered, including statuses with no dedicated tab', () => {
    expect(filterReviewsByKey(reviews, 'all')).toEqual(reviews)
  })

  it.each([
    ['awaiting_approval', ['r1']],
    ['running', ['r2']],
    ['done', ['r3']],
    ['error', ['r4']],
  ] as const)('"%s" matches only reviews with that status', (key, expectedIds) => {
    expect(filterReviewsByKey(reviews, key).map((r) => r.id)).toEqual(expectedIds)
  })

  it('returns an empty list when no review matches the filter', () => {
    expect(filterReviewsByKey([makeReview({ id: 'r1', status: 'pending' })], 'error')).toEqual([])
  })
})
