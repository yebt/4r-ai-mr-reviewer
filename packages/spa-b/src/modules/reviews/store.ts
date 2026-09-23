import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { useMutation, useQuery, useQueryCache } from '@pinia/colada'
import { useToast } from '@shared/composables/useToast'
import { resolveErrorMessage } from '@shared/api/resolveErrorMessage'
import { listRepos } from '@modules/repos/api'
import * as reviewsApi from './api'
import type { ReviewWithRepo } from './types'

export const REVIEWS_QUERY_KEY = 'reviews' as const

/**
 * There is no global reviews endpoint — the list is a fan-out over every
 * repo's `GET /repos/{id}/reviews`, with each row's owning `repoName`
 * attached so the list can render `repoName · !{mrIid}` without a second
 * lookup. Kept self-contained (calls `repos/api` directly) rather than
 * depending on `useReposStore`, so this module's data layer doesn't need a
 * live repos store instance to function.
 */
export async function fetchAllReviews(archived: boolean): Promise<ReviewWithRepo[]> {
  const repos = await listRepos()
  const perRepo = await Promise.all(
    repos.map((repo) =>
      reviewsApi
        .listRepoReviews(repo.id, archived)
        .then((reviews) => reviews.map((review) => ({ ...review, repoName: repo.name }))),
    ),
  )
  return perRepo.flat()
}

/**
 * Reviews store, backed by @pinia/colada. Unlike the settings modules
 * (providers/repos/...), reviews are not simple CRUD — there's no single
 * `POST/PATCH/DELETE /reviews` resource to build on `createCrudResource`,
 * so the query and its mutations are hand-rolled here.
 *
 * The `['reviews', { archived }]` query key is reactive: flipping `archived`
 * changes the key, which lazily fetches the archived fan-out on first use
 * instead of always fetching both up front.
 *
 * Every mutation (retry/archive/unarchive/approve/discard) is a one-off
 * state transition on a single review — there's no optimistic list patch to
 * maintain (unlike an in-place field edit), so each just awaits the
 * request, toasts success/error, and invalidates `['reviews']` to refetch
 * the authoritative list.
 */
export const useReviewsStore = defineStore('reviews', () => {
  const queryCache = useQueryCache()
  const toast = useToast()

  const archived = ref(false)

  const query = useQuery({
    key: () => [REVIEWS_QUERY_KEY, { archived: archived.value }],
    query: () => fetchAllReviews(archived.value),
  })

  const reviews = computed(() => query.data.value ?? [])

  function invalidate() {
    queryCache.invalidateQueries({ key: [REVIEWS_QUERY_KEY] })
  }

  const retryMutation = useMutation({
    mutation: (id: string) => reviewsApi.retryReview(id),
    onSuccess() {
      toast.success('Review retry queued')
    },
    onError(err) {
      toast.error(resolveErrorMessage(err, 'Failed to retry review'))
    },
    onSettled: invalidate,
  })

  const archiveMutation = useMutation({
    mutation: (id: string) => reviewsApi.archiveReview(id),
    onSuccess() {
      toast.success('Review archived')
    },
    onError(err) {
      toast.error(resolveErrorMessage(err, 'Failed to archive review'))
    },
    onSettled: invalidate,
  })

  const unarchiveMutation = useMutation({
    mutation: (id: string) => reviewsApi.unarchiveReview(id),
    onSuccess() {
      toast.success('Review unarchived')
    },
    onError(err) {
      toast.error(resolveErrorMessage(err, 'Failed to unarchive review'))
    },
    onSettled: invalidate,
  })

  const approveMutation = useMutation({
    mutation: (id: string) => reviewsApi.approveReview(id),
    onSuccess() {
      toast.success('Review approved')
    },
    onError(err) {
      toast.error(resolveErrorMessage(err, 'Failed to approve review'))
    },
    onSettled: invalidate,
  })

  const discardMutation = useMutation({
    mutation: (id: string) => reviewsApi.deleteReview(id),
    onSuccess() {
      toast.success('Review discarded')
    },
    onError(err) {
      toast.error(resolveErrorMessage(err, 'Failed to discard review'))
    },
    onSettled: invalidate,
  })

  return {
    // ['reviews', { archived }] query surface
    reviews,
    state: query.state,
    asyncStatus: query.asyncStatus,
    isLoading: query.isLoading,
    error: query.error,
    refetch: query.refetch,
    archived,

    // mutations — `mutateAsync` rethrows so callers can keep their own
    // try/catch flows (e.g. clearing a row-local "pending" id) on top of
    // the toast + invalidate handled above.
    retry: retryMutation.mutateAsync,
    isRetrying: retryMutation.isLoading,
    archive: archiveMutation.mutateAsync,
    isArchiving: archiveMutation.isLoading,
    unarchive: unarchiveMutation.mutateAsync,
    isUnarchiving: unarchiveMutation.isLoading,
    approve: approveMutation.mutateAsync,
    isApproving: approveMutation.isLoading,
    discard: discardMutation.mutateAsync,
    isDiscarding: discardMutation.isLoading,
  }
})
