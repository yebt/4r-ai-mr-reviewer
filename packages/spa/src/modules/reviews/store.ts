import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api, errorMessage } from '@shared/api/client'
import type { MergeRequest, Review } from '@shared/api/types'

export const useReviewsStore = defineStore('reviews', () => {
  // Repo-detail view: open MRs + the repo's reviews.
  const mrs = ref<MergeRequest[]>([])
  const mrsLoading = ref(false)
  const mrsError = ref<string | null>(null)

  const list = ref<Review[]>([])
  const listLoading = ref(false)
  const listError = ref<string | null>(null)

  // Review-detail view: one review.
  const current = ref<Review | null>(null)
  const currentLoading = ref(false)
  const currentError = ref<string | null>(null)

  async function fetchMergeRequests(repoId: string) {
    mrsLoading.value = true
    mrsError.value = null
    try {
      mrs.value = await api.listMergeRequests(repoId)
    } catch (e) {
      mrsError.value = errorMessage(e)
      mrs.value = []
    } finally {
      mrsLoading.value = false
    }
  }

  async function fetchReviews(repoId: string) {
    listLoading.value = true
    listError.value = null
    try {
      list.value = await api.listRepoReviews(repoId)
    } catch (e) {
      listError.value = errorMessage(e)
    } finally {
      listLoading.value = false
    }
  }

  async function create(repoId: string, mrIid: number, mode: string) {
    const created = await api.createReview({ repoId, mrIid, mode })
    list.value = [created, ...list.value]
    return created
  }

  async function load(id: string) {
    currentLoading.value = true
    currentError.value = null
    try {
      current.value = await api.getReview(id)
    } catch (e) {
      currentError.value = errorMessage(e)
      current.value = null
    } finally {
      currentLoading.value = false
    }
  }

  // refresh silently updates the current review (used while polling).
  async function refresh(id: string) {
    current.value = await api.getReview(id)
  }

  async function retry(id: string) {
    return api.retryReview(id)
  }

  async function publish(id: string, selection: { all?: boolean; indices?: number[] }) {
    await api.publishReview(id, selection)
    await refresh(id)
  }

  return {
    mrs,
    mrsLoading,
    mrsError,
    list,
    listLoading,
    listError,
    current,
    currentLoading,
    currentError,
    fetchMergeRequests,
    fetchReviews,
    create,
    load,
    refresh,
    retry,
    publish,
  }
})
