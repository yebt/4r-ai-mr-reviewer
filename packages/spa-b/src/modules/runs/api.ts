import { request } from '@shared/api/client'
import type { RoutineRun } from './types'

const DEFAULT_LIMIT = 30

/**
 * `GET /routines?limit=N[&archived=1]` — the global run list (every repo,
 * newest first, each row carrying `repoName`). Confirmed against the live
 * backend on :8082 with real routine data present.
 */
export function listRecentRoutines(limit = DEFAULT_LIMIT, archived = false): Promise<RoutineRun[]> {
  const params = new URLSearchParams({ limit: String(limit) })
  if (archived) params.set('archived', '1')
  return request<RoutineRun[]>('GET', `/routines?${params.toString()}`)
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
