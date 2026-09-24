import { request, requestPage, type Page } from '@shared/api/client'
import type { RoutineConfirmDecision, RoutineRun } from './types'

const DEFAULT_LIMIT = 30

/**
 * `GET /routines?limit&cursor[&archived=1]` — one keyset-paginated page of
 * the global run list (every repo, newest first, each row carrying
 * `repoName`), backed by `requestPage` (body = the run array, next cursor
 * in the `X-Next-Cursor` header; absent/empty = no more pages). `cursor` is
 * the opaque token from the previous page's `nextCursor`, or `null` for the
 * first page. Confirmed against the live backend on :8082 with real routine
 * data present.
 */
export function listRecentRoutines(
  cursor: string | null,
  archived = false,
  limit = DEFAULT_LIMIT,
): Promise<Page<RoutineRun>> {
  const params = new URLSearchParams({ limit: String(limit) })
  if (cursor) params.set('cursor', cursor)
  if (archived) params.set('archived', '1')
  return requestPage<RoutineRun>('GET', `/routines?${params.toString()}`)
}

/**
 * `GET /routines/{id}` — the full run (steps + state included), confirmed
 * against the live backend on :8082. Powers the detail page's query.
 */
export function getRoutine(id: string): Promise<RoutineRun> {
  return request<RoutineRun>('GET', `/routines/${id}`)
}

/** Unblocks a `blocked` run so it keeps progressing. */
export function resumeRoutine(id: string): Promise<void> {
  return request<void>('POST', `/routines/${id}/resume`)
}

/** Skips the current step of a `blocked` run. */
export function skipRoutine(id: string): Promise<void> {
  return request<void>('POST', `/routines/${id}/skip`)
}

/** Records the human decision for an `awaiting_confirmation` run. */
export function confirmRoutine(id: string, decision: RoutineConfirmDecision): Promise<void> {
  return request<void>('POST', `/routines/${id}/confirm`, { decision })
}

export function archiveRoutine(id: string): Promise<void> {
  return request<void>('POST', `/routines/${id}/archive`)
}

export function unarchiveRoutine(id: string): Promise<void> {
  return request<void>('POST', `/routines/${id}/unarchive`)
}

export function cancelRoutine(id: string): Promise<void> {
  return request<void>('POST', `/routines/${id}/cancel`)
}

/** 409 if the run is still running — surfaced to the caller as an `ApiError`. */
export function deleteRoutine(id: string): Promise<void> {
  return request<void>('DELETE', `/routines/${id}`)
}
