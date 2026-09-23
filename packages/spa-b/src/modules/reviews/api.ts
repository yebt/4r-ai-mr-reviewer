import { request } from '@shared/api/client'
import type { Review } from './types'

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
