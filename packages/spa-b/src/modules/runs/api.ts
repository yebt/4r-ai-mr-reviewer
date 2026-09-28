import { request, requestPage, type Page } from '@shared/api/client'
import type {
  CreateMainReleaseInput,
  CreateReleaseInput,
  PreviewTagQuery,
  PreviewTagResult,
  RoutineConfirmDecision,
  RoutineRun,
} from './types'

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

/**
 * Query key for a repo's own routine-run list (`GET /repos/{id}/routines`)
 * — NOT the global cursor-paginated `GET /routines` list
 * `listRecentRoutines` backs. Mirrors
 * `modules/reviews/api.ts#repoReviewsQueryKey`'s shape convention.
 */
export function repoRoutinesQueryKey(repoId: string): readonly [string, string] {
  return ['repo-routines', repoId]
}

/**
 * `GET /repos/{id}/routines` — a single repo's routine runs, newest first.
 * Unlike `listRecentRoutines`, this is a plain, non-paginated array with no
 * `repoName` (the caller already knows the repo). Backs the Flow
 * workspace's Actions tab (`modules/flow`).
 */
export function listRepoRoutines(repoId: string): Promise<RoutineRun[]> {
  return request<RoutineRun[]>('GET', `/repos/${repoId}/routines`)
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

/**
 * `POST /repos/{id}/routines/release` — dev-flow release for an existing MR
 * (`ReleaseDialog.vue` with `flow="dev"`). 400 for an invalid bump/MR id or a
 * non-`development` target; 409 (`ErrDuplicateRun`) when the MR already has
 * an active release run.
 */
export function createRelease(repoId: string, input: CreateReleaseInput): Promise<RoutineRun> {
  return request<RoutineRun>('POST', `/repos/${repoId}/routines/release`, input)
}

/**
 * `POST /repos/{id}/routines/release-main` — main-flow release
 * (`ReleaseDialog.vue` with `flow="main"`); creates the source→target MR
 * itself, so no MR needs to exist yet. 400 for an invalid bump or
 * same-branch source/target; 409 (`ErrDuplicateRun`) when the repo already
 * has an active main run targeting that branch.
 */
export function createMainRelease(repoId: string, input: CreateMainReleaseInput): Promise<RoutineRun> {
  return request<RoutineRun>('POST', `/repos/${repoId}/routines/release-main`, input)
}

/**
 * `GET /repos/{id}/routines/preview-tag` — a dry-run of the exact next
 * release tag (mirrors the `compute_tag` step), so `ReleaseDialog.vue` can
 * show it before launch. Omits optional query params rather than sending
 * empty values, matching `previewRoutineTag`'s server-side parsing.
 */
export function previewRoutineTag(repoId: string, query: PreviewTagQuery): Promise<PreviewTagResult> {
  const params = new URLSearchParams({ flow: query.flow, bump: query.bump })
  if (query.mrIid != null) params.set('mrIid', String(query.mrIid))
  if (query.source) params.set('source', query.source)
  if (query.target) params.set('target', query.target)
  if (query.includeDev) params.set('includeDev', '1')
  return request<PreviewTagResult>('GET', `/repos/${repoId}/routines/preview-tag?${params.toString()}`)
}
