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
    humanizeFinding: vi.fn(),
    humanizeSummary: vi.fn(),
    getHumanizations: vi.fn(),
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
  humanizeFinding: ReturnType<typeof vi.fn>
  humanizeSummary: ReturnType<typeof vi.fn>
  getHumanizations: ReturnType<typeof vi.fn>
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
    // load() always rehydrates humanize tabs; default to an empty server payload
    // so tests that do not care about it are unaffected.
    mocked.getHumanizations.mockResolvedValue({ summary: [], findings: {} })
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

  it('load rehydrates humanized tabs from the server', async () => {
    mocked.getReview.mockResolvedValue(review('1', 'done'))
    mocked.getHumanizations.mockResolvedValue({
      summary: [{ summary: 'nicer' }],
      findings: { 0: [{ issue: 'kind', why: 'w', fix: 'f' }] },
    })
    const store = useReviewsStore()

    await store.load('1')

    expect(mocked.getHumanizations).toHaveBeenCalledWith('1')
    expect(store.summaryTabs('1')).toEqual([{ summary: 'nicer' }])
    expect(store.findingTabs('1', 0)).toEqual([{ issue: 'kind', why: 'w', fix: 'f' }])
    // Selection stays on Original after rehydration.
    expect(store.selectedSummaryTab('1')).toBe(-1)
    expect(store.selectedFindingTab('1', 0)).toBe(-1)
  })

  it('load keeps the loaded review even if humanization rehydration fails', async () => {
    mocked.getReview.mockResolvedValue(review('1', 'done'))
    mocked.getHumanizations.mockRejectedValue(new Error('humanizations down'))
    const store = useReviewsStore()

    await store.load('1')

    expect(store.current?.status).toBe('done')
    expect(store.currentError).toBeNull()
    expect(store.summaryTabs('1')).toEqual([])
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

  it('publish forwards summaryOverride and findingOverrides to the api', async () => {
    mocked.publishReview.mockResolvedValue({ status: 'published' })
    mocked.getReview.mockResolvedValue(review('1', 'done'))
    const store = useReviewsStore()
    await store.publish('1', {
      all: true,
      includeSummary: true,
      summaryOverride: 'nicer summary',
      findingOverrides: [{ index: 0, text: 'kinder finding' }],
    })
    expect(mocked.publishReview).toHaveBeenCalledWith('1', {
      all: true,
      includeSummary: true,
      summaryOverride: 'nicer summary',
      findingOverrides: [{ index: 0, text: 'kinder finding' }],
    })
  })

  it('humanizeFinding appends a tab, auto-selects it, and clears the flag', async () => {
    mocked.humanizeFinding.mockResolvedValue({ issue: 'kind', why: 'w', fix: 'f' })
    const store = useReviewsStore()

    await store.humanizeFinding('1', 'p1', 0)

    expect(mocked.humanizeFinding).toHaveBeenCalledWith('1', 'p1', 0)
    expect(store.findingTabs('1', 0)).toEqual([{ issue: 'kind', why: 'w', fix: 'f' }])
    expect(store.selectedFindingTab('1', 0)).toBe(0)
    expect(store.findingHumanizing('1', 0)).toBe(false)
  })

  it('humanizeFinding accumulates tabs and auto-selects the newest', async () => {
    mocked.humanizeFinding
      .mockResolvedValueOnce({ issue: 'v1', why: '', fix: '' })
      .mockResolvedValueOnce({ issue: 'v2', why: '', fix: '' })
    const store = useReviewsStore()

    await store.humanizeFinding('1', 'p1', 0)
    await store.humanizeFinding('1', 'p1', 0)

    expect(store.findingTabs('1', 0).map((t) => t.issue)).toEqual(['v1', 'v2'])
    expect(store.selectedFindingTab('1', 0)).toBe(1)
  })

  it('humanizeFinding sets the flag while in flight and rethrows on error', async () => {
    let reject: (e: unknown) => void = () => {}
    mocked.humanizeFinding.mockReturnValue(
      new Promise((_res, rej) => {
        reject = rej
      }),
    )
    const store = useReviewsStore()

    const pending = store.humanizeFinding('1', 'p1', 0)
    expect(store.findingHumanizing('1', 0)).toBe(true)

    reject(new Error('style guide not ready'))
    await expect(pending).rejects.toThrow('style guide not ready')
    expect(store.findingHumanizing('1', 0)).toBe(false)
    expect(store.findingTabs('1', 0)).toEqual([])
  })

  it('humanizeSummary appends a tab, auto-selects it, and clears the flag', async () => {
    mocked.humanizeSummary.mockResolvedValue({ summary: 'nicer summary' })
    const store = useReviewsStore()

    await store.humanizeSummary('1', 'p1')

    expect(mocked.humanizeSummary).toHaveBeenCalledWith('1', 'p1')
    expect(store.summaryTabs('1')).toEqual([{ summary: 'nicer summary' }])
    expect(store.selectedSummaryTab('1')).toBe(0)
    expect(store.summaryHumanizing('1')).toBe(false)
  })

  it('humanizeSummary clears the flag and rethrows on error', async () => {
    mocked.humanizeSummary.mockRejectedValue(new Error('not ready'))
    const store = useReviewsStore()

    await expect(store.humanizeSummary('1', 'p1')).rejects.toThrow('not ready')
    expect(store.summaryHumanizing('1')).toBe(false)
    expect(store.summaryTabs('1')).toEqual([])
  })

  it('selectFindingTab and selectSummaryTab update the selection', () => {
    const store = useReviewsStore()
    store.selectFindingTab('1', 0, 2)
    store.selectSummaryTab('1', 1)
    expect(store.selectedFindingTab('1', 0)).toBe(2)
    expect(store.selectedSummaryTab('1')).toBe(1)
  })

  it('defaults every card selection to Original (-1)', () => {
    const store = useReviewsStore()
    expect(store.selectedFindingTab('1', 0)).toBe(-1)
    expect(store.selectedSummaryTab('1')).toBe(-1)
  })

  it('publish forwards the assembled findingOverrides for a humanized finding tab', async () => {
    mocked.publishReview.mockResolvedValue({ status: 'published' })
    mocked.getReview.mockResolvedValue(review('1', 'done'))
    const store = useReviewsStore()

    // Mirror what FindingCard assembles for a selected humanized tab.
    await store.publish('1', {
      indices: [0],
      includeSummary: false,
      findingOverrides: [{ index: 0, text: '**[R1 Risk · HIGH]** kind\n\n' }],
    })

    expect(mocked.publishReview).toHaveBeenCalledWith('1', {
      indices: [0],
      includeSummary: false,
      findingOverrides: [{ index: 0, text: '**[R1 Risk · HIGH]** kind\n\n' }],
    })
  })

  it('markPublished records the published tab per card and reads it back', () => {
    const store = useReviewsStore()

    store.markPublished('1', 'summary', 2)
    store.markPublished('1', 0, -1)
    store.markPublished('1', 3, 1)

    expect(store.publishedSummaryTab('1')).toBe(2)
    expect(store.publishedFindingTab('1', 0)).toBe(-1)
    expect(store.publishedFindingTab('1', 3)).toBe(1)
  })

  it('published-tab helpers return null when unset and lazy-init does not throw', () => {
    const store = useReviewsStore()
    expect(store.publishedSummaryTab('unknown')).toBeNull()
    expect(store.publishedFindingTab('unknown', 9)).toBeNull()

    // Marking one card must not fabricate entries for the others.
    store.markPublished('1', 5, 0)
    expect(store.publishedFindingTab('1', 6)).toBeNull()
    expect(store.publishedSummaryTab('1')).toBeNull()
  })

  it('publish forwards no overrides for an Original finding tab', async () => {
    mocked.publishReview.mockResolvedValue({ status: 'published' })
    mocked.getReview.mockResolvedValue(review('1', 'done'))
    const store = useReviewsStore()

    await store.publish('1', { indices: [0], includeSummary: false })

    expect(mocked.publishReview).toHaveBeenCalledWith('1', { indices: [0], includeSummary: false })
  })
})
