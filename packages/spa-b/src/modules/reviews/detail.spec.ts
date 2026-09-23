import { describe, expect, it, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { PiniaColada } from '@pinia/colada'
import { defineComponent } from 'vue'
import * as reviewsApi from './api'
import { isReviewActive, useReviewDetail } from './detail'
import type { Review, ReviewStatus } from './types'

vi.mock('./api')

// Same rationale as `modules/runs/store.spec.ts`: exercise the polling gate
// as the pure `isReviewActive` function rather than through real timers, so
// a leftover interval never calls the mocked API again after a test ends.
vi.mock('@vueuse/core', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@vueuse/core')>()
  return {
    ...actual,
    useIntervalFn: () => ({
      pause: vi.fn(),
      resume: vi.fn(),
      isActive: { value: false },
    }),
  }
})

const mockedGetReview = vi.mocked(reviewsApi.getReview)

function makeReview(overrides: Partial<Review> = {}): Review {
  return {
    id: 'r1',
    repoId: 'repo-1',
    mrIid: 1,
    contextMode: 'fast',
    status: 'done',
    phase: 'done',
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

function mountDetail(id: string) {
  let detail!: ReturnType<typeof useReviewDetail>
  const Harness = defineComponent({
    setup() {
      detail = useReviewDetail(id)
      return () => null
    },
  })
  const wrapper = mount(Harness, {
    global: { plugins: [createPinia(), PiniaColada] },
  })
  return { wrapper, detail }
}

describe('isReviewActive (pure)', () => {
  it.each([
    ['pending', true],
    ['running', true],
    ['done', false],
    ['error', false],
    ['cancelled', false],
    ['awaiting_approval', false],
  ] as [ReviewStatus, boolean][])('status "%s" -> %s', (status, expected) => {
    expect(isReviewActive(status)).toBe(expected)
  })
})

describe('useReviewDetail', () => {
  it('fetches the review by id and maps it into detail.review', async () => {
    mockedGetReview.mockResolvedValueOnce(makeReview({ id: 'r1', summary: 'looks fine' }))

    const { detail } = mountDetail('r1')
    await flushPromises()

    expect(mockedGetReview).toHaveBeenCalledWith('r1')
    expect(detail.review.value).toEqual(makeReview({ id: 'r1', summary: 'looks fine' }))
    expect(detail.state.value.status).toBe('success')
  })

  it('surfaces a query error without throwing', async () => {
    mockedGetReview.mockRejectedValueOnce(new Error('not found'))

    const { detail } = mountDetail('missing')
    await flushPromises()

    expect(detail.state.value.status).toBe('error')
    expect(detail.error.value).toBeInstanceOf(Error)
  })
})
