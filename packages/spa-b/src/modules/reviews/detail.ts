/**
 * Single-review detail query + live polling, for `src/pages/reviews/[id].vue`.
 * Page-local rather than a Pinia store — there's only ever one detail page
 * mounted at a time, so a full store (like `reviews/store.ts`'s list) buys
 * nothing here.
 *
 * Polling mirrors `modules/runs/store.ts`'s shape: `useIntervalFn` refetches
 * every 2.5s, gated on the review being non-terminal (`isReviewActive`) AND
 * the tab being visible (`useDocumentVisibility` — a backgrounded tab has no
 * reason to keep hitting the API), driven by a `watch(canPoll, ...)`
 * (immediate, so a page load that already has a running review starts
 * polling right away). `pausePolling` lets the page stop the interval
 * explicitly on unmount — `useIntervalFn`'s own cleanup only fires when
 * *this composable's calling component* unmounts, which the page always is
 * here, but exposing it keeps the shape consistent with the list store.
 */
import { computed, toValue, watch, type MaybeRefOrGetter } from 'vue'
import { useQuery } from '@pinia/colada'
import { useDocumentVisibility, useIntervalFn } from '@vueuse/core'
import * as reviewsApi from './api'
import type { ReviewStatus } from './types'

export const REVIEW_QUERY_KEY = 'review' as const

const POLL_INTERVAL_MS = 2500

/** Non-terminal statuses that should keep the detail page polling. */
export function isReviewActive(status: ReviewStatus): boolean {
  return status === 'pending' || status === 'running'
}

export function useReviewDetail(id: MaybeRefOrGetter<string>) {
  const query = useQuery({
    key: () => [REVIEW_QUERY_KEY, toValue(id)],
    query: () => reviewsApi.getReview(toValue(id)),
  })

  const documentVisibility = useDocumentVisibility()
  const canPoll = computed(() => {
    const review = query.data.value
    return !!review && isReviewActive(review.status) && documentVisibility.value === 'visible'
  })

  const { pause, resume, isActive: isPolling } = useIntervalFn(() => query.refetch(), POLL_INTERVAL_MS, {
    immediate: false,
  })

  watch(canPoll, (active) => (active ? resume() : pause()), { immediate: true })

  return {
    review: query.data,
    state: query.state,
    isLoading: query.isLoading,
    error: query.error,
    refetch: query.refetch,
    isPolling,
    pausePolling: pause,
  }
}
