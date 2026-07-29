import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('@shared/api/client', () => ({
  errorMessage: (e: unknown) => (e instanceof Error ? e.message : String(e)),
  api: {
    listRepoRoutines: vi.fn(),
    createApproveAndTag: vi.fn(),
    getRoutineRun: vi.fn(),
    resumeRoutine: vi.fn(),
  },
}))

import { api } from '@shared/api/client'
import type { RoutineRun, RoutineRunStatus } from '@shared/api/types'
import { useRoutinesStore } from '@modules/routines/store'

const mocked = api as unknown as {
  listRepoRoutines: ReturnType<typeof vi.fn>
  createApproveAndTag: ReturnType<typeof vi.fn>
  getRoutineRun: ReturnType<typeof vi.fn>
  resumeRoutine: ReturnType<typeof vi.fn>
}

const run = (id: string, status: RoutineRunStatus = 'pending'): RoutineRun => ({
  id,
  kind: 'approve_and_tag',
  repoId: 'r1',
  mrIid: 7,
  status,
  steps: [],
  state: {},
  lastError: '',
  createdAt: '',
  updatedAt: '',
})

describe('routines store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('list caches runs per repo and by id', async () => {
    mocked.listRepoRoutines.mockResolvedValue([run('a'), run('b')])
    const store = useRoutinesStore()
    await store.list('r1')
    expect(mocked.listRepoRoutines).toHaveBeenCalledWith('r1')
    expect(store.runsFor('r1').map((r) => r.id)).toEqual(['a', 'b'])
    expect(store.runById('a')?.id).toBe('a')
    expect(store.runsFor('other')).toEqual([])
  })

  it('list records the error on failure', async () => {
    mocked.listRepoRoutines.mockRejectedValue(new Error('gitlab down'))
    const store = useRoutinesStore()
    await store.list('r1')
    expect(store.listError).toBe('gitlab down')
    expect(store.runsFor('r1')).toEqual([])
  })

  it('create prepends the new run and only sends meaningful fields', async () => {
    mocked.createApproveAndTag.mockResolvedValue(run('new'))
    const store = useRoutinesStore()
    store.runsByRepo = { r1: [run('old')] }
    const created = await store.create('r1', {
      mrIid: 7,
      bump: 'minor',
      comment: '  LGFM  ',
      emojis: ['thumbsup'],
    })
    expect(created.id).toBe('new')
    expect(mocked.createApproveAndTag).toHaveBeenCalledWith('r1', {
      mrIid: 7,
      bump: 'minor',
      comment: 'LGFM',
      emojis: ['thumbsup'],
    })
    expect(store.runsFor('r1').map((r) => r.id)).toEqual(['new', 'old'])
  })

  it('create omits empty optional fields', async () => {
    mocked.createApproveAndTag.mockResolvedValue(run('x'))
    const store = useRoutinesStore()
    await store.create('r1', { mrIid: 3, comment: '   ', emojis: [] })
    expect(mocked.createApproveAndTag).toHaveBeenCalledWith('r1', { mrIid: 3 })
  })

  it('create propagates api errors (e.g. 409 duplicate)', async () => {
    mocked.createApproveAndTag.mockRejectedValue(new Error('a run already exists'))
    const store = useRoutinesStore()
    await expect(store.create('r1', { mrIid: 7 })).rejects.toThrow('a run already exists')
  })

  it('refresh updates the run in both caches', async () => {
    const store = useRoutinesStore()
    store.runsByRepo = { r1: [run('a', 'running')] }
    store.runsById = { a: run('a', 'running') }
    mocked.getRoutineRun.mockResolvedValue(run('a', 'blocked'))
    await store.refresh('a')
    expect(mocked.getRoutineRun).toHaveBeenCalledWith('a')
    expect(store.runsFor('r1')[0]?.status).toBe('blocked')
    expect(store.runById('a')?.status).toBe('blocked')
  })

  it('resume calls the api and adopts the returned run', async () => {
    const store = useRoutinesStore()
    store.runsByRepo = { r1: [run('a', 'blocked')] }
    store.runsById = { a: run('a', 'blocked') }
    mocked.resumeRoutine.mockResolvedValue(run('a', 'running'))
    await store.resume('a')
    expect(mocked.resumeRoutine).toHaveBeenCalledWith('a')
    expect(store.runById('a')?.status).toBe('running')
  })

  it('resume propagates a 409 when the run is not blocked', async () => {
    mocked.resumeRoutine.mockRejectedValue(new Error('run is not blocked'))
    const store = useRoutinesStore()
    await expect(store.resume('a')).rejects.toThrow('run is not blocked')
  })
})
