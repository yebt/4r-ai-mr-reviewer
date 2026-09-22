/**
 * Generic fetch-based HTTP client for talking to the backend API.
 *
 * All requests are same-origin and rely on the httpOnly `air_session` cookie
 * managed entirely by the server — this layer never reads or writes it.
 */

export type HttpMethod = 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'

const API_BASE_URL = import.meta.env.VITE_API_URL ?? '/api'

/** Thrown for any non-2xx response. */
export class ApiError extends Error {
  readonly status: number
  readonly body?: unknown

  constructor(status: number, message: string, body?: unknown) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.body = body
  }
}

type UnauthorizedListener = () => void

const unauthorizedListeners = new Set<UnauthorizedListener>()

/**
 * Subscribe to global 401 events (any 401 on a path NOT under `/auth/`).
 * Returns an unsubscribe function.
 */
export function onUnauthorized(listener: UnauthorizedListener): () => void {
  unauthorizedListeners.add(listener)
  return () => unauthorizedListeners.delete(listener)
}

function notifyUnauthorized(): void {
  for (const listener of unauthorizedListeners) listener()
}

/** `/auth/*` requests (login/status/logout) never trigger the global 401 hook. */
function isAuthPath(path: string): boolean {
  return path.startsWith('/auth/')
}

async function parseBody(response: Response): Promise<unknown> {
  const text = await response.text()
  if (!text) return undefined
  try {
    return JSON.parse(text)
  } catch {
    return text
  }
}

/**
 * Perform a same-origin JSON request against the API.
 * Throws `ApiError` on any non-2xx response.
 */
export async function request<T>(method: HttpMethod, path: string, body?: unknown): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    method,
    credentials: 'same-origin',
    headers: body !== undefined ? { 'Content-Type': 'application/json' } : undefined,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  })

  const parsed = await parseBody(response)

  if (!response.ok) {
    if (response.status === 401 && !isAuthPath(path)) {
      notifyUnauthorized()
    }
    const message =
      (typeof parsed === 'object' && parsed && 'message' in parsed && typeof parsed.message === 'string'
        ? parsed.message
        : undefined) ??
      response.statusText ??
      `Request failed with status ${response.status}`
    throw new ApiError(response.status, message, parsed)
  }

  // A 2xx whose body isn't JSON means something other than the API answered
  // (e.g. a missing/mis-set dev proxy returning index.html). Fail loudly rather
  // than hand back a raw string typed as T — that is what rendered the blank
  // "." provider rows before the proxy existed.
  if (typeof parsed === 'string') {
    throw new ApiError(
      response.status,
      'Unexpected non-JSON response from the API (check the dev proxy / VITE_API_TARGET)',
      parsed,
    )
  }

  return parsed as T
}
