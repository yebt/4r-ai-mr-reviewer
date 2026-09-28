import { describe, expect, it } from 'vitest'
import { latestReviewByMr } from './mergeRequests'
import type { Review } from '@modules/reviews'

function makeReview(overrides: Partial<Review> = {}): Review {
  return {
    id: 'rv1',
    repoId: 'repo1',
    mrIid: 1,
    contextMode: 'fast',
    status: 'done',
    phase: '',
    archived: false,
    summaryPublished: false,
    summary: '',
    recommendation: 'approve',
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

describe('latestReviewByMr', () => {
  it('keeps only the most recently created review per mrIid', () => {
    const older = makeReview({ id: 'a', mrIid: 5, createdAt: '2026-01-01T00:00:00.000Z', status: 'done' })
    const newer = makeReview({ id: 'b', mrIid: 5, createdAt: '2026-01-02T00:00:00.000Z', status: 'running' })
    const other = makeReview({ id: 'c', mrIid: 6, createdAt: '2026-01-01T00:00:00.000Z' })

    const result = latestReviewByMr([older, newer, other])

    expect(result.get(5)).toBe(newer)
    expect(result.get(6)).toBe(other)
    expect(result.has(7)).toBe(false)
  })

  it('returns an empty map for no reviews', () => {
    expect(latestReviewByMr([]).size).toBe(0)
  })
})
