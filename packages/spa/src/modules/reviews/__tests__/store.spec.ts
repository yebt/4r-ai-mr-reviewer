import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('@shared/api/client', () => ({
  errorMessage: (e: unknown) => (e instanceof Error ? e.message : String(e)),
  api: {
    listMergeRequests: vi.fn(),
    listRepoReviews: vi.fn(),
    createReview: vi.fn(),
    getReview: vi.fn(),
    deleteReview: vi.fn(),
    retryReview: vi.fn(),
    cancelReview: vi.fn(),
    archiveReview: vi.fn(),
    unarchiveReview: vi.fn(),
    publishReview: vi.fn(),
  },
}))

import { api } from '@shared/api/client'
import type { Review, ReviewStatus } from '@shared/api/types'
import { useReviewsStore } from '@modules/reviews/store'

const mocked = api as unknown as {
  listMergeRequests: ReturnType<typeof vi.fn>
  listRepoReviews: ReturnType<typeof vi.fn>
  createReview: ReturnType<typeof vi.fn>
  getReview: ReturnType<typeof vi.fn>
  deleteReview: ReturnType<typeof vi.fn>
  retryReview: ReturnType<typeof vi.fn>
  cancelReview: ReturnType<typeof vi.fn>
  archiveReview: ReturnType<typeof vi.fn>
  unarchiveReview: ReturnType<typeof vi.fn>
  publishReview: ReturnType<typeof vi.fn>
}

const review = (id: string, status: ReviewStatus = 'pending'): Review => ({
  id,
  repoId: 'r1',
  mrIid: 7,
  contextMode: 'fast',
  status,
  phase: '',
  archived: false,
  summaryPublished: false,
  summary: '',
  recommendation: 'comment',
  score: 0,
  inputTokens: 0,
  outputTokens: 0,
  findings: [],
  createdAt: '',
  updatedAt: '',
})

describe('reviews store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('caches merge requests per repo', async () => {
    mocked.listMergeRequests.mockResolvedValue([{ iid: 7, title: 'x' }])
    const store = useReviewsStore()
    await store.fetchMergeRequests('r1')
    expect(store.mergeRequestsFor('r1')).toHaveLength(1)
    expect(store.mergeRequestsFor('other')).toHaveLength(0)
  })

  it('keeps the cache empty and records the error on failure', async () => {
    mocked.listMergeRequests.mockRejectedValue(new Error('gitlab down'))
    const store = useReviewsStore()
    await store.fetchMergeRequests('r1')
    expect(store.mrsError).toBe('gitlab down')
    expect(store.mergeRequestsFor('r1')).toHaveLength(0)
  })

  it('create prepends the new review to the repo cache', async () => {
    mocked.createReview.mockResolvedValue(review('new'))
    const store = useReviewsStore()
    store.reviewsByRepo = { r1: [review('old')] }
    await store.create('r1', 7, 'fast')
    expect(store.reviewsFor('r1').map((r) => r.id)).toEqual(['new', 'old'])
  })

  it('load shows the cached review immediately, then refreshes', async () => {
    const store = useReviewsStore()
    store.reviewsById = { '1': review('1', 'pending') }
    mocked.getReview.mockResolvedValue(review('1', 'done'))
    await store.load('1')
    expect(store.currentLoading).toBe(false)
    expect(store.current?.status).toBe('done')
  })

  it('remove purges the review from every cache', async () => {
    mocked.deleteReview.mockResolvedValue(undefined)
    const store = useReviewsStore()
    store.reviewsById = { '1': review('1'), '2': review('2') }
    store.reviewsByRepo = { r1: [review('1'), review('2')] }
    store.current = review('1')

    await store.remove('1')

    expect(mocked.deleteReview).toHaveBeenCalledWith('1')
    expect(store.reviewsById['1']).toBeUndefined()
    expect(store.reviewsFor('r1').map((r) => r.id)).toEqual(['2'])
    expect(store.current).toBeNull()
  })

  it('archive removes the review from the active list and flags it archived', async () => {
    mocked.archiveReview.mockResolvedValue({ status: 'archived' })
    const store = useReviewsStore()
    store.reviewsById = { '1': review('1'), '2': review('2') }
    store.reviewsByRepo = { r1: [review('1'), review('2')] }
    store.current = review('1')

    await store.archive('1')

    expect(mocked.archiveReview).toHaveBeenCalledWith('1')
    expect(store.reviewsFor('r1').map((r) => r.id)).toEqual(['2'])
    expect(store.reviewsById['1']?.archived).toBe(true)
    expect(store.current?.archived).toBe(true)
  })

  it('unarchive drops the review from the archived list and clears the flag', async () => {
    mocked.unarchiveReview.mockResolvedValue({ status: 'unarchived' })
    const store = useReviewsStore()
    const archived = { ...review('1'), archived: true }
    store.reviewsById = { '1': archived }
    store.archivedByRepo = { r1: [archived] }
    store.current = archived

    await store.unarchive('1')

    expect(mocked.unarchiveReview).toHaveBeenCalledWith('1')
    expect(store.archivedReviewsFor('r1')).toEqual([])
    expect(store.reviewsById['1']?.archived).toBe(false)
    expect(store.current?.archived).toBe(false)
  })

  it('cancel calls the api then refreshes current', async () => {
    mocked.cancelReview.mockResolvedValue({ status: 'cancelling' })
    mocked.getReview.mockResolvedValue(review('1', 'cancelled'))
    const store = useReviewsStore()
    await store.cancel('1')
    expect(mocked.cancelReview).toHaveBeenCalledWith('1')
    expect(mocked.getReview).toHaveBeenCalledWith('1')
    expect(store.current?.status).toBe('cancelled')
  })

  it('publish calls the api then refreshes current', async () => {
    mocked.publishReview.mockResolvedValue({ status: 'published' })
    mocked.getReview.mockResolvedValue(review('1', 'done'))
    const store = useReviewsStore()
    await store.publish('1', { all: true })
    expect(mocked.publishReview).toHaveBeenCalledWith('1', { all: true })
    expect(mocked.getReview).toHaveBeenCalledWith('1')
    expect(store.current?.id).toBe('1')
  })

  it('publish forwards includeSummary to the api', async () => {
    mocked.publishReview.mockResolvedValue({ status: 'published' })
    mocked.getReview.mockResolvedValue(review('1', 'done'))
    const store = useReviewsStore()
    await store.publish('1', { indices: [0], includeSummary: false })
    expect(mocked.publishReview).toHaveBeenCalledWith('1', { indices: [0], includeSummary: false })
  })
})
