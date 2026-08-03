import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('@shared/api/client', () => ({
  errorMessage: (e: unknown) => (e instanceof Error ? e.message : String(e)),
  api: {
    listRepoRoutines: vi.fn(),
    listRecentRoutines: vi.fn(),
    createRelease: vi.fn(),
    createMainRelease: vi.fn(),
    getRoutineRun: vi.fn(),
    resumeRoutine: vi.fn(),
    skipRoutine: vi.fn(),
    confirmRoutine: vi.fn(),
    cancelRoutine: vi.fn(),
    deleteRoutine: vi.fn(),
  },
}))

import { api } from '@shared/api/client'
import type { RoutineRun, RoutineRunStatus } from '@shared/api/types'
import { useRoutinesStore } from '@modules/routines/store'

const mocked = api as unknown as {
  listRepoRoutines: ReturnType<typeof vi.fn>
  listRecentRoutines: ReturnType<typeof vi.fn>
  createRelease: ReturnType<typeof vi.fn>
  createMainRelease: ReturnType<typeof vi.fn>
  getRoutineRun: ReturnType<typeof vi.fn>
  resumeRoutine: ReturnType<typeof vi.fn>
  skipRoutine: ReturnType<typeof vi.fn>
  confirmRoutine: ReturnType<typeof vi.fn>
  cancelRoutine: ReturnType<typeof vi.fn>
  deleteRoutine: ReturnType<typeof vi.fn>
}

const run = (id: string, status: RoutineRunStatus = 'pending'): RoutineRun => ({
  id,
  kind: 'release',
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

  it('createRelease prepends the new run and sends the chosen fields', async () => {
    mocked.createRelease.mockResolvedValue(run('new'))
    const store = useRoutinesStore()
    store.runsByRepo = { r1: [run('old')] }
    const created = await store.createRelease('r1', {
      mrIid: 7,
      bump: 'minor',
      removeSourceBranch: false,
      mergeWhenPipelineSucceeds: true,
    })
    expect(created.id).toBe('new')
    expect(mocked.createRelease).toHaveBeenCalledWith('r1', {
      mrIid: 7,
      bump: 'minor',
      removeSourceBranch: false,
      mergeWhenPipelineSucceeds: true,
    })
    expect(store.runsFor('r1').map((r) => r.id)).toEqual(['new', 'old'])
  })

  it('createRelease omits absent optional fields but keeps explicit false toggles', async () => {
    mocked.createRelease.mockResolvedValue(run('x'))
    const store = useRoutinesStore()
    await store.createRelease('r1', { mrIid: 3, mergeWhenPipelineSucceeds: false })
    expect(mocked.createRelease).toHaveBeenCalledWith('r1', {
      mrIid: 3,
      mergeWhenPipelineSucceeds: false,
    })
  })

  it('createRelease propagates api errors (e.g. 400 bad target)', async () => {
    mocked.createRelease.mockRejectedValue(new Error('MR target must be development'))
    const store = useRoutinesStore()
    await expect(store.createRelease('r1', { mrIid: 7 })).rejects.toThrow(
      'MR target must be development',
    )
  })

  it('createMainRelease trims branch overrides and omits empty ones', async () => {
    mocked.createMainRelease.mockResolvedValue(run('m'))
    const store = useRoutinesStore()
    await store.createMainRelease('r1', {
      bump: 'major',
      sourceBranch: '  release/next  ',
      targetBranch: '   ',
      removeSourceBranch: true,
      mergeWhenPipelineSucceeds: false,
    })
    expect(mocked.createMainRelease).toHaveBeenCalledWith('r1', {
      bump: 'major',
      sourceBranch: 'release/next',
      removeSourceBranch: true,
      mergeWhenPipelineSucceeds: false,
    })
  })

  it('refresh updates the run in both caches', async () => {
    const store = useRoutinesStore()
    store.runsByRepo = { r1: [run('a', 'running')] }
    store.runsById = { a: run('a', 'running') }
    mocked.getRoutineRun.mockResolvedValue(run('a', 'awaiting_confirmation'))
    await store.refresh('a')
    expect(mocked.getRoutineRun).toHaveBeenCalledWith('a')
    expect(store.runsFor('r1')[0]?.status).toBe('awaiting_confirmation')
    expect(store.runById('a')?.status).toBe('awaiting_confirmation')
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

  it('skip calls the api and adopts the returned run', async () => {
    const store = useRoutinesStore()
    store.runsByRepo = { r1: [run('a', 'blocked')] }
    store.runsById = { a: run('a', 'blocked') }
    mocked.skipRoutine.mockResolvedValue(run('a', 'running'))
    await store.skip('a')
    expect(mocked.skipRoutine).toHaveBeenCalledWith('a')
    expect(store.runById('a')?.status).toBe('running')
  })

  it('skip propagates a 409 when the step is not skippable', async () => {
    mocked.skipRoutine.mockRejectedValue(new Error('step is not skippable'))
    const store = useRoutinesStore()
    store.runsByRepo = { r1: [run('a', 'blocked')] }
    store.runsById = { a: run('a', 'blocked') }
    await expect(store.skip('a')).rejects.toThrow('step is not skippable')
    expect(store.runById('a')?.status).toBe('blocked')
  })

  it('confirm sends the decision and adopts the returned run', async () => {
    const store = useRoutinesStore()
    store.runsByRepo = { r1: [run('a', 'awaiting_confirmation')] }
    store.runsById = { a: run('a', 'awaiting_confirmation') }
    mocked.confirmRoutine.mockResolvedValue(run('a', 'running'))
    await store.confirm('a', 'merge')
    expect(mocked.confirmRoutine).toHaveBeenCalledWith('a', 'merge')
    expect(store.runById('a')?.status).toBe('running')
  })

  it('listRecent caches runs by id and exposes them in order with the limit', async () => {
    mocked.listRecentRoutines.mockResolvedValue([run('a'), run('b')])
    const store = useRoutinesStore()
    await store.listRecent(20)
    expect(mocked.listRecentRoutines).toHaveBeenCalledWith(20)
    expect(store.recentRuns.map((r) => r.id)).toEqual(['a', 'b'])
    expect(store.runById('a')?.id).toBe('a')
  })

  it('recentRuns stays in sync with a poll refresh', async () => {
    mocked.listRecentRoutines.mockResolvedValue([run('a', 'running')])
    const store = useRoutinesStore()
    await store.listRecent()
    expect(mocked.listRecentRoutines).toHaveBeenCalledWith(undefined)
    mocked.getRoutineRun.mockResolvedValue(run('a', 'done'))
    await store.refresh('a')
    expect(store.recentRuns[0]?.status).toBe('done')
  })

  it('listRecent records the error on failure', async () => {
    mocked.listRecentRoutines.mockRejectedValue(new Error('gitlab down'))
    const store = useRoutinesStore()
    await store.listRecent()
    expect(store.listError).toBe('gitlab down')
    expect(store.recentRuns).toEqual([])
  })

  it('confirm propagates a 409 when the run is not awaiting confirmation', async () => {
    mocked.confirmRoutine.mockRejectedValue(new Error('run is not awaiting confirmation'))
    const store = useRoutinesStore()
    await expect(store.confirm('a', 'wait')).rejects.toThrow('run is not awaiting confirmation')
  })

  it('cancel calls the api and adopts the returned cancelled run', async () => {
    const store = useRoutinesStore()
    store.runsByRepo = { r1: [run('a', 'running')] }
    store.runsById = { a: run('a', 'running') }
    store.recentRunIds = ['a']
    mocked.cancelRoutine.mockResolvedValue(run('a', 'cancelled'))
    await store.cancel('a')
    expect(mocked.cancelRoutine).toHaveBeenCalledWith('a')
    expect(store.runById('a')?.status).toBe('cancelled')
    expect(store.runsFor('r1')[0]?.status).toBe('cancelled')
    expect(store.recentRuns[0]?.status).toBe('cancelled')
  })

  it('cancel propagates a 409 when the run is already terminal', async () => {
    mocked.cancelRoutine.mockRejectedValue(new Error('run is not cancelable'))
    const store = useRoutinesStore()
    store.runsByRepo = { r1: [run('a', 'done')] }
    store.runsById = { a: run('a', 'done') }
    await expect(store.cancel('a')).rejects.toThrow('run is not cancelable')
    expect(store.runById('a')?.status).toBe('done')
  })

  it('remove deletes the run and drops it from every cache', async () => {
    mocked.deleteRoutine.mockResolvedValue(undefined)
    const store = useRoutinesStore()
    store.runsByRepo = { r1: [run('a'), run('b')], r2: [run('a')] }
    store.runsById = { a: run('a'), b: run('b') }
    store.recentRunIds = ['a', 'b']
    await store.remove('a')
    expect(mocked.deleteRoutine).toHaveBeenCalledWith('a')
    expect(store.runById('a')).toBeNull()
    expect(store.runById('b')?.id).toBe('b')
    expect(store.runsFor('r1').map((r) => r.id)).toEqual(['b'])
    expect(store.runsFor('r2')).toEqual([])
    expect(store.recentRunIds).toEqual(['b'])
    expect(store.recentRuns.map((r) => r.id)).toEqual(['b'])
  })

  it('remove propagates a 409 and keeps the caches intact', async () => {
    mocked.deleteRoutine.mockRejectedValue(new Error("can't delete a running action"))
    const store = useRoutinesStore()
    store.runsByRepo = { r1: [run('a', 'running')] }
    store.runsById = { a: run('a', 'running') }
    store.recentRunIds = ['a']
    await expect(store.remove('a')).rejects.toThrow("can't delete a running action")
    expect(store.runById('a')?.id).toBe('a')
    expect(store.runsFor('r1').map((r) => r.id)).toEqual(['a'])
    expect(store.recentRunIds).toEqual(['a'])
  })
})
