import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { api, errorMessage } from '@shared/api/client'
import type { HumanizeVariant, MergeRequest, Review } from '@shared/api/types'

export const useReviewsStore = defineStore('reviews', () => {
  // Caches keyed by repo id, so revisiting a repo shows its previous MRs/reviews
  // immediately while we revalidate in the background (no loading flash).
  const mrsByRepo = ref<Record<string, MergeRequest[]>>({})
  const mrsLoading = ref(false)
  const mrsError = ref<string | null>(null)

  const reviewsByRepo = ref<Record<string, Review[]>>({})
  const listLoading = ref(false)
  const listError = ref<string | null>(null)

  // Archived reviews live in a separate cache slot so a "Show archived" toggle
  // can render them without clobbering the active list.
  const archivedByRepo = ref<Record<string, Review[]>>({})
  const archivedLoading = ref(false)
  const archivedError = ref<string | null>(null)

  // Review-detail cache keyed by review id.
  const reviewsById = ref<Record<string, Review>>({})
  const current = ref<Review | null>(null)
  const currentLoading = ref(false)
  const currentError = ref<string | null>(null)

  function mergeRequestsFor(repoId: string): MergeRequest[] {
    return mrsByRepo.value[repoId] ?? []
  }

  function reviewsFor(repoId: string): Review[] {
    return reviewsByRepo.value[repoId] ?? []
  }

  function archivedReviewsFor(repoId: string): Review[] {
    return archivedByRepo.value[repoId] ?? []
  }

  // Flattened view of every cached review, newest first — for the global list.
  const allReviews = computed(() =>
    Object.values(reviewsByRepo.value)
      .flat()
      .sort((a, b) => (a.createdAt < b.createdAt ? 1 : -1)),
  )

  async function fetchMergeRequests(repoId: string) {
    mrsLoading.value = true
    mrsError.value = null
    try {
      const data = await api.listMergeRequests(repoId)
      mrsByRepo.value = { ...mrsByRepo.value, [repoId]: data }
    } catch (e) {
      mrsError.value = errorMessage(e)
    } finally {
      mrsLoading.value = false
    }
  }

  async function fetchReviews(repoId: string) {
    listLoading.value = true
    listError.value = null
    try {
      const data = await api.listRepoReviews(repoId)
      reviewsByRepo.value = { ...reviewsByRepo.value, [repoId]: data }
    } catch (e) {
      listError.value = errorMessage(e)
    } finally {
      listLoading.value = false
    }
  }

  async function fetchArchivedReviews(repoId: string) {
    archivedLoading.value = true
    archivedError.value = null
    try {
      const data = await api.listRepoReviews(repoId, true)
      archivedByRepo.value = { ...archivedByRepo.value, [repoId]: data }
    } catch (e) {
      archivedError.value = errorMessage(e)
    } finally {
      archivedLoading.value = false
    }
  }

  function cacheReview(rv: Review) {
    reviewsById.value[rv.id] = rv
    const list = reviewsByRepo.value[rv.repoId]
    if (list) {
      const exists = list.some((r) => r.id === rv.id)
      reviewsByRepo.value = {
        ...reviewsByRepo.value,
        [rv.repoId]: exists ? list.map((r) => (r.id === rv.id ? rv : r)) : [rv, ...list],
      }
    }
  }

  async function create(repoId: string, mrIid: number, mode: string) {
    const created = await api.createReview({ repoId, mrIid, mode })
    cacheReview(created)
    return created
  }

  async function load(id: string) {
    const cached = reviewsById.value[id]
    if (cached) current.value = cached
    // Only show the spinner when we have nothing to show yet.
    currentLoading.value = !cached
    currentError.value = null
    try {
      const rv = await api.getReview(id)
      current.value = rv
      cacheReview(rv)
    } catch (e) {
      currentError.value = errorMessage(e)
    } finally {
      currentLoading.value = false
    }
  }

  // refresh silently updates the current review (used while polling).
  async function refresh(id: string) {
    const rv = await api.getReview(id)
    current.value = rv
    cacheReview(rv)
  }

  async function retry(id: string) {
    return api.retryReview(id)
  }

  // cancel requests cooperative cancellation of a pending/running review, then
  // refreshes so the UI observes the flip to the cancelled terminal state.
  async function cancel(id: string) {
    await api.cancelReview(id)
    await refresh(id)
  }

  // remove hard-deletes a review and purges it from every cache.
  async function remove(id: string) {
    await api.deleteReview(id)
    const { [id]: _removed, ...rest } = reviewsById.value
    reviewsById.value = rest
    reviewsByRepo.value = Object.fromEntries(
      Object.entries(reviewsByRepo.value).map(([repoId, list]) => [
        repoId,
        list.filter((r) => r.id !== id),
      ]),
    )
    if (current.value?.id === id) current.value = null
  }

  // archive soft-hides a review: it drops out of the active repo list and its
  // archived flag flips across the detail caches.
  async function archive(id: string) {
    await api.archiveReview(id)
    setArchivedFlag(id, true)
    // Remove from every active repo list; drop the whole archived cache slot so
    // it is refetched fresh when the toggle is next opened.
    reviewsByRepo.value = Object.fromEntries(
      Object.entries(reviewsByRepo.value).map(([repoId, list]) => [
        repoId,
        list.filter((r) => r.id !== id),
      ]),
    )
    const repoId = reviewsById.value[id]?.repoId ?? current.value?.repoId
    if (repoId) {
      const { [repoId]: _dropped, ...rest } = archivedByRepo.value
      archivedByRepo.value = rest
    }
  }

  // unarchive restores a review to the active list.
  async function unarchive(id: string) {
    await api.unarchiveReview(id)
    setArchivedFlag(id, false)
    archivedByRepo.value = Object.fromEntries(
      Object.entries(archivedByRepo.value).map(([repoId, list]) => [
        repoId,
        list.filter((r) => r.id !== id),
      ]),
    )
    const repoId = reviewsById.value[id]?.repoId ?? current.value?.repoId
    if (repoId) {
      const { [repoId]: _dropped, ...rest } = reviewsByRepo.value
      reviewsByRepo.value = rest
    }
  }

  // setArchivedFlag keeps the detail caches (by id + current) in sync.
  function setArchivedFlag(id: string, archived: boolean) {
    const cached = reviewsById.value[id]
    if (cached) reviewsById.value[id] = { ...cached, archived }
    if (current.value?.id === id) current.value = { ...current.value, archived }
  }

  async function publish(
    id: string,
    selection: { all?: boolean; indices?: number[]; includeSummary?: boolean },
  ) {
    await api.publishReview(id, selection)
    await refresh(id)
  }

  // humanize returns ephemeral rewritten variants of a finished review in the
  // given profile's voice. Nothing is persisted, so callers own the result.
  async function humanize(id: string, profileId: string, count = 3): Promise<HumanizeVariant[]> {
    const { variants } = await api.humanizeReview(id, { profileId, count })
    return variants
  }

  return {
    mrsByRepo,
    mrsLoading,
    mrsError,
    reviewsByRepo,
    listLoading,
    listError,
    archivedByRepo,
    archivedLoading,
    archivedError,
    reviewsById,
    current,
    currentLoading,
    currentError,
    mergeRequestsFor,
    reviewsFor,
    archivedReviewsFor,
    allReviews,
    fetchMergeRequests,
    fetchReviews,
    fetchArchivedReviews,
    create,
    load,
    refresh,
    retry,
    cancel,
    remove,
    archive,
    unarchive,
    publish,
    humanize,
  }
})
