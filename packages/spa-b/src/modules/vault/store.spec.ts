import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { PiniaColada } from '@pinia/colada'
import { defineComponent } from 'vue'
import { ApiError } from '@shared/api/client'
import * as vaultApi from './api'
import {
  fetchVaultStatusOrUnavailable,
  isVaultUnavailable,
  resolveVaultErrorMessage,
  useVaultStore,
} from './store'
import type { ChangeVaultPasswordResult, VaultStatus } from './types'

vi.mock('./api')

const mockedFetchVaultStatus = vi.mocked(vaultApi.fetchVaultStatus)
const mockedChangeVaultPassword = vi.mocked(vaultApi.changeVaultPassword)

function makeStatus(overrides: Partial<VaultStatus> = {}): VaultStatus {
  return { initialized: true, passwordProtected: false, ...overrides }
}

/**
 * Mounts a throwaway component that just instantiates `useVaultStore()`
 * under a real Pinia + Pinia Colada app context — required because
 * `useQuery`/`useMutation`/`useQueryCache` need an active injection context,
 * exactly like any other Vue composable.
 */
function mountStore() {
  let store!: ReturnType<typeof useVaultStore>
  const Harness = defineComponent({
    setup() {
      store = useVaultStore()
      return () => null
    },
  })
  const wrapper = mount(Harness, {
    global: { plugins: [createPinia(), PiniaColada] },
  })
  return { wrapper, store }
}

describe('pure helpers', () => {
  it('isVaultUnavailable narrows to the { unavailable: true } sentinel only', () => {
    expect(isVaultUnavailable({ unavailable: true })).toBe(true)
    expect(isVaultUnavailable(makeStatus())).toBe(false)
    expect(isVaultUnavailable(undefined)).toBe(false)
  })

  it('fetchVaultStatusOrUnavailable resolves the real status on success', async () => {
    mockedFetchVaultStatus.mockResolvedValueOnce(makeStatus({ passwordProtected: true }))
    await expect(fetchVaultStatusOrUnavailable()).resolves.toEqual({
      initialized: true,
      passwordProtected: true,
    })
  })

  it('fetchVaultStatusOrUnavailable turns a 501 ApiError into the unavailable sentinel', async () => {
    mockedFetchVaultStatus.mockRejectedValueOnce(new ApiError(501, 'not implemented'))
    await expect(fetchVaultStatusOrUnavailable()).resolves.toEqual({ unavailable: true })
  })

  it('fetchVaultStatusOrUnavailable rethrows any other error (not caught as unavailable)', async () => {
    mockedFetchVaultStatus.mockRejectedValueOnce(new ApiError(500, 'boom'))
    await expect(fetchVaultStatusOrUnavailable()).rejects.toThrow('boom')
  })

  it('resolveVaultErrorMessage maps documented status codes', () => {
    expect(resolveVaultErrorMessage(new ApiError(401, 'x'))).toBe('Current password is incorrect')
    expect(resolveVaultErrorMessage(new ApiError(409, 'x'))).toBe('Vault is not initialized')
    expect(resolveVaultErrorMessage(new ApiError(501, 'x'))).toBe("Vault management isn't available on this instance.")
  })

  it('resolveVaultErrorMessage falls back to the Error message or a default', () => {
    expect(resolveVaultErrorMessage(new Error('boom'))).toBe('boom')
    expect(resolveVaultErrorMessage('not an error')).toBe('Failed to change the master key')
  })
})

describe('useVaultStore (@pinia/colada)', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('status/unavailable reflect a normal successful fetch', async () => {
    mockedFetchVaultStatus.mockResolvedValueOnce(makeStatus({ passwordProtected: true }))

    const { store } = mountStore()
    await flushPromises()

    expect(store.unavailable).toBe(false)
    expect(store.status).toEqual({ initialized: true, passwordProtected: true })
    expect(store.statusState.status).toBe('success')
  })

  it('a 501 response surfaces as store.unavailable = true, store.status = null, and no query error', async () => {
    mockedFetchVaultStatus.mockRejectedValueOnce(new ApiError(501, 'not implemented'))

    const { store } = mountStore()
    await flushPromises()

    expect(store.unavailable).toBe(true)
    expect(store.status).toBeNull()
    expect(store.statusState.status).toBe('success')
    expect(store.error).toBeNull()
  })

  it('changePassword awaits the mutation and refetches status on success', async () => {
    mockedFetchVaultStatus.mockResolvedValueOnce(makeStatus({ passwordProtected: false }))
    const { store } = mountStore()
    await flushPromises()

    const result: ChangeVaultPasswordResult = { passwordProtected: true, warning: 'Update AIR_PASSWORD before restart.' }
    mockedChangeVaultPassword.mockResolvedValueOnce(result)
    mockedFetchVaultStatus.mockResolvedValueOnce(makeStatus({ passwordProtected: true }))

    await expect(store.changePassword({ oldPassword: '', newPassword: 'new-pass' })).resolves.toEqual(result)
    await flushPromises()

    expect(mockedFetchVaultStatus.mock.calls.length).toBeGreaterThanOrEqual(2)
    expect(store.status).toEqual({ initialized: true, passwordProtected: true })
  })

  it('changePassword rethrows on failure without mutating status', async () => {
    mockedFetchVaultStatus.mockResolvedValueOnce(makeStatus({ passwordProtected: true }))
    const { store } = mountStore()
    await flushPromises()

    mockedChangeVaultPassword.mockRejectedValueOnce(new ApiError(401, 'wrong password'))

    await expect(store.changePassword({ oldPassword: 'bad', newPassword: 'new-pass' })).rejects.toThrow(
      'wrong password',
    )
    expect(store.status).toEqual({ initialized: true, passwordProtected: true })
  })
})
