import { describe, expect, it } from 'vitest'
import { attentionCountsByRepo } from './attention'
import type { ReviewStatus } from '@modules/reviews'
import type { RoutineRunStatus } from '@modules/runs'

function review(repoId: string, status: ReviewStatus) {
  return { repoId, status }
}

function run(repoId: string, status: RoutineRunStatus) {
  return { repoId, status }
}

describe('attentionCountsByRepo', () => {
  it('counts awaiting_approval/error reviews and awaiting_confirmation/blocked runs per repo', () => {
    const reviews = [
      review('repo1', 'awaiting_approval'),
      review('repo1', 'error'),
      review('repo1', 'done'),
      review('repo2', 'error'),
    ]
    const runs = [run('repo1', 'blocked'), run('repo1', 'running'), run('repo3', 'awaiting_confirmation')]

    const counts = attentionCountsByRepo(reviews, runs)

    expect(counts.get('repo1')).toBe(3)
    expect(counts.get('repo2')).toBe(1)
    expect(counts.get('repo3')).toBe(1)
    expect(counts.has('repo4')).toBe(false)
  })

  it('ignores statuses that do not need a human (pending/running/done/cancelled)', () => {
    const reviews = [review('repo1', 'pending'), review('repo1', 'running'), review('repo1', 'done')]
    const runs = [run('repo1', 'pending'), run('repo1', 'running'), run('repo1', 'done'), run('repo1', 'cancelled')]

    expect(attentionCountsByRepo(reviews, runs).size).toBe(0)
  })

  it('returns an empty map for empty inputs', () => {
    expect(attentionCountsByRepo([], []).size).toBe(0)
  })
})
