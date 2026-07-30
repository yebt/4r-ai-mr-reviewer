import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'

// Capture the handler the store registers via onUnauthorized so tests can
// simulate a mid-session 401. Hoisted so the vi.mock factory can reference it.
const hoisted = vi.hoisted(() => ({
  handler: { current: null as null | (() => void) },
}))

vi.mock('@shared/api/client', () => ({
  errorMessage: (e: unknown) => (e instanceof Error ? e.message : String(e)),
  onUnauthorized: (fn: () => void) => {
    hoisted.handler.current = fn
    return () => {}
  },
  api: {
    authStatus: vi.fn(),
    login: vi.fn(),
    logout: vi.fn(),
  },
}))

import { api } from '@shared/api/client'
import { useAuthStore } from '@modules/auth/store'

const mocked = api as unknown as {
  authStatus: ReturnType<typeof vi.fn>
  login: ReturnType<typeof vi.fn>
  logout: ReturnType<typeof vi.fn>
}

describe('auth store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    hoisted.handler.current = null
  })

  it('starts not ready, disabled and unauthenticated', () => {
    const store = useAuthStore()
    expect(store.ready).toBe(false)
    expect(store.enabled).toBe(false)
    expect(store.authenticated).toBe(false)
  })

  it('fetchStatus adopts enabled/authenticated and marks ready', async () => {
    mocked.authStatus.mockResolvedValue({ authEnabled: true, authenticated: true })
    const store = useAuthStore()
    await store.fetchStatus()
    expect(mocked.authStatus).toHaveBeenCalledOnce()
    expect(store.enabled).toBe(true)
    expect(store.authenticated).toBe(true)
    expect(store.ready).toBe(true)
  })

  it('fetchStatus reflects the disabled (default) case as a no-op gate', async () => {
    mocked.authStatus.mockResolvedValue({ authEnabled: false, authenticated: false })
    const store = useAuthStore()
    await store.fetchStatus()
    expect(store.enabled).toBe(false)
    expect(store.authenticated).toBe(false)
    expect(store.ready).toBe(true)
  })

  it('fetchStatus fails open (disabled) and still becomes ready on error', async () => {
    mocked.authStatus.mockRejectedValue(new Error('network down'))
    const store = useAuthStore()
    await store.fetchStatus()
    expect(store.enabled).toBe(false)
    expect(store.authenticated).toBe(false)
    expect(store.ready).toBe(true)
  })

  it('login sets authenticated and enabled on success', async () => {
    mocked.login.mockResolvedValue({ authenticated: true })
    const store = useAuthStore()
    await store.login('secret')
    expect(mocked.login).toHaveBeenCalledWith('secret')
    expect(store.authenticated).toBe(true)
    expect(store.enabled).toBe(true)
  })

  it('login propagates the error and leaves state unauthenticated', async () => {
    mocked.login.mockRejectedValue(new Error('invalid password'))
    const store = useAuthStore()
    await expect(store.login('nope')).rejects.toThrow('invalid password')
    expect(store.authenticated).toBe(false)
  })

  it('logout clears authenticated', async () => {
    mocked.logout.mockResolvedValue(undefined)
    const store = useAuthStore()
    store.authenticated = true
    await store.logout()
    expect(mocked.logout).toHaveBeenCalledOnce()
    expect(store.authenticated).toBe(false)
  })

  it('logout still clears authenticated even when the request fails', async () => {
    mocked.logout.mockRejectedValue(new Error('boom'))
    const store = useAuthStore()
    store.authenticated = true
    // The error propagates (so the UI can toast it) but local state is cleared.
    await expect(store.logout()).rejects.toThrow('boom')
    expect(store.authenticated).toBe(false)
  })

  it('the onUnauthorized hook flips authenticated to false (session expiry)', () => {
    const store = useAuthStore()
    store.authenticated = true
    expect(hoisted.handler.current).toBeTypeOf('function')
    hoisted.handler.current?.()
    expect(store.authenticated).toBe(false)
  })
})
