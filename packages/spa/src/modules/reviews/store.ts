import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api, errorMessage } from '@shared/api/client'
import type { MergeRequest, Review } from '@shared/api/types'

export const useReviewsStore = defineStore('reviews', () => {
  // Caches keyed by repo id, so revisiting a repo shows its previous MRs/reviews
  // immediately while we revalidate in the background (no loading flash).
  const mrsByRepo = ref<Record<string, MergeRequest[]>>({})
  const mrsLoading = ref(false)
  const mrsError = ref<string | null>(null)

  const reviewsByRepo = ref<Record<string, Review[]>>({})
  const listLoading = ref(false)
  const listError = ref<string | null>(null)

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

  async function publish(id: string, selection: { all?: boolean; indices?: number[] }) {
    await api.publishReview(id, selection)
    await refresh(id)
  }

  return {
    mrsByRepo,
    mrsLoading,
    mrsError,
    reviewsByRepo,
    listLoading,
    listError,
    reviewsById,
    current,
    currentLoading,
    currentError,
    mergeRequestsFor,
    reviewsFor,
    fetchMergeRequests,
    fetchReviews,
    create,
    load,
    refresh,
    retry,
    publish,
  }
})
