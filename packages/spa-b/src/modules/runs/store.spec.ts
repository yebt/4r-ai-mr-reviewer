import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { defineComponent } from 'vue'
import * as runsApi from './api'
import { shouldPoll, useRunsStore } from './store'
import type { RoutineRun } from './types'

vi.mock('./api')

// `vi.mock` factories are hoisted above regular top-level statements, so the
// spies they close over must be created via `vi.hoisted` — a plain `const`
// declared below would still be in its temporal dead zone when the factory
// first runs. Mirrors `modules/repos/store.spec.ts`.
const { mockToastSuccess, mockToastError } = vi.hoisted(() => ({
  mockToastSuccess: vi.fn(),
  mockToastError: vi.fn(),
}))

vi.mock('@shared/composables/useToast', () => ({
  useToast: () => ({
    success: mockToastSuccess,
    error: mockToastError,
    info: vi.fn(),
  }),
}))

// The polling interval itself is exercised as a pure function (`shouldPoll`,
// see below) rather than through real timers — a real `setInterval` left
// running past a test (the store is never unmounted here, matching
// `modules/repos/store.spec.ts`) would call the mocked API again with an
// exhausted `mockResolvedValueOnce` queue and make later tests flaky.
vi.mock('@vueuse/core', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@vueuse/core')>()
  return {
    ...actual,
    useIntervalFn: () => ({
      pause: vi.fn(),
      resume: vi.fn(),
      isActive: { value: false },
    }),
  }
})

const mockedListRecentRoutines = vi.mocked(runsApi.listRecentRoutines)
const mockedArchiveRoutine = vi.mocked(runsApi.archiveRoutine)
const mockedCancelRoutine = vi.mocked(runsApi.cancelRoutine)

function makeRun(overrides: Partial<RoutineRun> = {}): RoutineRun {
  return {
    id: 'run1',
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
    repoName: 'my-repo',
    ...overrides,
  }
}

/**
 * Mounts a throwaway component that just instantiates `useRunsStore()`
 * under a real Pinia app context — required for `defineStore`'s injection
 * context. Unlike before, the store no longer uses `@pinia/colada` (the
 * list is backed by `useInfiniteList` instead), so no `PiniaColada` plugin
 * is needed here.
 */
function mountStore() {
  let store!: ReturnType<typeof useRunsStore>
  const Harness = defineComponent({
    setup() {
      store = useRunsStore()
      return () => null
    },
  })
  const wrapper = mount(Harness, {
    global: { plugins: [createPinia()] },
  })
  return { wrapper, store }
}

describe('shouldPoll (pure)', () => {
  it('is true when at least one run is active (pending/running/blocked/awaiting_confirmation)', () => {
    expect(shouldPoll([makeRun({ status: 'done' }), makeRun({ status: 'running' })])).toBe(true)
    expect(shouldPoll([makeRun({ status: 'blocked' })])).toBe(true)
  })

  it('is false when every run is terminal, and for an empty list', () => {
    expect(shouldPoll([makeRun({ status: 'done' }), makeRun({ status: 'cancelled' })])).toBe(false)
    expect(shouldPoll([])).toBe(false)
  })
})

describe('useRunsStore (useInfiniteList-backed)', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('loads the first page on mount and maps it into store.runs', async () => {
    const runs = [makeRun()]
    mockedListRecentRoutines.mockResolvedValueOnce({ items: runs, nextCursor: null })

    const { store } = mountStore()
    await flushPromises()

    expect(mockedListRecentRoutines).toHaveBeenCalledWith(null, false)
    expect(store.runs).toEqual(runs)
    expect(store.isLoading).toBe(false)
    expect(store.error).toBeNull()
    expect(store.hasMore).toBe(false)
  })

  it('hasMore reflects a non-null nextCursor, and loadMore appends the next page', async () => {
    mockedListRecentRoutines.mockResolvedValueOnce({
      items: [makeRun({ id: 'run1' })],
      nextCursor: 'cursor-1',
    })
    const { store } = mountStore()
    await flushPromises()
    expect(store.hasMore).toBe(true)

    mockedListRecentRoutines.mockResolvedValueOnce({
      items: [makeRun({ id: 'run2' })],
      nextCursor: null,
    })
    await store.loadMore()
    await flushPromises()

    expect(mockedListRecentRoutines).toHaveBeenLastCalledWith('cursor-1', false)
    expect(store.runs.map((run) => run.id)).toEqual(['run1', 'run2'])
    expect(store.hasMore).toBe(false)
  })

  it('toggling archived resets and re-queries with archived=true', async () => {
    mockedListRecentRoutines.mockResolvedValueOnce({ items: [], nextCursor: null })
    const { store } = mountStore()
    await flushPromises()

    mockedListRecentRoutines.mockResolvedValueOnce({ items: [makeRun({ archived: true })], nextCursor: null })
    store.archived = true
    await flushPromises()

    expect(mockedListRecentRoutines).toHaveBeenCalledWith(null, true)
    expect(store.runs.every((run) => run.archived)).toBe(true)
  })

  it('archiveRun removes the run from the accumulated list locally and toasts on success', async () => {
    mockedListRecentRoutines.mockResolvedValueOnce({
      items: [makeRun({ id: 'run1', archived: false })],
      nextCursor: null,
    })
    const { store } = mountStore()
    await flushPromises()

    mockedArchiveRoutine.mockResolvedValueOnce(undefined)

    await store.archiveRun('run1')
    await flushPromises()

    expect(mockedArchiveRoutine).toHaveBeenCalledWith('run1')
    expect(mockToastSuccess).toHaveBeenCalledWith('Run archived')
    expect(store.runs).toEqual([])
    // No colada invalidate/refetch anymore — membership change is applied locally.
    expect(mockedListRecentRoutines).toHaveBeenCalledTimes(1)
  })

  it('cancelRun toasts an error on failure and does not refresh the list', async () => {
    mockedListRecentRoutines.mockResolvedValueOnce({
      items: [makeRun({ id: 'run1', status: 'running' })],
      nextCursor: null,
    })
    const { store } = mountStore()
    await flushPromises()

    mockedCancelRoutine.mockRejectedValueOnce(new Error('Cancel failed'))

    await store.cancelRun('run1').catch(() => undefined)
    await flushPromises()

    expect(mockedCancelRoutine).toHaveBeenCalledWith('run1')
    expect(mockToastError).toHaveBeenCalledWith('Cancel failed')
    expect(mockedListRecentRoutines).toHaveBeenCalledTimes(1)
  })

  it('cancelRun refreshes the first page (merging the updated status in place) on success', async () => {
    mockedListRecentRoutines.mockResolvedValueOnce({
      items: [makeRun({ id: 'run1', status: 'running' })],
      nextCursor: null,
    })
    const { store } = mountStore()
    await flushPromises()

    mockedCancelRoutine.mockResolvedValueOnce(undefined)
    mockedListRecentRoutines.mockResolvedValueOnce({
      items: [makeRun({ id: 'run1', status: 'cancelled' })],
      nextCursor: null,
    })

    await store.cancelRun('run1')
    await flushPromises()

    expect(mockToastSuccess).toHaveBeenCalledWith('Run cancelled')
    expect(store.runs).toEqual([makeRun({ id: 'run1', status: 'cancelled' })])
    expect(mockedListRecentRoutines).toHaveBeenCalledTimes(2)
  })
})
