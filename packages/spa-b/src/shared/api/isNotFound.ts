import { ApiError } from './client'

/** True when the request failed with HTTP 404 — the thing asked for does not (or no longer) exist. */
export function isNotFound(err: unknown): boolean {
  return err instanceof ApiError && err.status === 404
}
