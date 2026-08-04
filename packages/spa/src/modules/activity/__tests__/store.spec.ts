import { createPinia, setActivePinia } from 'pinia'
import { nextTick } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'

// The activity store reads the routines/reviews stores, which import the api
// client at module load. Mock it so those stores instantiate; the activity
// tests seed state directly and never hit the network.
vi.mock('@shared/api/client', () => ({
  errorMessage: (e: unknown) => (e instanceof Error ? e.message : String(e)),
  api: {
    listRecentRoutines: vi.fn().mockResolvedValue([]),
    listRepoReviews: vi.fn().mockResolvedValue([]),
  },
}))

import type { Review, RoutineRun, RoutineRunStatus, ReviewStatus } from '@shared/api/types'
import { useRoutinesStore } from '@modules/routines/store'
import { useReviewsStore } from '@modules/reviews/store'
import { useActivityStore } from '@modules/activity/store'

const run = (id: string, status: RoutineRunStatus): RoutineRun => ({
  id,
  kind: 'release',
  repoId: 'r1',
  mrIid: 7,
  status,
  steps: [],
  state: { mrTitle: `Run ${id}` },
  lastError: '',
  createdAt: '',
  updatedAt: '',
})

// createdAt is derived from the id suffix (r1 -> newest) so allReviews' newest-
// first sort yields a deterministic r1, r2, r3, r4 order.
const review = (id: string, status: ReviewStatus, mrIid = 42): Review =>
  ({
    id,
    repoId: 'r1',
    mrIid,
    status,
    createdAt: `2026-01-0${9 - Number(id.slice(1))}`,
  }) as unknown as Review

function seedRuns(...runs: RoutineRun[]) {
  const routines = useRoutinesStore()
  routines.recentRunIds = runs.map((r) => r.id)
  for (const r of runs) routines.runsById[r.id] = r
}

function seedReviews(...reviews: Review[]) {
  const store = useReviewsStore()
  store.reviewsByRepo = { r1: reviews }
}

describe('activity store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('aggregates active runs and non-terminal reviews, excluding terminal ones', () => {
    seedRuns(run('a', 'running'), run('b', 'done'), run('c', 'pending'))
    seedReviews(
      review('r1', 'running'),
      review('r2', 'awaiting_approval'),
      review('r3', 'done'),
      review('r4', 'cancelled'),
    )
    const activity = useActivityStore()

    const ids = activity.activeOps.map((op) => op.id)
    // Active runs a, c (b is done) + non-terminal reviews r1, r2 (r3/r4 terminal).
    expect(ids).toEqual(['a', 'c', 'r1', 'r2'])
    expect(activity.count).toBe(4)

    const run0 = activity.activeOps[0]
    expect(run0).toMatchObject({ kind: 'action', id: 'a', title: 'Run a', to: '/actions/a' })
    const rev = activity.activeOps.find((op) => op.id === 'r1')
    expect(rev).toMatchObject({ kind: 'review', title: 'Review !42', to: '/reviews/r1' })
  })

  it('count is zero when everything is terminal', () => {
    seedRuns(run('a', 'done'), run('b', 'cancelled'))
    seedReviews(review('r1', 'done'), review('r2', 'error'))
    const activity = useActivityStore()
    expect(activity.count).toBe(0)
    expect(activity.activeOps).toEqual([])
  })

  it('dismiss hides the sheet and a new active op re-reveals it', async () => {
    seedRuns(run('a', 'running'))
    const activity = useActivityStore()
    expect(activity.dismissed).toBe(false)

    activity.dismiss()
    expect(activity.dismissed).toBe(true)

    // A new active op appears -> the growth watcher clears the dismissed flag.
    seedRuns(run('a', 'running'), run('b', 'pending'))
    await nextTick()
    expect(activity.count).toBe(2)
    expect(activity.dismissed).toBe(false)
  })
})
