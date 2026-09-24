import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { defineComponent } from 'vue'
import * as reviewsApi from './api'
import { useReviewsStore } from './store'
import type { ReviewWithRepo } from './types'

vi.mock('./api')

// `vi.mock` factories are hoisted above regular top-level statements, so the
// spies they close over must be created via `vi.hoisted` (same pattern as
// `modules/runs/store.spec.ts`).
const { mockToastSuccess, mockToastError } = vi.hoisted(() => ({
  mockToastSuccess: vi.fn(),
  mockToastError: vi.fn(),
}))

vi.mock('@shared/composables/useToast', () => ({
  useToast: () => ({
    success: mockToastSuccess,
    error: mockToastError,
    info: vi.fn(),
  }),
}))

const mockedListRecentReviews = vi.mocked(reviewsApi.listRecentReviews)
const mockedRetryReview = vi.mocked(reviewsApi.retryReview)
const mockedArchiveReview = vi.mocked(reviewsApi.archiveReview)
const mockedApproveReview = vi.mocked(reviewsApi.approveReview)
const mockedDeleteReview = vi.mocked(reviewsApi.deleteReview)
const mockedPublishReview = vi.mocked(reviewsApi.publishReview)

function makeReview(overrides: Partial<ReviewWithRepo> = {}): ReviewWithRepo {
  return {
    id: 'r1',
    repoId: 'repo-1',
    repoName: 'alpha',
    mrIid: 1,
    contextMode: 'fast',
    status: 'pending',
    phase: 'queued',
    archived: false,
    summaryPublished: false,
    summary: '',
    recommendation: 'comment',
    score: 0,
    inputTokens: 0,
    outputTokens: 0,
    findings: [],
    reasonings: [],
    createdAt: '2026-01-01T00:00:00.000Z',
    updatedAt: '2026-01-01T00:00:00.000Z',
    ...overrides,
  }
}

/**
 * Mounts a throwaway component that just instantiates `useReviewsStore()`
 * under a real Pinia app context — required for `defineStore`'s injection
 * context. The store no longer uses `@pinia/colada` (the list is backed by
 * `useInfiniteList` instead), so no `PiniaColada` plugin is needed here —
 * mirrors `modules/runs/store.spec.ts`.
 */
function mountStore() {
  let store!: ReturnType<typeof useReviewsStore>
  const Harness = defineComponent({
    setup() {
      store = useReviewsStore()
      return () => null
    },
  })
  const wrapper = mount(Harness, {
    global: { plugins: [createPinia()] },
  })
  return { wrapper, store }
}

describe('useReviewsStore (useInfiniteList-backed)', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('loads the first page on mount and maps it into store.reviews', async () => {
    const reviews = [makeReview()]
    mockedListRecentReviews.mockResolvedValueOnce({ items: reviews, nextCursor: null })

    const { store } = mountStore()
    await flushPromises()

    expect(mockedListRecentReviews).toHaveBeenCalledWith(null, false)
    expect(store.reviews).toEqual(reviews)
    expect(store.isLoading).toBe(false)
    expect(store.error).toBeNull()
    expect(store.hasMore).toBe(false)
  })

  it('hasMore reflects a non-null nextCursor, and loadMore appends the next page', async () => {
    mockedListRecentReviews.mockResolvedValueOnce({
      items: [makeReview({ id: 'r1' })],
      nextCursor: 'cursor-1',
    })
    const { store } = mountStore()
    await flushPromises()
    expect(store.hasMore).toBe(true)

    mockedListRecentReviews.mockResolvedValueOnce({
      items: [makeReview({ id: 'r2' })],
      nextCursor: null,
    })
    await store.loadMore()
    await flushPromises()

    expect(mockedListRecentReviews).toHaveBeenLastCalledWith('cursor-1', false)
    expect(store.reviews.map((review) => review.id)).toEqual(['r1', 'r2'])
    expect(store.hasMore).toBe(false)
  })

  it('toggling archived resets and re-queries with archived=true', async () => {
    mockedListRecentReviews.mockResolvedValueOnce({ items: [], nextCursor: null })
    const { store } = mountStore()
    await flushPromises()

    mockedListRecentReviews.mockResolvedValueOnce({
      items: [makeReview({ archived: true })],
      nextCursor: null,
    })
    store.archived = true
    await flushPromises()

    expect(mockedListRecentReviews).toHaveBeenCalledWith(null, true)
    expect(store.reviews.every((review) => review.archived)).toBe(true)
  })

  it('an error surfaces on store.error and refetch() resets + reloads', async () => {
    mockedListRecentReviews.mockRejectedValueOnce(new Error('Network down'))
    const { store } = mountStore()
    await flushPromises()

    expect(store.error?.message).toBe('Network down')

    mockedListRecentReviews.mockResolvedValueOnce({ items: [makeReview()], nextCursor: null })
    await store.refetch()
    await flushPromises()

    expect(store.error).toBeNull()
    expect(store.reviews).toEqual([makeReview()])
  })

  it('retry() awaits the request, toasts success, and merges a refreshed first page', async () => {
    mockedListRecentReviews.mockResolvedValueOnce({
      items: [makeReview({ id: 'r1', status: 'error' })],
      nextCursor: null,
    })
    const { store } = mountStore()
    await flushPromises()

    mockedRetryReview.mockResolvedValueOnce(makeReview({ id: 'r2', status: 'pending' }))
    mockedListRecentReviews.mockResolvedValueOnce({
      items: [makeReview({ id: 'r1', status: 'error' }), makeReview({ id: 'r2', status: 'pending' })],
      nextCursor: null,
    })

    await store.retry('r1')
    await flushPromises()

    expect(mockedRetryReview).toHaveBeenCalledWith('r1')
    expect(mockToastSuccess).toHaveBeenCalledWith('Review retried')
    expect(store.reviews.map((review) => review.id)).toEqual(['r2', 'r1'])
  })

  it('archive() removes the review from the accumulated list locally and toasts on success', async () => {
    mockedListRecentReviews.mockResolvedValueOnce({
      items: [makeReview({ id: 'r1', archived: false })],
      nextCursor: null,
    })
    const { store } = mountStore()
    await flushPromises()

    mockedArchiveReview.mockResolvedValueOnce(undefined)

    await store.archive('r1')
    await flushPromises()

    expect(mockedArchiveReview).toHaveBeenCalledWith('r1')
    expect(mockToastSuccess).toHaveBeenCalledWith('Review archived')
    expect(store.reviews).toEqual([])
    // No colada invalidate/refetch anymore — membership change is applied locally.
    expect(mockedListRecentReviews).toHaveBeenCalledTimes(1)
  })

  it('archive() toasts an error and rethrows when the request rejects, without touching the list', async () => {
    mockedListRecentReviews.mockResolvedValueOnce({
      items: [makeReview({ id: 'r1' })],
      nextCursor: null,
    })
    const { store } = mountStore()
    await flushPromises()

    mockedArchiveReview.mockRejectedValueOnce(new Error('Archive failed'))

    await expect(store.archive('r1')).rejects.toThrow('Archive failed')
    await flushPromises()

    expect(mockToastError).toHaveBeenCalledWith('Archive failed')
    expect(store.reviews.map((review) => review.id)).toEqual(['r1'])
  })

  it('approve() refreshes the first page (merging the updated status in place) on success', async () => {
    mockedListRecentReviews.mockResolvedValueOnce({
      items: [makeReview({ id: 'r1', status: 'awaiting_approval' })],
      nextCursor: null,
    })
    const { store } = mountStore()
    await flushPromises()

    mockedApproveReview.mockResolvedValueOnce(undefined)
    mockedListRecentReviews.mockResolvedValueOnce({
      items: [makeReview({ id: 'r1', status: 'done' })],
      nextCursor: null,
    })

    await store.approve('r1')
    await flushPromises()

    expect(mockToastSuccess).toHaveBeenCalledWith('Review approved')
    expect(store.reviews).toEqual([makeReview({ id: 'r1', status: 'done' })])
    expect(mockedListRecentReviews).toHaveBeenCalledTimes(2)
  })

  it('discard() removes the review from the accumulated list locally and toasts on success', async () => {
    mockedListRecentReviews.mockResolvedValueOnce({
      items: [makeReview({ id: 'r1' })],
      nextCursor: null,
    })
    const { store } = mountStore()
    await flushPromises()

    mockedDeleteReview.mockResolvedValueOnce(undefined)

    await store.discard('r1')
    await flushPromises()

    expect(mockedDeleteReview).toHaveBeenCalledWith('r1')
    expect(mockToastSuccess).toHaveBeenCalledWith('Review discarded')
    expect(store.reviews).toEqual([])
  })

  it('publish() awaits the request, toasts success, and never touches the loaded list', async () => {
    mockedListRecentReviews.mockResolvedValueOnce({
      items: [makeReview({ id: 'r1' })],
      nextCursor: null,
    })
    const { store } = mountStore()
    await flushPromises()

    mockedPublishReview.mockResolvedValueOnce(undefined)

    await store.publish({ id: 'r1', selection: { all: true } })
    await flushPromises()

    expect(mockedPublishReview).toHaveBeenCalledWith('r1', { all: true })
    expect(mockToastSuccess).toHaveBeenCalledWith('Published to the MR')
    expect(store.reviews.map((review) => review.id)).toEqual(['r1'])
    expect(mockedListRecentReviews).toHaveBeenCalledTimes(1)
  })

  it('publish() toasts an error and rethrows when the request rejects', async () => {
    mockedListRecentReviews.mockResolvedValueOnce({ items: [], nextCursor: null })
    const { store } = mountStore()
    await flushPromises()

    mockedPublishReview.mockRejectedValueOnce(new Error('Publish failed'))

    await expect(store.publish({ id: 'r1', selection: { all: true } })).rejects.toThrow('Publish failed')
    await flushPromises()

    expect(mockToastError).toHaveBeenCalledWith('Publish failed')
  })
})
