import { request, requestPage, type Page } from '@shared/api/client'
import type { FindingHumanized, Humanizations, Review, ReviewWithRepo, SummaryHumanized } from './types'

const DEFAULT_LIMIT = 30

/** Wire shape of one row off the global list endpoint — `repoName` is `omitempty` server-side, so it's optional here and normalized to a fallback below before it reaches `ReviewWithRepo` (which every other consumer, e.g. `filters.ts`, treats as always-present). */
type RawReviewListItem = Review & { repoName?: string }

/**
 * `GET /reviews?limit&cursor[&archived=1]` — one keyset-paginated page of
 * the global review list (every repo, newest first, each row carrying
 * `repoName`), backed by `requestPage` (body = the review array, next
 * cursor in the `X-Next-Cursor` response header; absent/empty = no more
 * pages). `cursor` is the opaque token from the previous page's
 * `nextCursor`, or `null` for the first page. Mirrors
 * `modules/runs/api.ts#listRecentRoutines`. Replaces the earlier per-repo
 * `listRepoReviews` fan-out (removed — see `store.ts`'s module doc).
 */
export function listRecentReviews(
  cursor: string | null,
  archived = false,
  limit = DEFAULT_LIMIT,
): Promise<Page<ReviewWithRepo>> {
  const params = new URLSearchParams({ limit: String(limit) })
  if (cursor) params.set('cursor', cursor)
  if (archived) params.set('archived', '1')
  return requestPage<RawReviewListItem>('GET', `/reviews?${params.toString()}`).then((page) => ({
    nextCursor: page.nextCursor,
    items: page.items.map((review) => ({ ...review, repoName: review.repoName ?? 'Unknown repo' })),
  }))
}

/** The full `Review` (with `findings`/`reasonings`) for the detail page. */
export function getReview(id: string): Promise<Review> {
  return request<Review>('GET', `/reviews/${id}`)
}

/**
 * Query key for a repo's own review list (`GET /repos/{id}/reviews`) — NOT
 * the global cursor-paginated `GET /reviews` list `listRecentReviews` backs.
 * Exported so `modules/flow`'s MRs tab (best-effort latest-review-status
 * lookup) and Reviews tab share the exact same @pinia/colada cache entry
 * instead of fetching this list twice per repo visit.
 */
export function repoReviewsQueryKey(repoId: string): readonly [string, string] {
  return ['repo-reviews', repoId]
}

/**
 * `GET /repos/{id}/reviews` — a single repo's reviews, newest first. Unlike
 * `listRecentReviews`, this is a plain, non-paginated array with no
 * `repoName` (the caller already knows the repo — see `reviewResp`'s doc
 * server-side). Backs the Flow workspace's Reviews tab (`modules/flow`).
 */
export function listRepoReviews(repoId: string): Promise<Review[]> {
  return request<Review[]>('GET', `/repos/${repoId}/reviews`)
}

/** Returns the newly created retry `Review` (201). */
export function retryReview(id: string): Promise<Review> {
  return request<Review>('POST', `/reviews/${id}/retry`)
}

export function approveReview(id: string): Promise<void> {
  return request<void>('POST', `/reviews/${id}/approve`)
}

export function archiveReview(id: string): Promise<void> {
  return request<void>('POST', `/reviews/${id}/archive`)
}

export function unarchiveReview(id: string): Promise<void> {
  return request<void>('POST', `/reviews/${id}/unarchive`)
}

export function deleteReview(id: string): Promise<void> {
  return request<void>('DELETE', `/reviews/${id}`)
}

/**
 * Selection payload for `POST /reviews/{id}/publish`. All fields optional —
 * `all: true` publishes every not-yet-published finding plus the summary (if
 * not yet published); otherwise `indices`/`includeSummary` select what to
 * post. `summaryOverride`/`findingOverrides` (slice 2) replace the generated
 * body wholesale with humanized text — see `modules/reviews/humanize.ts`.
 */
export interface PublishSelection {
  all?: boolean
  indices?: number[]
  includeSummary?: boolean
  summaryOverride?: string
  findingOverrides?: { index: number; text: string }[]
}

/** Posts findings/summary to the live GitLab MR. The 200 body is `{status:"published"}`; callers only need the outcome. */
export function publishReview(id: string, selection: PublishSelection): Promise<void> {
  return request<{ status: string }>('POST', `/reviews/${id}/publish`, selection).then(() => undefined)
}

/**
 * Runs one humanize pass over a single finding, in `profileId`'s voice. Each
 * run is persisted server-side (see `getHumanizations`), so calling this
 * again for the same finding produces a new tab, not a replacement.
 */
export function humanizeFinding(id: string, profileId: string, index: number): Promise<FindingHumanized> {
  return request<FindingHumanized>('POST', `/reviews/${id}/humanize`, { profileId, target: 'finding', index })
}

/** Runs one humanize pass over the review summary, in `profileId`'s voice. */
export function humanizeSummary(id: string, profileId: string): Promise<SummaryHumanized> {
  return request<SummaryHumanized>('POST', `/reviews/${id}/humanize`, { profileId, target: 'summary' })
}

/** Every past humanize run for a review, in run order (= tab order). */
export function getHumanizations(id: string): Promise<Humanizations> {
  return request<Humanizations>('GET', `/reviews/${id}/humanizations`)
}
