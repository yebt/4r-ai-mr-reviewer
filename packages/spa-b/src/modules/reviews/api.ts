import { request } from '@shared/api/client'
import type { FindingHumanized, Humanizations, Review, SummaryHumanized } from './types'

/**
 * There is no global reviews endpoint — reviews are always scoped to a
 * repo. `archived` (default false) toggles between the live and archived
 * list for that repo.
 */
export function listRepoReviews(repoId: string, archived = false): Promise<Review[]> {
  const query = archived ? '?archived=1' : ''
  return request<Review[]>('GET', `/repos/${repoId}/reviews${query}`)
}

/** The full `Review` (with `findings`/`reasonings`) for the detail page. */
export function getReview(id: string): Promise<Review> {
  return request<Review>('GET', `/reviews/${id}`)
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
