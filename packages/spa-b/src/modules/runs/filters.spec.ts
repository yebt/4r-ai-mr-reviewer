import { describe, expect, it } from 'vitest'
import { ALL_REPOS_VALUE, ALL_STATUSES_VALUE, RUN_STATUS_FILTER_OPTIONS, filterRuns, repoFilterOptions } from './filters'
import type { RoutineRun } from './types'

function makeRun(overrides: Partial<RoutineRun> = {}): RoutineRun {
  return {
    id: 'r1',
    kind: 'release',
    repoId: 'repo1',
    mrIid: 42,
    status: 'running',
    steps: [],
    state: {},
    lastError: '',
    createdAt: '2026-01-01T00:00:00.000Z',
    updatedAt: '2026-01-01T00:00:00.000Z',
    archived: false,
    repoName: 'repo-one',
    ...overrides,
  }
}

describe('RUN_STATUS_FILTER_OPTIONS', () => {
  it('starts with "All statuses" followed by every RoutineRunStatus', () => {
    expect(RUN_STATUS_FILTER_OPTIONS[0]).toEqual({ label: 'All statuses', value: ALL_STATUSES_VALUE })
    expect(RUN_STATUS_FILTER_OPTIONS.map((o) => o.value).slice(1)).toEqual([
      'pending',
      'running',
      'blocked',
      'awaiting_confirmation',
      'done',
      'cancelled',
    ])
  })
})

describe('repoFilterOptions', () => {
  it('prefers the repos store list (sorted by name), prefixed with "All repositories"', () => {
    const runs = [makeRun({ repoId: 'repo1', repoName: 'repo-one' })]
    const repos = [
      { id: 'repo2', name: 'zeta' },
      { id: 'repo1', name: 'alpha' },
    ]
    expect(repoFilterOptions(runs, repos)).toEqual([
      { label: 'All repositories', value: ALL_REPOS_VALUE },
      { label: 'alpha', value: 'repo1' },
      { label: 'zeta', value: 'repo2' },
    ])
  })

  it('falls back to the distinct repos present in the run list when repos is empty', () => {
    const runs = [
      makeRun({ repoId: 'repo1', repoName: 'repo-one' }),
      makeRun({ repoId: 'repo2', repoName: 'repo-two' }),
      makeRun({ repoId: 'repo1', repoName: 'repo-one' }),
    ]
    expect(repoFilterOptions(runs, [])).toEqual([
      { label: 'All repositories', value: ALL_REPOS_VALUE },
      { label: 'repo-one', value: 'repo1' },
      { label: 'repo-two', value: 'repo2' },
    ])
  })
})

describe('filterRuns', () => {
  const runs = [
    makeRun({ id: 'r1', repoId: 'repo1', status: 'running' }),
    makeRun({ id: 'r2', repoId: 'repo1', status: 'done' }),
    makeRun({ id: 'r3', repoId: 'repo2', status: 'running' }),
  ]

  it('returns every run when both filters are "all"', () => {
    expect(filterRuns(runs, { repoId: ALL_REPOS_VALUE, status: ALL_STATUSES_VALUE })).toEqual(runs)
  })

  it('filters by repoId', () => {
    expect(filterRuns(runs, { repoId: 'repo1', status: ALL_STATUSES_VALUE }).map((r) => r.id)).toEqual(['r1', 'r2'])
  })

  it('filters by status', () => {
    expect(filterRuns(runs, { repoId: ALL_REPOS_VALUE, status: 'running' }).map((r) => r.id)).toEqual(['r1', 'r3'])
  })

  it('combines both filters', () => {
    expect(filterRuns(runs, { repoId: 'repo1', status: 'running' }).map((r) => r.id)).toEqual(['r1'])
  })
})
