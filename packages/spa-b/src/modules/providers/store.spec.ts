import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { request } from '@shared/api/client'
import { useProvidersStore } from './store'
import type { Provider } from './types'

vi.mock('@shared/api/client', () => ({
  request: vi.fn<typeof request>(),
}))

const mockedRequest = vi.mocked(request)

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

describe('useProvidersStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('fetchProviders populates the list from GET /providers', async () => {
    const providers = [makeProvider()]
    mockedRequest.mockResolvedValueOnce(providers)
    const store = useProvidersStore()

    await store.fetchProviders()

    expect(mockedRequest).toHaveBeenCalledWith('GET', '/providers')
    expect(store.providers).toEqual(providers)
    expect(store.loading).toBe(false)
    expect(store.error).toBeNull()
  })

  it('createProvider posts the full body with a numeric temperature coerced correctly', async () => {
    const created = makeProvider({ id: 'p2', temperature: 0.7 })
    mockedRequest.mockResolvedValueOnce(created)
    const store = useProvidersStore()

    await store.createProvider({
      name: 'My OpenAI',
      kind: 'openai-compat',
      baseUrl: 'https://api.openai.com/v1',
      model: 'gpt-4o',
      apiKey: 'sk-test',
      makeDefault: false,
      temperature: 0.7,
      models: ['gpt-4o'],
    })

    expect(mockedRequest).toHaveBeenCalledWith('POST', '/providers', {
      name: 'My OpenAI',
      kind: 'openai-compat',
      baseUrl: 'https://api.openai.com/v1',
      model: 'gpt-4o',
      apiKey: 'sk-test',
      makeDefault: false,
      temperature: 0.7,
      models: ['gpt-4o'],
    })
    expect(store.providers).toEqual([created])
  })

  it('createProvider forwards a blank temperature as null, not NaN or a string', async () => {
    const created = makeProvider({ id: 'p3', temperature: null })
    mockedRequest.mockResolvedValueOnce(created)
    const store = useProvidersStore()

    await store.createProvider({
      name: 'My Anthropic',
      kind: 'anthropic',
      baseUrl: 'https://api.anthropic.com',
      model: 'claude-opus',
      apiKey: 'sk-ant',
      makeDefault: false,
      temperature: null,
      models: [],
    })

    const [, , body] = mockedRequest.mock.calls[0]!
    expect((body as { temperature: unknown }).temperature).toBeNull()
  })

  it('updateProvider PATCHes without makeDefault in the body', async () => {
    const updated = makeProvider()
    mockedRequest.mockResolvedValueOnce(updated)
    const store = useProvidersStore()

    await store.updateProvider('p1', {
      name: 'My OpenAI',
      kind: 'openai-compat',
      baseUrl: 'https://api.openai.com/v1',
      model: 'gpt-4o',
      apiKey: '',
      temperature: null,
      models: [],
    })

    expect(mockedRequest).toHaveBeenCalledWith('PATCH', '/providers/p1', {
      name: 'My OpenAI',
      kind: 'openai-compat',
      baseUrl: 'https://api.openai.com/v1',
      model: 'gpt-4o',
      apiKey: '',
      temperature: null,
      models: [],
    })
    const [, , body] = mockedRequest.mock.calls[0]!
    expect(body).not.toHaveProperty('makeDefault')
  })

  it('setDefaultProvider hits POST /providers/{id}/default and flips local isDefault', async () => {
    mockedRequest.mockResolvedValueOnce([
      makeProvider({ id: 'p1', isDefault: true }),
      makeProvider({ id: 'p2' }),
    ])
    const store = useProvidersStore()
    await store.fetchProviders()

    mockedRequest.mockResolvedValueOnce(undefined)
    await store.setDefaultProvider('p2')

    expect(mockedRequest).toHaveBeenCalledWith('POST', '/providers/p2/default')
    expect(store.providers.find((p) => p.id === 'p1')?.isDefault).toBe(false)
    expect(store.providers.find((p) => p.id === 'p2')?.isDefault).toBe(true)
  })

  it('testConnection resolves {ok:false, error} without throwing for a rejecting provider', async () => {
    mockedRequest.mockResolvedValueOnce({ ok: false, error: 'Invalid API key' })
    const store = useProvidersStore()

    const result = await store.testConnection({
      kind: 'openai-compat',
      baseUrl: 'https://api.openai.com/v1',
      model: 'gpt-4o',
      apiKey: 'bad-key',
    })

    expect(mockedRequest).toHaveBeenCalledWith('POST', '/providers/test', {
      kind: 'openai-compat',
      baseUrl: 'https://api.openai.com/v1',
      model: 'gpt-4o',
      apiKey: 'bad-key',
    })
    expect(result).toEqual({ ok: false, error: 'Invalid API key' })
  })

  it('removeProvider deletes and removes it from local state', async () => {
    mockedRequest.mockResolvedValueOnce([makeProvider({ id: 'p1' })])
    const store = useProvidersStore()
    await store.fetchProviders()

    mockedRequest.mockResolvedValueOnce(undefined)
    await store.removeProvider('p1')

    expect(mockedRequest).toHaveBeenCalledWith('DELETE', '/providers/p1')
    expect(store.providers).toEqual([])
  })
})
