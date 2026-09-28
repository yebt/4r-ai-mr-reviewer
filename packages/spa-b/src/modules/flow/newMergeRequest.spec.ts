import { describe, expect, it } from 'vitest'
import {
  hasDraftText,
  isDraftDirty,
  isMergeRequestFormValid,
  isSameBranch,
  readyVoiceProfiles,
  resolveDefaultProfileId,
  resolveDefaultTargetBranch,
} from './newMergeRequest'
import type { Profile } from '@modules/profiles/types'
import type { Repo } from '@modules/repos/types'

function makeRepo(overrides: Partial<Repo> = {}): Repo {
  return {
    id: 'repo1',
    name: 'my-repo',
    url: 'https://gitlab.com/group/my-repo',
    accountId: 'acc1',
    providerId: '',
    model: '',
    defaultProfileId: '',
    webhookEnabled: false,
    webhookRequireConfirmation: false,
    webhookSecret: '',
    webhookPath: '/webhooks/repo1',
    createdAt: '2026-01-01T00:00:00.000Z',
    ...overrides,
  }
}

function makeProfile(overrides: Partial<Profile> = {}): Profile {
  return {
    id: 'profile1',
    name: 'Voice 1',
    language: 'en',
    formality: 'neutral',
    emojis: false,
    samples: [],
    styleGuide: '',
    styleGuideStatus: 'ready',
    styleGuideError: '',
    createdAt: '2026-01-01T00:00:00.000Z',
    updatedAt: '2026-01-01T00:00:00.000Z',
    ...overrides,
  }
}

describe('resolveDefaultTargetBranch', () => {
  it('prefers "development" when present', () => {
    expect(resolveDefaultTargetBranch(['main', 'development', 'feature/x'])).toBe('development')
  })

  it('falls back to "main" when there is no "development"', () => {
    expect(resolveDefaultTargetBranch(['main', 'feature/x'])).toBe('main')
  })

  it('falls back to "master" when there is no "development"/"main"', () => {
    expect(resolveDefaultTargetBranch(['master', 'feature/x'])).toBe('master')
  })

  it("falls back to the repo's first branch when none of the conventional names exist", () => {
    expect(resolveDefaultTargetBranch(['sensei', 'feature/x'])).toBe('sensei')
  })

  it('returns an empty string when the repo has no branches', () => {
    expect(resolveDefaultTargetBranch([])).toBe('')
  })
})

describe('resolveDefaultProfileId', () => {
  it("returns the repo's default profile when its style guide is ready", () => {
    const profiles = [makeProfile({ id: 'p1', styleGuideStatus: 'ready' })]
    const repo = makeRepo({ defaultProfileId: 'p1' })

    expect(resolveDefaultProfileId(repo, profiles)).toBe('p1')
  })

  it("returns '' when the repo's default profile isn't ready yet", () => {
    const profiles = [makeProfile({ id: 'p1', styleGuideStatus: 'pending' })]
    const repo = makeRepo({ defaultProfileId: 'p1' })

    expect(resolveDefaultProfileId(repo, profiles)).toBe('')
  })

  it("returns '' when the repo's default profile no longer exists", () => {
    const repo = makeRepo({ defaultProfileId: 'deleted-profile' })

    expect(resolveDefaultProfileId(repo, [])).toBe('')
  })

  it("returns '' when the repo has no default profile", () => {
    expect(resolveDefaultProfileId(makeRepo(), [makeProfile()])).toBe('')
  })
})

describe('readyVoiceProfiles', () => {
  it('keeps only profiles whose style guide is ready', () => {
    const profiles = [
      makeProfile({ id: 'p1', styleGuideStatus: 'ready' }),
      makeProfile({ id: 'p2', styleGuideStatus: 'pending' }),
      makeProfile({ id: 'p3', styleGuideStatus: 'error' }),
    ]

    expect(readyVoiceProfiles(profiles).map((p) => p.id)).toEqual(['p1'])
  })
})

describe('isSameBranch', () => {
  it('is true when both branches are picked and identical', () => {
    expect(isSameBranch({ sourceBranch: 'feature/x', targetBranch: 'feature/x' })).toBe(true)
  })

  it('is false when the branches differ', () => {
    expect(isSameBranch({ sourceBranch: 'feature/x', targetBranch: 'development' })).toBe(false)
  })

  it('is false when neither branch is picked yet', () => {
    expect(isSameBranch({ sourceBranch: '', targetBranch: '' })).toBe(false)
  })
})

describe('isMergeRequestFormValid', () => {
  it('is true with both branches, different, and a title', () => {
    expect(
      isMergeRequestFormValid({ sourceBranch: 'feature/x', targetBranch: 'development', title: 'Add x' }),
    ).toBe(true)
  })

  it('is false without a source branch', () => {
    expect(isMergeRequestFormValid({ sourceBranch: '', targetBranch: 'development', title: 'Add x' })).toBe(
      false,
    )
  })

  it('is false without a target branch', () => {
    expect(isMergeRequestFormValid({ sourceBranch: 'feature/x', targetBranch: '', title: 'Add x' })).toBe(
      false,
    )
  })

  it('is false when source and target are the same branch', () => {
    expect(
      isMergeRequestFormValid({ sourceBranch: 'feature/x', targetBranch: 'feature/x', title: 'Add x' }),
    ).toBe(false)
  })

  it('is false with a blank title', () => {
    expect(
      isMergeRequestFormValid({ sourceBranch: 'feature/x', targetBranch: 'development', title: '   ' }),
    ).toBe(false)
  })
})

describe('isDraftDirty', () => {
  it('is false with nothing filled in', () => {
    expect(isDraftDirty({ sourceBranch: '', title: '', description: '' })).toBe(false)
  })

  it('is true once a source branch is picked', () => {
    expect(isDraftDirty({ sourceBranch: 'feature/x', title: '', description: '' })).toBe(true)
  })

  it('is true once a title is typed', () => {
    expect(isDraftDirty({ sourceBranch: '', title: 'Add x', description: '' })).toBe(true)
  })

  it('is true once a description is typed', () => {
    expect(isDraftDirty({ sourceBranch: '', title: '', description: 'Adds x.' })).toBe(true)
  })
})

describe('hasDraftText', () => {
  it('is false with a blank title and description', () => {
    expect(hasDraftText({ title: '  ', description: '' })).toBe(false)
  })

  it('is true with a title', () => {
    expect(hasDraftText({ title: 'Add x', description: '' })).toBe(true)
  })

  it('is true with a description', () => {
    expect(hasDraftText({ title: '', description: 'Adds x.' })).toBe(true)
  })
})
