import { describe, expect, it } from 'vitest'
import { ALL_REPOS_VALUE, ALL_STATUSES_VALUE, REVIEW_STATUS_FILTER_OPTIONS, filterReviews, repoFilterOptions } from './filters'
import type { ReviewWithRepo } from './types'

function makeReview(overrides: Partial<ReviewWithRepo> = {}): ReviewWithRepo {
  return {
    id: 'rev1',
    repoId: 'repo1',
    repoName: 'repo-one',
    mrIid: 42,
    contextMode: 'fast',
    status: 'running',
    phase: 'review',
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

describe('REVIEW_STATUS_FILTER_OPTIONS', () => {
  it('starts with "All statuses" followed by every ReviewStatus', () => {
    expect(REVIEW_STATUS_FILTER_OPTIONS[0]).toEqual({ label: 'All statuses', value: ALL_STATUSES_VALUE })
    expect(REVIEW_STATUS_FILTER_OPTIONS.map((o) => o.value).slice(1)).toEqual([
      'awaiting_approval',
      'pending',
      'running',
      'done',
      'error',
      'cancelled',
    ])
  })
})

describe('repoFilterOptions', () => {
  it('returns the distinct repos present in the list (sorted by name), prefixed with "All repositories"', () => {
    const reviews = [
      makeReview({ id: 'r1', repoId: 'repo2', repoName: 'zeta' }),
      makeReview({ id: 'r2', repoId: 'repo1', repoName: 'alpha' }),
      makeReview({ id: 'r3', repoId: 'repo1', repoName: 'alpha' }),
    ]
    expect(repoFilterOptions(reviews)).toEqual([
      { label: 'All repositories', value: ALL_REPOS_VALUE },
      { label: 'alpha', value: 'repo1' },
      { label: 'zeta', value: 'repo2' },
    ])
  })

  it('returns just the "All repositories" sentinel for an empty list', () => {
    expect(repoFilterOptions([])).toEqual([{ label: 'All repositories', value: ALL_REPOS_VALUE }])
  })
})

describe('filterReviews', () => {
  const reviews = [
    makeReview({ id: 'r1', repoId: 'repo1', status: 'running' }),
    makeReview({ id: 'r2', repoId: 'repo1', status: 'done' }),
    makeReview({ id: 'r3', repoId: 'repo2', status: 'running' }),
  ]

  it('returns every review when both filters use the "all" sentinel', () => {
    expect(filterReviews(reviews, { repoId: ALL_REPOS_VALUE, status: ALL_STATUSES_VALUE })).toEqual(reviews)
  })

  it('filters by repoId', () => {
    expect(filterReviews(reviews, { repoId: 'repo1', status: ALL_STATUSES_VALUE }).map((r) => r.id)).toEqual([
      'r1',
      'r2',
    ])
  })

  it('filters by status', () => {
    expect(filterReviews(reviews, { repoId: ALL_REPOS_VALUE, status: 'running' }).map((r) => r.id)).toEqual([
      'r1',
      'r3',
    ])
  })

  it('combines both filters', () => {
    expect(filterReviews(reviews, { repoId: 'repo1', status: 'running' }).map((r) => r.id)).toEqual(['r1'])
  })
})
