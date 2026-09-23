import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { PiniaColada } from '@pinia/colada'
import { defineComponent } from 'vue'
import * as reposApi from '@modules/repos/api'
import * as reviewsApi from './api'
import { fetchAllReviews, useReviewsStore } from './store'
import type { Repo } from '@modules/repos/types'
import type { Review } from './types'

vi.mock('@modules/repos/api')
vi.mock('./api')

// `vi.mock` factories are hoisted above regular top-level statements, so the
// spies they close over must be created via `vi.hoisted` (see providers'
// store.spec.ts for the same pattern).
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

const mockedListRepos = vi.mocked(reposApi.listRepos)
const mockedListRepoReviews = vi.mocked(reviewsApi.listRepoReviews)
const mockedRetryReview = vi.mocked(reviewsApi.retryReview)
const mockedArchiveReview = vi.mocked(reviewsApi.archiveReview)

function makeRepo(overrides: Partial<Repo> = {}): Repo {
  return {
    id: 'repo-1',
    name: 'repo-one',
    url: 'https://gitlab.example.com/repo-one',
    accountId: 'acc-1',
    providerId: 'prov-1',
    model: '',
    defaultProfileId: '',
    webhookEnabled: false,
    webhookRequireConfirmation: false,
    webhookSecret: '',
    webhookPath: '',
    createdAt: '2026-01-01T00:00:00.000Z',
    ...overrides,
  }
}

function makeReview(overrides: Partial<Review> = {}): Review {
  return {
    id: 'r1',
    repoId: 'repo-1',
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

/** Mounts a throwaway component so `useReviewsStore()` runs inside a real
 * Pinia + Pinia Colada injection context (same pattern as providers'
 * `store.spec.ts`). */
function mountStore() {
  let store!: ReturnType<typeof useReviewsStore>
  const Harness = defineComponent({
    setup() {
      store = useReviewsStore()
      return () => null
    },
  })
  const wrapper = mount(Harness, {
    global: { plugins: [createPinia(), PiniaColada] },
  })
  return { wrapper, store }
}

describe('fetchAllReviews (fan-out)', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('fetches every repo, then every repo\'s reviews, flattening them with repoName attached', async () => {
    mockedListRepos.mockResolvedValueOnce([makeRepo({ id: 'repo-1', name: 'alpha' }), makeRepo({ id: 'repo-2', name: 'beta' })])
    mockedListRepoReviews.mockImplementation((repoId) =>
      Promise.resolve(repoId === 'repo-1' ? [makeReview({ id: 'r1', repoId: 'repo-1' })] : [makeReview({ id: 'r2', repoId: 'repo-2' })]),
    )

    const result = await fetchAllReviews(false)

    expect(mockedListRepos).toHaveBeenCalledTimes(1)
    expect(mockedListRepoReviews).toHaveBeenCalledWith('repo-1', false)
    expect(mockedListRepoReviews).toHaveBeenCalledWith('repo-2', false)
    expect(result.reviews).toEqual([
      { ...makeReview({ id: 'r1', repoId: 'repo-1' }), repoName: 'alpha' },
      { ...makeReview({ id: 'r2', repoId: 'repo-2' }), repoName: 'beta' },
    ])
    expect(result.failedRepoNames).toEqual([])
  })

  it('returns an empty list when there are no repos, without calling listRepoReviews', async () => {
    mockedListRepos.mockResolvedValueOnce([])

    const result = await fetchAllReviews(false)

    expect(result.reviews).toEqual([])
    expect(result.failedRepoNames).toEqual([])
    expect(mockedListRepoReviews).not.toHaveBeenCalled()
  })

  it('forwards the archived flag to every per-repo call', async () => {
    mockedListRepos.mockResolvedValueOnce([makeRepo({ id: 'repo-1' })])
    mockedListRepoReviews.mockResolvedValueOnce([])

    await fetchAllReviews(true)

    expect(mockedListRepoReviews).toHaveBeenCalledWith('repo-1', true)
  })

  it('merges the fulfilled repos and reports the rejected ones by name, instead of failing closed', async () => {
    mockedListRepos.mockResolvedValueOnce([
      makeRepo({ id: 'repo-1', name: 'alpha' }),
      makeRepo({ id: 'repo-2', name: 'beta' }),
      makeRepo({ id: 'repo-3', name: 'gamma' }),
    ])
    mockedListRepoReviews.mockImplementation((repoId) => {
      if (repoId === 'repo-2') return Promise.reject(new Error('unreachable'))
      return Promise.resolve([makeReview({ id: `r-${repoId}`, repoId })])
    })

    const result = await fetchAllReviews(false)

    expect(result.reviews).toEqual([
      { ...makeReview({ id: 'r-repo-1', repoId: 'repo-1' }), repoName: 'alpha' },
      { ...makeReview({ id: 'r-repo-3', repoId: 'repo-3' }), repoName: 'gamma' },
    ])
    expect(result.failedRepoNames).toEqual(['beta'])
  })
})

describe('useReviewsStore (@pinia/colada)', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('the reviews query maps the fan-out result into store.reviews', async () => {
    mockedListRepos.mockResolvedValueOnce([makeRepo({ id: 'repo-1', name: 'alpha' })])
    mockedListRepoReviews.mockResolvedValueOnce([makeReview({ id: 'r1' })])

    const { store } = mountStore()
    await flushPromises()

    expect(store.reviews).toEqual([{ ...makeReview({ id: 'r1' }), repoName: 'alpha' }])
    expect(store.isLoading).toBe(false)
    expect(store.state.status).toBe('success')
  })

  it('retry() awaits the request, toasts success, and invalidates the reviews query (refetch)', async () => {
    mockedListRepos.mockResolvedValue([makeRepo({ id: 'repo-1', name: 'alpha' })])
    mockedListRepoReviews.mockResolvedValueOnce([makeReview({ id: 'r1', status: 'error' })])

    const { store } = mountStore()
    await flushPromises()
    expect(mockedListRepoReviews).toHaveBeenCalledTimes(1)

    mockedRetryReview.mockResolvedValueOnce(makeReview({ id: 'r2', status: 'pending' }))
    mockedListRepoReviews.mockResolvedValueOnce([makeReview({ id: 'r1', status: 'error' }), makeReview({ id: 'r2', status: 'pending' })])

    await store.retry('r1')
    await flushPromises()

    expect(mockedRetryReview).toHaveBeenCalledWith('r1')
    expect(mockToastSuccess).toHaveBeenCalledWith('Review retried')
    // Invalidation on settle triggers a refetch of the ['reviews', {archived}] query.
    expect(mockedListRepoReviews.mock.calls.length).toBeGreaterThanOrEqual(2)
  })

  it('archive() toasts an error and still invalidates (refetches) when the request rejects', async () => {
    mockedListRepos.mockResolvedValue([makeRepo({ id: 'repo-1' })])
    mockedListRepoReviews.mockResolvedValueOnce([makeReview({ id: 'r1' })])

    const { store } = mountStore()
    await flushPromises()
    expect(mockedListRepoReviews).toHaveBeenCalledTimes(1)

    mockedArchiveReview.mockRejectedValueOnce(new Error('Archive failed'))
    mockedListRepoReviews.mockResolvedValueOnce([makeReview({ id: 'r1' })])

    await expect(store.archive('r1')).rejects.toThrow('Archive failed')
    await flushPromises()

    expect(mockToastError).toHaveBeenCalledWith('Archive failed')
    expect(mockedListRepoReviews.mock.calls.length).toBeGreaterThanOrEqual(2)
  })

  it('flipping store.archived changes the query key, lazily fetching the archived fan-out', async () => {
    mockedListRepos.mockResolvedValue([makeRepo({ id: 'repo-1' })])
    mockedListRepoReviews.mockResolvedValueOnce([makeReview({ id: 'r1', archived: false })])

    const { store } = mountStore()
    await flushPromises()
    expect(mockedListRepoReviews).toHaveBeenCalledWith('repo-1', false)

    mockedListRepoReviews.mockResolvedValueOnce([makeReview({ id: 'r1', archived: true })])
    store.archived = true
    await flushPromises()

    expect(mockedListRepoReviews).toHaveBeenCalledWith('repo-1', true)
  })

  it('store.reviews still holds the successful repos and store.failedRepoCount reports the rejected one', async () => {
    mockedListRepos.mockResolvedValueOnce([makeRepo({ id: 'repo-1', name: 'alpha' }), makeRepo({ id: 'repo-2', name: 'beta' })])
    mockedListRepoReviews.mockImplementation((repoId) => {
      if (repoId === 'repo-2') return Promise.reject(new Error('unreachable'))
      return Promise.resolve([makeReview({ id: 'r1', repoId: 'repo-1' })])
    })

    const { store } = mountStore()
    await flushPromises()

    expect(store.reviews).toEqual([{ ...makeReview({ id: 'r1', repoId: 'repo-1' }), repoName: 'alpha' }])
    expect(store.failedRepoCount).toBe(1)
    expect(store.failedRepoNames).toEqual(['beta'])
    expect(store.state.status).toBe('success')
  })
})
