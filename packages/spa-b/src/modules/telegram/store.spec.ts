import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { PiniaColada } from '@pinia/colada'
import { defineComponent } from 'vue'
import * as telegramApi from './api'
import {
  makeOptimisticTarget,
  resolveErrorMessage,
  sortDefaultFirst,
  toTargetPatch,
  useTelegramStore,
  withDefaultFlippedTo,
  withOptimisticCreate,
  withPatchedTarget,
  withoutTarget,
} from './store'
import type { TelegramTarget } from './types'

vi.mock('./api')

// `vi.mock` factories are hoisted above regular top-level statements, so the
// spies they close over must be created via `vi.hoisted` — a plain `const`
// declared below would still be in its temporal dead zone when the factory
// first runs.
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

const mockedListTelegramTargets = vi.mocked(telegramApi.listTelegramTargets)
const mockedCreateTelegramTarget = vi.mocked(telegramApi.createTelegramTarget)
const mockedUpdateTelegramTarget = vi.mocked(telegramApi.updateTelegramTarget)
const mockedDeleteTelegramTarget = vi.mocked(telegramApi.deleteTelegramTarget)
const mockedSetDefaultTelegramTarget = vi.mocked(telegramApi.setDefaultTelegramTarget)

function makeTarget(overrides: Partial<TelegramTarget> = {}): TelegramTarget {
  return {
    id: 't1',
    name: 'Internal channel',
    chatId: '-1004396084945',
    threadId: '3',
    isDefault: false,
    isBot: false,
    createdAt: '2026-01-01T00:00:00.000Z',
    ...overrides,
  }
}

/** A promise plus its resolve/reject, so a test can assert the optimistic
 * cache state before the underlying request settles. */
function deferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (reason: unknown) => void
  const promise = new Promise<T>((res, rej) => {
    resolve = res
    reject = rej
  })
  return { promise, resolve, reject }
}

/**
 * Mounts a throwaway component that just instantiates `useTelegramStore()`
 * under a real Pinia + Pinia Colada app context — required because
 * `useQuery`/`useMutation`/`useQueryCache` need an active injection context,
 * exactly like any other Vue composable.
 */
function mountStore() {
  let store!: ReturnType<typeof useTelegramStore>
  const Harness = defineComponent({
    setup() {
      store = useTelegramStore()
      return () => null
    },
  })
  const wrapper = mount(Harness, {
    global: { plugins: [createPinia(), PiniaColada] },
  })
  return { wrapper, store }
}

describe('pure optimistic-patch helpers', () => {
  it('withOptimisticCreate appends a new row', () => {
    const existing = [makeTarget({ id: 't1' })]
    const optimistic = makeTarget({ id: 'temp-1' })

    expect(withOptimisticCreate(existing, optimistic)).toEqual([...existing, optimistic])
  })

  it('withoutTarget drops the matching row only', () => {
    const existing = [makeTarget({ id: 't1' }), makeTarget({ id: 't2' })]
    expect(withoutTarget(existing, 't1')).toEqual([existing[1]])
  })

  it('withPatchedTarget merges a partial patch onto the matching row only', () => {
    const existing = [makeTarget({ id: 't1', name: 'Old' }), makeTarget({ id: 't2', name: 'Other' })]
    const patched = withPatchedTarget(existing, 't1', { name: 'New' })
    expect(patched[0]).toEqual({ ...existing[0], name: 'New' })
    expect(patched[1]).toEqual(existing[1])
  })

  it('withDefaultFlippedTo flips isDefault to exactly one id', () => {
    const existing = [
      makeTarget({ id: 't1', isDefault: true }),
      makeTarget({ id: 't2', isDefault: false }),
    ]
    expect(withDefaultFlippedTo(existing, 't2')).toEqual([
      { ...existing[0]!, isDefault: false },
      { ...existing[1]!, isDefault: true },
    ])
  })

  it('sortDefaultFirst moves the default row to the top and keeps the rest stable', () => {
    const existing = [
      makeTarget({ id: 't1', name: 'First' }),
      makeTarget({ id: 't2', name: 'Default', isDefault: true }),
      makeTarget({ id: 't3', name: 'Third' }),
    ]
    expect(sortDefaultFirst(existing).map((target) => target.id)).toEqual(['t2', 't1', 't3'])
  })

  it('sortDefaultFirst is a no-op ordering when nothing is default', () => {
    const existing = [makeTarget({ id: 't1' }), makeTarget({ id: 't2' })]
    expect(sortDefaultFirst(existing)).toEqual(existing)
  })

  it('toTargetPatch strips the write-only botToken field', () => {
    const patch = toTargetPatch({
      name: 'N',
      chatId: 'c1',
      threadId: '1',
      botToken: 'should-not-leak',
      isBot: true,
    })
    expect(patch).not.toHaveProperty('botToken')
    expect(patch).toEqual({ name: 'N', chatId: 'c1', threadId: '1', isBot: true })
  })

  it('makeOptimisticTarget builds a temp-id row carrying the create payload', () => {
    const optimistic = makeOptimisticTarget({
      name: 'N',
      chatId: 'c1',
      threadId: '1',
      botToken: 'tok',
      isBot: true,
    })
    expect(optimistic.id).toMatch(/^temp-/)
    expect(optimistic.isDefault).toBe(false)
    expect(optimistic.name).toBe('N')
    expect(optimistic.isBot).toBe(true)
  })

  it('resolveErrorMessage prefers the Error message and falls back otherwise', () => {
    expect(resolveErrorMessage(new Error('boom'), 'fallback')).toBe('boom')
    expect(resolveErrorMessage('not an error', 'fallback')).toBe('fallback')
  })
})

describe('useTelegramStore (@pinia/colada)', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('the telegram query maps the mocked listTelegramTargets() result into store.targets', async () => {
    const targets = [makeTarget()]
    mockedListTelegramTargets.mockResolvedValueOnce(targets)

    const { store } = mountStore()
    await flushPromises()

    expect(mockedListTelegramTargets).toHaveBeenCalledTimes(1)
    expect(store.targets).toEqual(targets)
    expect(store.isLoading).toBe(false)
    expect(store.targetsState.status).toBe('success')
  })

  it('setDefaultTarget flips isDefault optimistically before the request resolves, then invalidates', async () => {
    mockedListTelegramTargets.mockResolvedValueOnce([
      makeTarget({ id: 't1', isDefault: true }),
      makeTarget({ id: 't2', isDefault: false }),
    ])
    const { store } = mountStore()
    await flushPromises()

    const { promise, resolve } = deferred<void>()
    mockedSetDefaultTelegramTarget.mockReturnValueOnce(promise)
    mockedListTelegramTargets.mockResolvedValueOnce([
      makeTarget({ id: 't1', isDefault: false }),
      makeTarget({ id: 't2', isDefault: true }),
    ])

    const call = store.setDefaultTarget('t2')
    await Promise.resolve()
    await Promise.resolve()

    // Optimistic: flipped locally before the request settles.
    expect(store.targets.find((t) => t.id === 't1')?.isDefault).toBe(false)
    expect(store.targets.find((t) => t.id === 't2')?.isDefault).toBe(true)

    resolve()
    await call
    await flushPromises()

    expect(mockToastSuccess).toHaveBeenCalledWith('Set as default')
    expect(mockedListTelegramTargets.mock.calls.length).toBeGreaterThanOrEqual(2)
  })

  it('removeTarget optimistically drops the row and rolls back + toasts on error', async () => {
    mockedListTelegramTargets.mockResolvedValueOnce([makeTarget({ id: 't1' }), makeTarget({ id: 't2' })])
    const { store } = mountStore()
    await flushPromises()

    const { promise, reject } = deferred<void>()
    mockedDeleteTelegramTarget.mockReturnValueOnce(promise)
    mockedListTelegramTargets.mockResolvedValueOnce([makeTarget({ id: 't1' }), makeTarget({ id: 't2' })])

    const call = store.removeTarget('t1').catch(() => undefined)
    await Promise.resolve()
    await Promise.resolve()

    expect(store.targets.map((t) => t.id)).toEqual(['t2'])

    reject(new Error('Delete failed'))
    await call
    await flushPromises()

    // Rolled back to the pre-mutation snapshot.
    expect(store.targets.map((t) => t.id).sort()).toEqual(['t1', 't2'])
    expect(mockToastError).toHaveBeenCalledWith('Delete failed')
  })

  it('createTarget appends an optimistic temp row, then reconciles with the server id on success', async () => {
    mockedListTelegramTargets.mockResolvedValueOnce([])
    const { store } = mountStore()
    await flushPromises()

    const created = makeTarget({ id: 'server-1' })
    const { promise, resolve } = deferred<TelegramTarget>()
    mockedCreateTelegramTarget.mockReturnValueOnce(promise)
    mockedListTelegramTargets.mockResolvedValueOnce([created])

    const call = store.createTarget({
      name: created.name,
      chatId: created.chatId,
      threadId: created.threadId,
      botToken: '',
      isBot: false,
    })
    await Promise.resolve()
    await Promise.resolve()

    expect(store.targets).toHaveLength(1)
    expect(store.targets[0]!.id).toMatch(/^temp-/)

    resolve(created)
    await call
    await flushPromises()

    expect(store.targets.some((t) => t.id === 'server-1')).toBe(true)
    expect(mockToastSuccess).toHaveBeenCalledWith('Telegram target added')
  })

  it('updateTarget optimistically patches fields before the request resolves', async () => {
    mockedListTelegramTargets.mockResolvedValueOnce([makeTarget({ id: 't1', name: 'Old name' })])
    const { store } = mountStore()
    await flushPromises()

    const updated = makeTarget({ id: 't1', name: 'New name' })
    const { promise, resolve } = deferred<TelegramTarget>()
    mockedUpdateTelegramTarget.mockReturnValueOnce(promise)
    mockedListTelegramTargets.mockResolvedValueOnce([updated])

    const call = store.updateTarget('t1', {
      name: 'New name',
      chatId: '-1004396084945',
      threadId: '3',
      botToken: '',
      isBot: false,
    })
    await Promise.resolve()
    await Promise.resolve()

    expect(store.targets[0]!.name).toBe('New name')

    resolve(updated)
    await call
    await flushPromises()

    expect(mockToastSuccess).toHaveBeenCalledWith('Telegram target saved')
    expect(mockToastError).not.toHaveBeenCalled()
  })

  it('testTarget delegates straight to the API and never throws for a rejecting target', async () => {
    mockedListTelegramTargets.mockResolvedValueOnce([])
    const { store } = mountStore()
    await flushPromises()

    vi.mocked(telegramApi.testTelegramTarget).mockResolvedValueOnce({ ok: false, error: 'Chat not found' })
    const result = await store.testTarget('t1')

    expect(result).toEqual({ ok: false, error: 'Chat not found' })
  })
})
