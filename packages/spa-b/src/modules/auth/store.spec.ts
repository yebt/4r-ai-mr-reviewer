import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { request } from '@shared/api/client'
import { useAuthStore } from './store'

vi.mock('./api', () => ({
  authStatus: vi.fn(),
  login: vi.fn(),
  logout: vi.fn(),
}))

import { authStatus, login, logout } from './api'

describe('useAuthStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('login success sets authenticated=true and enabled=true', async () => {
    vi.mocked(login).mockResolvedValue({ authenticated: true })
    const store = useAuthStore()

    await store.login('secret')

    expect(store.authenticated).toBe(true)
    expect(store.enabled).toBe(true)
  })

  it('fetchStatus network failure fails open: enabled=false, ready=true', async () => {
    vi.mocked(authStatus).mockRejectedValue(new Error('network down'))
    const store = useAuthStore()

    await store.fetchStatus()

    expect(store.enabled).toBe(false)
    expect(store.ready).toBe(true)
  })

  it('logout sets authenticated=false', async () => {
    vi.mocked(logout).mockResolvedValue({ authenticated: false })
    const store = useAuthStore()
    store.authenticated = true

    await store.logout()

    expect(store.authenticated).toBe(false)
  })

  it('a global 401 (non-/auth/ path) flips authenticated=false', async () => {
    const store = useAuthStore()
    store.authenticated = true

    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: false,
        status: 401,
        statusText: 'Unauthorized',
        text: async () => '{}',
      }),
    )

    await expect(request('GET', '/projects')).rejects.toThrow()

    expect(store.authenticated).toBe(false)
  })
})
