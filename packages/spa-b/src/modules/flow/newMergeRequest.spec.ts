import { beforeEach, describe, expect, it } from 'vitest'
import {
  hasDraftText,
  isDraftDirty,
  isMergeRequestFormValid,
  isSameBranch,
  readyVoiceProfiles,
  resolveDefaultProfileId,
  resolveDefaultTargetBranch,
  loadMergeRequestSetup,
  restoreBranch,
  restoreSetupChoices,
  saveMergeRequestSetup,
} from './newMergeRequest'
import type { MergeRequestSetup } from './newMergeRequest'
import type { Profile } from '@modules/profiles/types'
import type { Provider } from '@modules/providers/types'
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

  it('is false when the source branch is just the remembered one', () => {
    expect(isDraftDirty({ sourceBranch: 'feature/x', title: '', description: '' }, 'feature/x')).toBe(false)
  })

  it('is true when the source branch differs from the remembered one', () => {
    expect(isDraftDirty({ sourceBranch: 'feature/y', title: '', description: '' }, 'feature/x')).toBe(true)
  })
})

function makeProvider(overrides: Partial<Provider> = {}): Provider {
  return {
    id: 'prov1',
    name: 'OpenAI',
    kind: 'openai',
    baseUrl: '',
    model: 'gpt-a',
    isDefault: true,
    temperature: null,
    models: ['gpt-a', 'gpt-b'],
    ...overrides,
  } as Provider
}

const SETUP: MergeRequestSetup = {
  sourceBranch: 'feature/x',
  targetBranch: 'development',
  profileId: 'p1',
  providerId: 'prov1',
  model: 'gpt-b',
}

describe('restoreBranch', () => {
  it('keeps a remembered branch that still exists', () => {
    expect(restoreBranch('feature/x', ['development', 'feature/x'])).toBe('feature/x')
  })

  it('drops a remembered branch that no longer exists', () => {
    expect(restoreBranch('feature/gone', ['development'])).toBe('')
  })
})

describe('restoreSetupChoices', () => {
  const profiles = [makeProfile({ id: 'p1' })]
  const providers = [makeProvider()]

  it('restores a still-valid profile, provider and model', () => {
    expect(restoreSetupChoices(SETUP, profiles, providers)).toEqual({
      profileId: 'p1',
      providerId: 'prov1',
      model: 'gpt-b',
    })
  })

  it('keeps an explicit "No voice profile" choice', () => {
    expect(restoreSetupChoices({ ...SETUP, profileId: '' }, profiles, providers)?.profileId).toBe('')
  })

  it('drops a profile that is gone or no longer ready', () => {
    const pending = [makeProfile({ id: 'p1', styleGuideStatus: 'pending' })]
    expect(restoreSetupChoices(SETUP, pending, providers)?.profileId).toBe('')
  })

  it('returns null for the provider pair when the provider is gone', () => {
    expect(restoreSetupChoices(SETUP, profiles, [])).toEqual({ profileId: 'p1', providerId: null, model: null })
  })

  it('falls back to the provider default when the model is gone', () => {
    const narrowed = [makeProvider({ models: ['gpt-a'] })]
    expect(restoreSetupChoices(SETUP, profiles, narrowed)?.model).toBe('')
  })
})

// This sandbox's Node/jsdom combo leaves `window.localStorage` undefined
// (same workaround as useSidebar.spec.ts): give it an in-memory store.
function installStorage(storage: Partial<Storage>) {
  Object.defineProperty(window, 'localStorage', { value: storage, configurable: true })
}

function memoryStorage(): Partial<Storage> {
  const data = new Map<string, string>()
  return {
    getItem: (key) => data.get(key) ?? null,
    setItem: (key, value) => void data.set(key, String(value)),
  }
}

describe('loadMergeRequestSetup / saveMergeRequestSetup', () => {
  beforeEach(() => installStorage(memoryStorage()))

  it('round-trips a setup per repo', () => {
    saveMergeRequestSetup('repo-rt', SETUP)
    expect(loadMergeRequestSetup('repo-rt')).toEqual(SETUP)
    expect(loadMergeRequestSetup('repo-other')).toBeNull()
  })

  it('returns null for corrupt stored data', () => {
    localStorage.setItem('4r:new-mr-setup:repo-bad', '{not json')
    expect(loadMergeRequestSetup('repo-bad')).toBeNull()
  })

  it('never throws when storage is unavailable', () => {
    const blocked = () => {
      throw new Error('blocked')
    }
    installStorage({ getItem: blocked, setItem: blocked })
    expect(loadMergeRequestSetup('repo-x')).toBeNull()
    expect(() => saveMergeRequestSetup('repo-x', SETUP)).not.toThrow()
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
