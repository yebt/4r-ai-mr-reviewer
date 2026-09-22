import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { PiniaColada } from '@pinia/colada'
import { defineComponent } from 'vue'
import * as providersApi from './api'
import {
  makeOptimisticProvider,
  resolveErrorMessage,
  toProviderPatch,
  useProvidersStore,
  withDefaultFlippedTo,
  withOptimisticCreate,
  withPatchedProvider,
  withoutProvider,
} from './store'
import type { Provider } from './types'

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

const mockedListProviders = vi.mocked(providersApi.listProviders)
const mockedCreateProvider = vi.mocked(providersApi.createProvider)
const mockedUpdateProvider = vi.mocked(providersApi.updateProvider)
const mockedDeleteProvider = vi.mocked(providersApi.deleteProvider)
const mockedSetDefaultProvider = vi.mocked(providersApi.setDefaultProvider)
const mockedFetchOpenRouterModels = vi.mocked(providersApi.fetchOpenRouterModels)

function makeProvider(overrides: Partial<Provider> = {}): Provider {
  return {
    id: 'p1',
    name: 'My OpenAI',
    kind: 'openai-compat',
    baseUrl: 'https://api.openai.com/v1',
    model: 'gpt-4o',
    isDefault: false,
    temperature: null,
    models: [],
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
 * Mounts a throwaway component that just instantiates `useProvidersStore()`
 * under a real Pinia + Pinia Colada app context — required because
 * `useQuery`/`useMutation`/`useQueryCache` need an active injection context,
 * exactly like any other Vue composable (see the Pinia Colada testing
 * cookbook: mount with `global.plugins: [createPinia(), PiniaColada]`).
 */
function mountStore() {
  let store!: ReturnType<typeof useProvidersStore>
  const Harness = defineComponent({
    setup() {
      store = useProvidersStore()
      return () => null
    },
  })
  const wrapper = mount(Harness, {
    global: { plugins: [createPinia(), PiniaColada] },
  })
  return { wrapper, store }
}

describe('pure optimistic-patch helpers', () => {
  it('withOptimisticCreate appends a temp row and unflips other defaults when it is the new default', () => {
    const existing = [makeProvider({ id: 'p1', isDefault: true })]
    const optimistic = makeProvider({ id: 'temp-1', isDefault: true })

    expect(withOptimisticCreate(existing, optimistic)).toEqual([
      { ...existing[0]!, isDefault: false },
      optimistic,
    ])
  })

  it('withOptimisticCreate leaves other rows untouched when the new row is not default', () => {
    const existing = [makeProvider({ id: 'p1', isDefault: true })]
    const optimistic = makeProvider({ id: 'temp-1', isDefault: false })

    expect(withOptimisticCreate(existing, optimistic)).toEqual([...existing, optimistic])
  })

  it('withoutProvider drops the matching row only', () => {
    const existing = [makeProvider({ id: 'p1' }), makeProvider({ id: 'p2' })]
    expect(withoutProvider(existing, 'p1')).toEqual([existing[1]])
  })

  it('withPatchedProvider merges a partial patch onto the matching row only', () => {
    const existing = [makeProvider({ id: 'p1', name: 'Old' }), makeProvider({ id: 'p2', name: 'Other' })]
    const patched = withPatchedProvider(existing, 'p1', { name: 'New' })
    expect(patched[0]).toEqual({ ...existing[0], name: 'New' })
    expect(patched[1]).toEqual(existing[1])
  })

  it('withDefaultFlippedTo flips isDefault to exactly one id', () => {
    const existing = [
      makeProvider({ id: 'p1', isDefault: true }),
      makeProvider({ id: 'p2', isDefault: false }),
    ]
    expect(withDefaultFlippedTo(existing, 'p2')).toEqual([
      { ...existing[0]!, isDefault: false },
      { ...existing[1]!, isDefault: true },
    ])
  })

  it('toProviderPatch strips the write-only apiKey field', () => {
    const patch = toProviderPatch({
      name: 'N',
      kind: 'anthropic',
      baseUrl: 'https://a',
      model: 'm',
      apiKey: 'sk-should-not-leak',
      temperature: 0.5,
      models: ['m'],
    })
    expect(patch).not.toHaveProperty('apiKey')
    expect(patch).toEqual({ name: 'N', kind: 'anthropic', baseUrl: 'https://a', model: 'm', temperature: 0.5, models: ['m'] })
  })

  it('makeOptimisticProvider builds a temp-id row carrying the create payload', () => {
    const optimistic = makeOptimisticProvider({
      name: 'N',
      kind: 'openai-compat',
      baseUrl: 'https://a',
      model: 'm',
      apiKey: 'sk',
      makeDefault: true,
      temperature: null,
      models: [],
    })
    expect(optimistic.id).toMatch(/^temp-/)
    expect(optimistic.isDefault).toBe(true)
    expect(optimistic.name).toBe('N')
  })

  it('resolveErrorMessage prefers the Error message and falls back otherwise', () => {
    expect(resolveErrorMessage(new Error('boom'), 'fallback')).toBe('boom')
    expect(resolveErrorMessage('not an error', 'fallback')).toBe('fallback')
  })
})

describe('useProvidersStore (@pinia/colada)', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('the providers query maps the mocked listProviders() result into store.providers', async () => {
    const providers = [makeProvider()]
    mockedListProviders.mockResolvedValueOnce(providers)

    const { store } = mountStore()
    await flushPromises()

    expect(mockedListProviders).toHaveBeenCalledTimes(1)
    expect(store.providers).toEqual(providers)
    expect(store.isLoading).toBe(false)
    expect(store.providersState.status).toBe('success')
  })

  it('setDefaultProvider flips isDefault optimistically before the request resolves, then invalidates', async () => {
    mockedListProviders.mockResolvedValueOnce([
      makeProvider({ id: 'p1', isDefault: true }),
      makeProvider({ id: 'p2', isDefault: false }),
    ])
    const { store } = mountStore()
    await flushPromises()

    const { promise, resolve } = deferred<void>()
    mockedSetDefaultProvider.mockReturnValueOnce(promise)
    mockedListProviders.mockResolvedValueOnce([
      makeProvider({ id: 'p1', isDefault: false }),
      makeProvider({ id: 'p2', isDefault: true }),
    ])

    const call = store.setDefaultProvider('p2')
    await Promise.resolve()
    await Promise.resolve()

    // Optimistic: flipped locally before the request settles.
    expect(store.providers.find((p) => p.id === 'p1')?.isDefault).toBe(false)
    expect(store.providers.find((p) => p.id === 'p2')?.isDefault).toBe(true)

    resolve()
    await call
    await flushPromises()

    expect(mockToastSuccess).toHaveBeenCalledWith('Set as default')
    // Invalidation on settle triggers a refetch of the ['providers'] query.
    expect(mockedListProviders.mock.calls.length).toBeGreaterThanOrEqual(2)
  })

  it('removeProvider optimistically drops the row and rolls back + toasts on error', async () => {
    mockedListProviders.mockResolvedValueOnce([makeProvider({ id: 'p1' }), makeProvider({ id: 'p2' })])
    const { store } = mountStore()
    await flushPromises()

    const { promise, reject } = deferred<void>()
    mockedDeleteProvider.mockReturnValueOnce(promise)
    mockedListProviders.mockResolvedValueOnce([makeProvider({ id: 'p1' }), makeProvider({ id: 'p2' })])

    const call = store.removeProvider('p1').catch(() => undefined)
    await Promise.resolve()
    await Promise.resolve()

    expect(store.providers.map((p) => p.id)).toEqual(['p2'])

    reject(new Error('Delete failed'))
    await call
    await flushPromises()

    // Rolled back to the pre-mutation snapshot.
    expect(store.providers.map((p) => p.id).sort()).toEqual(['p1', 'p2'])
    expect(mockToastError).toHaveBeenCalledWith('Delete failed')
  })

  it('createProvider appends an optimistic temp row, then reconciles with the server id on success', async () => {
    mockedListProviders.mockResolvedValueOnce([])
    const { store } = mountStore()
    await flushPromises()

    const created = makeProvider({ id: 'server-1' })
    const { promise, resolve } = deferred<Provider>()
    mockedCreateProvider.mockReturnValueOnce(promise)
    mockedListProviders.mockResolvedValueOnce([created])

    const call = store.createProvider({
      name: created.name,
      kind: created.kind,
      baseUrl: created.baseUrl,
      model: created.model,
      apiKey: '',
      makeDefault: false,
      temperature: null,
      models: [],
    })
    await Promise.resolve()
    await Promise.resolve()

    expect(store.providers).toHaveLength(1)
    expect(store.providers[0]!.id).toMatch(/^temp-/)

    resolve(created)
    await call
    await flushPromises()

    expect(store.providers.some((p) => p.id === 'server-1')).toBe(true)
    expect(mockToastSuccess).toHaveBeenCalledWith('Provider added')
  })

  it('updateProvider optimistically patches fields before the request resolves', async () => {
    mockedListProviders.mockResolvedValueOnce([makeProvider({ id: 'p1', name: 'Old name' })])
    const { store } = mountStore()
    await flushPromises()

    const updated = makeProvider({ id: 'p1', name: 'New name' })
    const { promise, resolve } = deferred<Provider>()
    mockedUpdateProvider.mockReturnValueOnce(promise)
    mockedListProviders.mockResolvedValueOnce([updated])

    const call = store.updateProvider('p1', {
      name: 'New name',
      kind: 'openai-compat',
      baseUrl: 'https://api.openai.com/v1',
      model: 'gpt-4o',
      apiKey: '',
      temperature: null,
      models: [],
    })
    await Promise.resolve()
    await Promise.resolve()

    expect(store.providers[0]!.name).toBe('New name')

    resolve(updated)
    await call
    await flushPromises()

    expect(mockToastSuccess).toHaveBeenCalledWith('Provider saved')
    expect(mockToastError).not.toHaveBeenCalled()
  })

  it('testConnection delegates straight to the API and never throws for a rejecting provider', async () => {
    mockedListProviders.mockResolvedValueOnce([])
    const { store } = mountStore()
    await flushPromises()

    vi.mocked(providersApi.testProvider).mockResolvedValueOnce({ ok: false, error: 'Invalid API key' })
    const result = await store.testConnection({
      kind: 'openai-compat',
      baseUrl: 'https://api.openai.com/v1',
      model: 'gpt-4o',
      apiKey: 'bad-key',
    })

    expect(result).toEqual({ ok: false, error: 'Invalid API key' })
  })

  it('fetchOpenRouterModels is lazy: not fetched on mount, only on demand', async () => {
    mockedListProviders.mockResolvedValueOnce([])
    const { store } = mountStore()
    await flushPromises()

    expect(mockedFetchOpenRouterModels).not.toHaveBeenCalled()

    mockedFetchOpenRouterModels.mockResolvedValueOnce([{ id: 'm1', name: 'Model 1', contextLength: 8192 }])
    await store.fetchOpenRouterModels()
    await flushPromises()

    expect(mockedFetchOpenRouterModels).toHaveBeenCalledTimes(1)
    expect(store.openRouterModels).toEqual([{ id: 'm1', name: 'Model 1', contextLength: 8192 }])
  })
})
