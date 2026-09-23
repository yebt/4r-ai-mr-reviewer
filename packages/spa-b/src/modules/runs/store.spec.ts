import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { PiniaColada } from '@pinia/colada'
import { defineComponent } from 'vue'
import * as runsApi from './api'
import { runsQueryKey, shouldPoll, useRunsStore } from './store'
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
 * under a real Pinia + Pinia Colada app context — required because
 * `useQuery`/`useMutation`/`useQueryCache` need an active injection context.
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
    global: { plugins: [createPinia(), PiniaColada] },
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

describe('runsQueryKey (pure)', () => {
  it('builds the ["routines", { archived }] key', () => {
    expect(runsQueryKey(false)).toEqual(['routines', { archived: false }])
    expect(runsQueryKey(true)).toEqual(['routines', { archived: true }])
  })
})

describe('useRunsStore (@pinia/colada)', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('the routines query maps the mocked listRecentRoutines() result into store.runs', async () => {
    const runs = [makeRun()]
    mockedListRecentRoutines.mockResolvedValueOnce(runs)

    const { store } = mountStore()
    await flushPromises()

    expect(mockedListRecentRoutines).toHaveBeenCalledWith(30, false)
    expect(store.runs).toEqual(runs)
    expect(store.isLoading).toBe(false)
    expect(store.runsState.status).toBe('success')
  })

  it('toggling archived re-queries with archived=true', async () => {
    mockedListRecentRoutines.mockResolvedValueOnce([])
    const { store } = mountStore()
    await flushPromises()

    mockedListRecentRoutines.mockResolvedValueOnce([makeRun({ archived: true })])
    store.archived = true
    await flushPromises()

    expect(mockedListRecentRoutines).toHaveBeenCalledWith(30, true)
    expect(store.runs.every((run) => run.archived)).toBe(true)
  })

  it('archiveRun invalidates the routines query on settle and toasts on success', async () => {
    mockedListRecentRoutines.mockResolvedValueOnce([makeRun({ id: 'run1', archived: false })])
    const { store } = mountStore()
    await flushPromises()

    mockedArchiveRoutine.mockResolvedValueOnce(undefined)
    mockedListRecentRoutines.mockResolvedValueOnce([makeRun({ id: 'run1', archived: true })])

    await store.archiveRun('run1')
    await flushPromises()

    expect(mockedArchiveRoutine).toHaveBeenCalledWith('run1')
    expect(mockToastSuccess).toHaveBeenCalledWith('Run archived')
    // Settle re-fetched the query, reflecting the now-archived run.
    expect(mockedListRecentRoutines).toHaveBeenCalledTimes(2)
  })

  it('cancelRun invalidates the routines query on settle and toasts an error on failure', async () => {
    mockedListRecentRoutines.mockResolvedValueOnce([makeRun({ id: 'run1', status: 'running' })])
    const { store } = mountStore()
    await flushPromises()

    mockedCancelRoutine.mockRejectedValueOnce(new Error('Cancel failed'))
    mockedListRecentRoutines.mockResolvedValueOnce([makeRun({ id: 'run1', status: 'running' })])

    await store.cancelRun('run1').catch(() => undefined)
    await flushPromises()

    expect(mockedCancelRoutine).toHaveBeenCalledWith('run1')
    expect(mockToastError).toHaveBeenCalledWith('Cancel failed')
    // Settle re-fetched the query even though the mutation failed.
    expect(mockedListRecentRoutines).toHaveBeenCalledTimes(2)
  })
})
