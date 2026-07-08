import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('@shared/api/client', () => ({
  errorMessage: (e: unknown) => (e instanceof Error ? e.message : String(e)),
  api: {
    listAccounts: vi.fn(),
    createAccount: vi.fn(),
    deleteAccount: vi.fn(),
  },
}))

import { api } from '@shared/api/client'
import { useAccountsStore } from '@modules/accounts/store'

const mocked = api as unknown as {
  listAccounts: ReturnType<typeof vi.fn>
  createAccount: ReturnType<typeof vi.fn>
  deleteAccount: ReturnType<typeof vi.fn>
}

const account = (id: string) => ({ id, name: `acc-${id}`, baseUrl: 'https://gitlab.com', createdAt: '' })

describe('accounts store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('fetchAll populates items and clears loading', async () => {
    mocked.listAccounts.mockResolvedValue([account('1')])
    const store = useAccountsStore()
    await store.fetchAll()
    expect(store.items).toHaveLength(1)
    expect(store.loading).toBe(false)
    expect(store.error).toBeNull()
  })

  it('fetchAll captures the error message', async () => {
    mocked.listAccounts.mockRejectedValue(new Error('network down'))
    const store = useAccountsStore()
    await store.fetchAll()
    expect(store.error).toBe('network down')
    expect(store.items).toHaveLength(0)
  })

  it('add appends the created account', async () => {
    mocked.createAccount.mockResolvedValue(account('2'))
    const store = useAccountsStore()
    await store.add({ name: 'acc-2', baseUrl: 'u', token: 't' })
    expect(store.items.map((a) => a.id)).toContain('2')
  })

  it('remove drops the account', async () => {
    mocked.deleteAccount.mockResolvedValue(undefined)
    const store = useAccountsStore()
    store.items = [account('1'), account('2')]
    await store.remove('1')
    expect(store.items.map((a) => a.id)).toEqual(['2'])
  })
})
