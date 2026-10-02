import { ApiError } from '@shared/api/client'

/** Maps a login-flow error into a user-facing message. */
export function errorMessage(e: unknown): string {
  if (e instanceof ApiError) {
    if (e.status === 401) return 'invalid password'
    if (e.status === 429) return 'too many attempts, wait a minute'
    return e.message || 'something went wrong'
  }
  return 'something went wrong'
}
