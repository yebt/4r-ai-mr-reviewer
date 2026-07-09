import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('@shared/api/client', () => ({
  errorMessage: (e: unknown) => (e instanceof Error ? e.message : String(e)),
  api: {
    listMergeRequests: vi.fn(),
    listRepoReviews: vi.fn(),
    createReview: vi.fn(),
    getReview: vi.fn(),
    retryReview: vi.fn(),
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
  retryReview: ReturnType<typeof vi.fn>
  publishReview: ReturnType<typeof vi.fn>
}

const review = (id: string, status: ReviewStatus = 'pending'): Review => ({
  id,
  repoId: 'r1',
  mrIid: 7,
  contextMode: 'fast',
  status,
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

  it('fetchMergeRequests populates mrs', async () => {
    mocked.listMergeRequests.mockResolvedValue([{ iid: 7, title: 'x' }])
    const store = useReviewsStore()
    await store.fetchMergeRequests('r1')
    expect(store.mrs).toHaveLength(1)
  })

  it('fetchMergeRequests clears mrs on error', async () => {
    mocked.listMergeRequests.mockRejectedValue(new Error('gitlab down'))
    const store = useReviewsStore()
    await store.fetchMergeRequests('r1')
    expect(store.mrsError).toBe('gitlab down')
    expect(store.mrs).toHaveLength(0)
  })

  it('create prepends the new review', async () => {
    mocked.createReview.mockResolvedValue(review('new'))
    const store = useReviewsStore()
    store.list = [review('old')]
    await store.create('r1', 7, 'fast')
    expect(store.list.map((r) => r.id)).toEqual(['new', 'old'])
  })

  it('load sets current', async () => {
    mocked.getReview.mockResolvedValue(review('1', 'done'))
    const store = useReviewsStore()
    await store.load('1')
    expect(store.current?.status).toBe('done')
  })

  it('publish calls api then refreshes current', async () => {
    mocked.publishReview.mockResolvedValue({ status: 'published' })
    mocked.getReview.mockResolvedValue(review('1', 'done'))
    const store = useReviewsStore()
    await store.publish('1', { all: true })
    expect(mocked.publishReview).toHaveBeenCalledWith('1', { all: true })
    expect(mocked.getReview).toHaveBeenCalledWith('1')
    expect(store.current?.id).toBe('1')
  })
})
