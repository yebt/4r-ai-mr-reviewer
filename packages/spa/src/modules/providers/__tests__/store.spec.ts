import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('@shared/api/client', () => ({
  errorMessage: (e: unknown) => (e instanceof Error ? e.message : String(e)),
  api: {
    listProviders: vi.fn(),
    createProvider: vi.fn(),
    setDefaultProvider: vi.fn(),
    deleteProvider: vi.fn(),
  },
}))

import { api } from '@shared/api/client'
import { useProvidersStore } from '@modules/providers/store'

const mocked = api as unknown as {
  listProviders: ReturnType<typeof vi.fn>
  createProvider: ReturnType<typeof vi.fn>
  setDefaultProvider: ReturnType<typeof vi.fn>
  deleteProvider: ReturnType<typeof vi.fn>
}

const provider = (id: string, isDefault = false) => ({
  id,
  name: `p-${id}`,
  kind: 'openai-compat' as const,
  baseUrl: '',
  model: 'm',
  isDefault,
  temperature: null,
  models: [] as string[],
  createdAt: '',
})

describe('providers store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('add creates then refetches to reflect default changes', async () => {
    mocked.createProvider.mockResolvedValue(provider('2', true))
    mocked.listProviders.mockResolvedValue([provider('1', false), provider('2', true)])
    const store = useProvidersStore()
    await store.add({ name: 'p-2', kind: 'openai-compat', baseUrl: '', model: 'm', apiKey: 'k', makeDefault: true, temperature: null, models: [] })
    expect(mocked.listProviders).toHaveBeenCalledOnce()
    expect(store.items).toHaveLength(2)
  })

  it('setDefault flips isDefault to a single provider', async () => {
    mocked.setDefaultProvider.mockResolvedValue(undefined)
    const store = useProvidersStore()
    store.items = [provider('1', true), provider('2', false)]
    await store.setDefault('2')
    expect(store.items.find((p) => p.id === '2')?.isDefault).toBe(true)
    expect(store.items.find((p) => p.id === '1')?.isDefault).toBe(false)
  })

  it('remove drops the provider', async () => {
    mocked.deleteProvider.mockResolvedValue(undefined)
    const store = useProvidersStore()
    store.items = [provider('1'), provider('2')]
    await store.remove('1')
    expect(store.items.map((p) => p.id)).toEqual(['2'])
  })
})
