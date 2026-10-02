import { afterEach, describe, expect, it, vi } from 'vitest'
import { ApiError, onUnauthorized, request } from './client'

function mockFetchOnce(status: number, body: unknown, statusText = '') {
  vi.stubGlobal(
    'fetch',
    vi.fn().mockResolvedValue({
      ok: status >= 200 && status < 300,
      status,
      statusText,
      text: async () => JSON.stringify(body),
    }),
  )
}

describe('api client', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('fires onUnauthorized listeners on a 401 for a non-/auth/ path', async () => {
    mockFetchOnce(401, { message: 'nope' })
    const listener = vi.fn()
    const unsubscribe = onUnauthorized(listener)

    await expect(request('GET', '/projects')).rejects.toBeInstanceOf(ApiError)

    expect(listener).toHaveBeenCalledTimes(1)
    unsubscribe()
  })

  it('does NOT fire onUnauthorized listeners on a 401 for an /auth/ path', async () => {
    mockFetchOnce(401, { message: 'invalid password' })
    const listener = vi.fn()
    const unsubscribe = onUnauthorized(listener)

    await expect(request('POST', '/auth/login', { password: 'wrong' })).rejects.toBeInstanceOf(
      ApiError,
    )

    expect(listener).not.toHaveBeenCalled()
    unsubscribe()
  })

  it('ApiError carries the response status and parsed body', async () => {
    mockFetchOnce(429, { message: 'slow down' })

    const error = (await request('POST', '/auth/login', { password: 'x' }).catch(
      (e) => e,
    )) as ApiError

    expect(error).toBeInstanceOf(ApiError)
    expect(error.status).toBe(429)
    expect(error.body).toEqual({ message: 'slow down' })
  })

  it('resolves parsed JSON on a 2xx response', async () => {
    mockFetchOnce(200, { authEnabled: true, authenticated: false })

    await expect(request('GET', '/auth/status')).resolves.toEqual({
      authEnabled: true,
      authenticated: false,
    })
  })
})
