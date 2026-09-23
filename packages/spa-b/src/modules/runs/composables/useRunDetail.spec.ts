import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { PiniaColada } from '@pinia/colada'
import { defineComponent } from 'vue'
import * as runsApi from '../api'
import { runDetailQueryKey, useRunDetail } from './useRunDetail'
import type { RoutineRun } from '../types'

vi.mock('../api')

// `vi.mock` factories are hoisted above regular top-level statements, so the
// spies they close over must be created via `vi.hoisted` — mirrors
// `store.spec.ts`.
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

// Same rationale as `store.spec.ts`: the polling interval is exercised via
// the pure `isRunActive`/`canPoll` wiring, not real timers — a real
// `setInterval` left running past a test would call the mocked API again
// with an exhausted `mockResolvedValueOnce` queue and make later tests flaky.
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

const mockedGetRoutine = vi.mocked(runsApi.getRoutine)
const mockedResumeRoutine = vi.mocked(runsApi.resumeRoutine)
const mockedConfirmRoutine = vi.mocked(runsApi.confirmRoutine)

function makeRun(overrides: Partial<RoutineRun> = {}): RoutineRun {
  return {
    id: 'run1',
    kind: 'release',
    repoId: 'repo1',
    mrIid: 42,
    status: 'blocked',
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
 * Mounts a throwaway component that just instantiates `useRunDetail(id)`
 * under a real Pinia + Pinia Colada app context — required because
 * `useQuery`/`useMutation`/`useQueryCache` need an active injection context.
 */
function mountDetail(id = 'run1') {
  let detail!: ReturnType<typeof useRunDetail>
  const Harness = defineComponent({
    setup() {
      detail = useRunDetail(id)
      return () => null
    },
  })
  const wrapper = mount(Harness, {
    global: { plugins: [createPinia(), PiniaColada] },
  })
  return { wrapper, detail }
}

describe('runDetailQueryKey (pure)', () => {
  it('builds the ["routine", id] key', () => {
    expect(runDetailQueryKey('run1')).toEqual(['routine', 'run1'])
  })
})

describe('useRunDetail (@pinia/colada)', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('the routine query maps the mocked getRoutine() result into detail.run', async () => {
    const run = makeRun()
    mockedGetRoutine.mockResolvedValueOnce(run)

    const { detail } = mountDetail('run1')
    await flushPromises()

    expect(mockedGetRoutine).toHaveBeenCalledWith('run1')
    expect(detail.run.value).toEqual(run)
    expect(detail.state.value.status).toBe('success')
  })

  it('resumeRun invalidates the routine query on settle and toasts on success', async () => {
    mockedGetRoutine.mockResolvedValueOnce(makeRun({ status: 'blocked' }))
    const { detail } = mountDetail('run1')
    await flushPromises()

    mockedResumeRoutine.mockResolvedValueOnce(undefined)
    mockedGetRoutine.mockResolvedValueOnce(makeRun({ status: 'running' }))

    await detail.resumeRun()
    await flushPromises()

    expect(mockedResumeRoutine).toHaveBeenCalledWith('run1')
    expect(mockToastSuccess).toHaveBeenCalledWith('Run resumed')
    // Settle re-fetched the single-run query, reflecting the now-running run.
    expect(mockedGetRoutine).toHaveBeenCalledTimes(2)
    expect(detail.run.value?.status).toBe('running')
  })

  it('confirmRun passes the decision through and toasts an error on failure', async () => {
    mockedGetRoutine.mockResolvedValueOnce(makeRun({ status: 'awaiting_confirmation' }))
    const { detail } = mountDetail('run1')
    await flushPromises()

    mockedConfirmRoutine.mockRejectedValueOnce(new Error('Confirm failed'))
    mockedGetRoutine.mockResolvedValueOnce(makeRun({ status: 'awaiting_confirmation' }))

    await detail.confirmRun('merge').catch(() => undefined)
    await flushPromises()

    expect(mockedConfirmRoutine).toHaveBeenCalledWith('run1', 'merge')
    expect(mockToastError).toHaveBeenCalledWith('Confirm failed')
    // Settle re-fetched even though the mutation failed.
    expect(mockedGetRoutine).toHaveBeenCalledTimes(2)
  })
})
