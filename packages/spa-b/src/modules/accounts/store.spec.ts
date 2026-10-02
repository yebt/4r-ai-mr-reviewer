import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { PiniaColada } from '@pinia/colada'
import { defineComponent } from 'vue'
import * as accountsApi from './api'
import {
  makeOptimisticAccount,
  resolveErrorMessage,
  toAccountPatch,
  useAccountsStore,
  withOptimisticCreate,
  withPatchedAccount,
  withoutAccount,
} from './store'
import type { Account } from './types'

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

const mockedListAccounts = vi.mocked(accountsApi.listAccounts)
const mockedCreateAccount = vi.mocked(accountsApi.createAccount)
const mockedUpdateAccount = vi.mocked(accountsApi.updateAccount)
const mockedDeleteAccount = vi.mocked(accountsApi.deleteAccount)

function makeAccount(overrides: Partial<Account> = {}): Account {
  return {
    id: 'a1',
    name: 'Work',
    baseUrl: 'https://gitlab.com',
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
 * Mounts a throwaway component that just instantiates `useAccountsStore()`
 * under a real Pinia + Pinia Colada app context — required because
 * `useQuery`/`useMutation`/`useQueryCache` need an active injection context,
 * exactly like any other Vue composable (see the Pinia Colada testing
 * cookbook: mount with `global.plugins: [createPinia(), PiniaColada]`).
 */
function mountStore() {
  let store!: ReturnType<typeof useAccountsStore>
  const Harness = defineComponent({
    setup() {
      store = useAccountsStore()
      return () => null
    },
  })
  const wrapper = mount(Harness, {
    global: { plugins: [createPinia(), PiniaColada] },
  })
  return { wrapper, store }
}

describe('pure optimistic-patch helpers', () => {
  it('withOptimisticCreate appends the new row', () => {
    const existing = [makeAccount({ id: 'a1' })]
    const optimistic = makeAccount({ id: 'temp-1' })

    expect(withOptimisticCreate(existing, optimistic)).toEqual([...existing, optimistic])
  })

  it('withoutAccount drops the matching row only', () => {
    const existing = [makeAccount({ id: 'a1' }), makeAccount({ id: 'a2' })]
    expect(withoutAccount(existing, 'a1')).toEqual([existing[1]])
  })

  it('withPatchedAccount merges a partial patch onto the matching row only', () => {
    const existing = [makeAccount({ id: 'a1', name: 'Old' }), makeAccount({ id: 'a2', name: 'Other' })]
    const patched = withPatchedAccount(existing, 'a1', { name: 'New' })
    expect(patched[0]).toEqual({ ...existing[0], name: 'New' })
    expect(patched[1]).toEqual(existing[1])
  })

  it('toAccountPatch strips the write-only token field', () => {
    const patch = toAccountPatch({
      name: 'N',
      baseUrl: 'https://gitlab.example.com',
      token: 'glpat-should-not-leak',
    })
    expect(patch).not.toHaveProperty('token')
    expect(patch).toEqual({ name: 'N', baseUrl: 'https://gitlab.example.com' })
  })

  it('makeOptimisticAccount builds a temp-id row carrying the create payload', () => {
    const optimistic = makeOptimisticAccount({
      name: 'N',
      baseUrl: 'https://gitlab.example.com',
      token: 'glpat-x',
    })
    expect(optimistic.id).toMatch(/^temp-/)
    expect(optimistic.name).toBe('N')
    expect(optimistic.baseUrl).toBe('https://gitlab.example.com')
  })

  it('resolveErrorMessage prefers the Error message and falls back otherwise', () => {
    expect(resolveErrorMessage(new Error('boom'), 'fallback')).toBe('boom')
    expect(resolveErrorMessage('not an error', 'fallback')).toBe('fallback')
  })
})

describe('useAccountsStore (@pinia/colada)', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('the accounts query maps the mocked listAccounts() result into store.accounts', async () => {
    const accounts = [makeAccount()]
    mockedListAccounts.mockResolvedValueOnce(accounts)

    const { store } = mountStore()
    await flushPromises()

    expect(mockedListAccounts).toHaveBeenCalledTimes(1)
    expect(store.accounts).toEqual(accounts)
    expect(store.isLoading).toBe(false)
    expect(store.accountsState.status).toBe('success')
  })

  it('createAccount appends an optimistic temp row, then reconciles with the server id on success', async () => {
    mockedListAccounts.mockResolvedValueOnce([])
    const { store } = mountStore()
    await flushPromises()

    const created = makeAccount({ id: 'server-1' })
    const { promise, resolve } = deferred<Account>()
    mockedCreateAccount.mockReturnValueOnce(promise)
    mockedListAccounts.mockResolvedValueOnce([created])

    const call = store.createAccount({
      name: created.name,
      baseUrl: created.baseUrl,
      token: 'glpat-x',
    })
    await Promise.resolve()
    await Promise.resolve()

    expect(store.accounts).toHaveLength(1)
    expect(store.accounts[0]!.id).toMatch(/^temp-/)

    resolve(created)
    await call
    await flushPromises()

    expect(store.accounts.some((a) => a.id === 'server-1')).toBe(true)
    expect(mockToastSuccess).toHaveBeenCalledWith('Account added')
  })

  it('removeAccount optimistically drops the row and rolls back + toasts on error', async () => {
    mockedListAccounts.mockResolvedValueOnce([makeAccount({ id: 'a1' }), makeAccount({ id: 'a2' })])
    const { store } = mountStore()
    await flushPromises()

    const { promise, reject } = deferred<void>()
    mockedDeleteAccount.mockReturnValueOnce(promise)
    mockedListAccounts.mockResolvedValueOnce([makeAccount({ id: 'a1' }), makeAccount({ id: 'a2' })])

    const call = store.removeAccount('a1').catch(() => undefined)
    await Promise.resolve()
    await Promise.resolve()

    expect(store.accounts.map((a) => a.id)).toEqual(['a2'])

    reject(new Error('Delete failed'))
    await call
    await flushPromises()

    // Rolled back to the pre-mutation snapshot.
    expect(store.accounts.map((a) => a.id).sort()).toEqual(['a1', 'a2'])
    expect(mockToastError).toHaveBeenCalledWith('Delete failed')
  })

  it('updateAccount optimistically patches fields before the request resolves', async () => {
    mockedListAccounts.mockResolvedValueOnce([makeAccount({ id: 'a1', name: 'Old name' })])
    const { store } = mountStore()
    await flushPromises()

    const updated = makeAccount({ id: 'a1', name: 'New name' })
    const { promise, resolve } = deferred<Account>()
    mockedUpdateAccount.mockReturnValueOnce(promise)
    mockedListAccounts.mockResolvedValueOnce([updated])

    const call = store.updateAccount('a1', {
      name: 'New name',
      baseUrl: 'https://gitlab.com',
      token: '',
    })
    await Promise.resolve()
    await Promise.resolve()

    expect(store.accounts[0]!.name).toBe('New name')

    resolve(updated)
    await call
    await flushPromises()

    expect(mockToastSuccess).toHaveBeenCalledWith('Account saved')
    expect(mockToastError).not.toHaveBeenCalled()
  })
})
