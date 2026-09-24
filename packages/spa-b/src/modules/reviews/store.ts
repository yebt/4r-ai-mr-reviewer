import { computed, ref, watch } from 'vue'
import { defineStore } from 'pinia'
import { useToast } from '@shared/composables/useToast'
import { resolveErrorMessage } from '@shared/api/resolveErrorMessage'
import { useInfiniteList } from '@shared/data/useInfiniteList'
import * as reviewsApi from './api'
import type { PublishSelection } from './api'
import type { ReviewWithRepo } from './types'

export { resolveErrorMessage }

/**
 * Reviews store — the global review list (every repo, newest first),
 * cursor-paginated via `useInfiniteList` (see `@shared/data/useInfiniteList`)
 * over `reviewsApi.listRecentReviews`. Replaces the earlier per-repo
 * `fetchAllReviews` fan-out (`Promise.allSettled` over every repo's
 * `GET /repos/{id}/reviews`, removed along with `api.ts#listRepoReviews`)
 * now that `GET /reviews` is global — see `api.ts`'s module doc. Mirrors
 * `modules/runs/store.ts`'s shape: not a `createCrudResource` (this
 * resource's mutations are custom action endpoints, not generic
 * create/update/remove), so the list + mutations are hand-rolled here too.
 *
 * `archived` is a plain ref toggle: flipping it resets the accumulated list
 * and reloads its first page from scratch, since active/archived are
 * disjoint result sets, not different pages of the same one.
 *
 * Unlike `useRunsStore`, this list does NOT poll on an interval: the earlier
 * @pinia/colada-backed store had no live-refresh behavior either (it only
 * ever refetched on an explicit `invalidate()` after a mutation), and the
 * detail page already owns its own 2.5s poll for the single review in view
 * (`useReviewDetail`, `reviews/detail.ts`) — so keeping this list free of a
 * second independent poller avoids two different intervals hitting the API
 * for overlapping reasons. The list instead refreshes on explicit actions:
 * a status-changing mutation (retry/approve) merges a fresh first page via
 * `refreshFirstPage()`, the archived toggle reloads from scratch, and the
 * section's error-retry button calls `refetch()`.
 */
export const useReviewsStore = defineStore('reviews', () => {
  const toast = useToast()

  const archived = ref(false)

  const list = useInfiniteList<ReviewWithRepo>((cursor) => reviewsApi.listRecentReviews(cursor, archived.value))

  const reviews = computed(() => list.items.value)

  // Normalizes `list.error` (`unknown`, since `useInfiniteList` is generic
  // and catches any thrown value) down to `Error | null` — `ReviewsListSection.vue`
  // reads `.message` off it, and the API layer only ever throws
  // `ApiError extends Error` anyway.
  const error = computed<Error | null>(() => (list.error.value instanceof Error ? list.error.value : null))

  // Archived is a disjoint result set from active, not a further page of it —
  // toggling it starts the accumulated list over from its own first page.
  watch(archived, () => {
    list.reset()
    void list.loadInitial()
  })

  void list.loadInitial()

  /** Reset + reload from the first page — used by the section's error-retry button. */
  function refetch(): Promise<void> {
    list.reset()
    return list.loadInitial()
  }

  /** Drops a review from the accumulated list by id, for mutations that change list membership (archive/unarchive/discard). */
  function removeReviewLocally(id: string): void {
    list.items.value = list.items.value.filter((review) => review.id !== id)
  }

  const isRetrying = ref(false)
  /** Posts a new retry `Review` server-side; the list keeps the original row and picks up the current state via `refreshFirstPage`. */
  async function retry(id: string): Promise<void> {
    isRetrying.value = true
    try {
      await reviewsApi.retryReview(id)
      toast.success('Review retried')
      await list.refreshFirstPage()
    } catch (err) {
      toast.error(resolveErrorMessage(err, 'Failed to retry review'))
      throw err
    } finally {
      isRetrying.value = false
    }
  }

  const isArchiving = ref(false)
  async function archive(id: string): Promise<void> {
    isArchiving.value = true
    try {
      await reviewsApi.archiveReview(id)
      removeReviewLocally(id)
      toast.success('Review archived')
    } catch (err) {
      toast.error(resolveErrorMessage(err, 'Failed to archive review'))
      throw err
    } finally {
      isArchiving.value = false
    }
  }

  const isUnarchiving = ref(false)
  async function unarchive(id: string): Promise<void> {
    isUnarchiving.value = true
    try {
      await reviewsApi.unarchiveReview(id)
      removeReviewLocally(id)
      toast.success('Review unarchived')
    } catch (err) {
      toast.error(resolveErrorMessage(err, 'Failed to unarchive review'))
      throw err
    } finally {
      isUnarchiving.value = false
    }
  }

  const isApproving = ref(false)
  async function approve(id: string): Promise<void> {
    isApproving.value = true
    try {
      await reviewsApi.approveReview(id)
      toast.success('Review approved')
      // Approve keeps the review in view (just changes its status) — merge
      // the refreshed first page so the row picks up its new status in place.
      await list.refreshFirstPage()
    } catch (err) {
      toast.error(resolveErrorMessage(err, 'Failed to approve review'))
      throw err
    } finally {
      isApproving.value = false
    }
  }

  const isDiscarding = ref(false)
  async function discard(id: string): Promise<void> {
    isDiscarding.value = true
    try {
      await reviewsApi.deleteReview(id)
      removeReviewLocally(id)
      toast.success('Review discarded')
    } catch (err) {
      toast.error(resolveErrorMessage(err, 'Failed to discard review'))
      throw err
    } finally {
      isDiscarding.value = false
    }
  }

  const isPublishing = ref(false)
  /** Posts findings/summary to the live GitLab MR — doesn't change list membership or status, so it never touches the loaded list. */
  async function publish(args: { id: string; selection: PublishSelection }): Promise<void> {
    isPublishing.value = true
    try {
      await reviewsApi.publishReview(args.id, args.selection)
      toast.success('Published to the MR')
    } catch (err) {
      toast.error(resolveErrorMessage(err, 'Failed to publish to the MR'))
      throw err
    } finally {
      isPublishing.value = false
    }
  }

  return {
    // Infinite-list surface backing the reviews list.
    reviews,
    isLoading: list.isLoading,
    isLoadingMore: list.isLoadingMore,
    hasMore: list.hasMore,
    error,
    loadMore: list.loadMore,
    refetch,
    archived,

    // mutations — each rethrows so callers keep their own try/catch (e.g.
    // clearing a row-local "pending" id in `finally`); the store already
    // toasted the error before rethrowing. Kept 1:1 with the earlier
    // @pinia/colada mutation surface so the detail page
    // (`src/pages/reviews/[id].vue`) and `ReviewsListSection.vue` need no
    // changes to their call sites.
    retry,
    isRetrying,
    archive,
    isArchiving,
    unarchive,
    isUnarchiving,
    approve,
    isApproving,
    discard,
    isDiscarding,
    publish,
    isPublishing,
  }
})
