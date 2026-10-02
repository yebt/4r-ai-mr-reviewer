import { describe, expect, it } from 'vitest'
import { attentionRuns, recentRuns, runStats } from './dashboard'
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
    ...overrides,
  }
}

describe('runStats', () => {
  it('counts total/active/attention across one run of each status', () => {
    const runs = [
      makeRun({ id: 'pending', status: 'pending' }),
      makeRun({ id: 'running', status: 'running' }),
      makeRun({ id: 'blocked', status: 'blocked' }),
      makeRun({ id: 'awaiting', status: 'awaiting_confirmation' }),
      makeRun({ id: 'done', status: 'done' }),
      makeRun({ id: 'cancelled', status: 'cancelled' }),
    ]

    expect(runStats(runs)).toEqual({
      // every status counts toward total, including the terminal done/cancelled
      total: 6,
      // active = isRunActive: pending, running, blocked, awaiting_confirmation
      active: 4,
      // attention = blocked | awaiting_confirmation only
      attention: 2,
    })
  })

  it('returns all zeros for an empty list', () => {
    expect(runStats([])).toEqual({ total: 0, active: 0, attention: 0 })
  })
})

describe('attentionRuns', () => {
  it('returns only blocked and awaiting_confirmation runs', () => {
    const runs = [
      makeRun({ id: 'a', status: 'blocked' }),
      makeRun({ id: 'b', status: 'running' }),
      makeRun({ id: 'c', status: 'awaiting_confirmation' }),
      makeRun({ id: 'd', status: 'done' }),
    ]

    expect(attentionRuns(runs).map((r) => r.id)).toEqual(['a', 'c'])
  })

  it('returns an empty array when nothing needs a human', () => {
    const runs = [makeRun({ status: 'running' }), makeRun({ status: 'done' })]
    expect(attentionRuns(runs)).toEqual([])
  })
})

describe('recentRuns', () => {
  const runs = [
    makeRun({ id: 'r1' }),
    makeRun({ id: 'r2' }),
    makeRun({ id: 'r3' }),
    makeRun({ id: 'r4' }),
  ]

  it('slices the first n runs without re-sorting', () => {
    expect(recentRuns(runs, 2).map((r) => r.id)).toEqual(['r1', 'r2'])
  })

  it('returns every run when n exceeds the list length', () => {
    expect(recentRuns(runs, 10).map((r) => r.id)).toEqual(['r1', 'r2', 'r3', 'r4'])
  })

  it('returns an empty array for an empty list', () => {
    expect(recentRuns([], 6)).toEqual([])
  })
})
