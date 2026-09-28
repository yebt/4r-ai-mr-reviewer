import { describe, expect, it } from 'vitest'
import { resolveDefaultModel, resolveDefaultProviderId } from './reviewLaunch'
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

function makeProvider(overrides: Partial<Provider> = {}): Provider {
  return {
    id: 'p1',
    name: 'Provider 1',
    kind: 'openai-compat',
    baseUrl: 'https://api.openai.com/v1',
    model: 'gpt-4o',
    isDefault: false,
    temperature: null,
    models: [],
    createdAt: '2026-01-01T00:00:00.000Z',
    ...overrides,
  }
}

describe('resolveDefaultProviderId', () => {
  it("prefers the repo's own provider when it is still configured", () => {
    const providers = [makeProvider({ id: 'p1' }), makeProvider({ id: 'p2', isDefault: true })]
    const repo = makeRepo({ providerId: 'p1' })

    expect(resolveDefaultProviderId(repo, providers)).toBe('p1')
  })

  it("falls back to the account-wide default provider when the repo has no provider", () => {
    const providers = [makeProvider({ id: 'p1' }), makeProvider({ id: 'p2', isDefault: true })]
    const repo = makeRepo({ providerId: '' })

    expect(resolveDefaultProviderId(repo, providers)).toBe('p2')
  })

  it("falls back to the default provider when the repo's saved provider no longer exists", () => {
    const providers = [makeProvider({ id: 'p2', isDefault: true })]
    const repo = makeRepo({ providerId: 'deleted-provider' })

    expect(resolveDefaultProviderId(repo, providers)).toBe('p2')
  })

  it('returns an empty string when there are no providers at all', () => {
    expect(resolveDefaultProviderId(makeRepo(), [])).toBe('')
  })
})

describe('resolveDefaultModel', () => {
  it("uses the repo's saved model when its own provider is selected", () => {
    const providers = [makeProvider({ id: 'p1', model: 'gpt-4o' })]
    const repo = makeRepo({ providerId: 'p1', model: 'gpt-4o-mini' })

    expect(resolveDefaultModel(repo, 'p1', providers)).toBe('gpt-4o-mini')
  })

  it("falls back to the provider's own default model when the repo has no saved model", () => {
    const providers = [makeProvider({ id: 'p1', model: 'gpt-4o' })]
    const repo = makeRepo({ providerId: 'p1', model: '' })

    expect(resolveDefaultModel(repo, 'p1', providers)).toBe('gpt-4o')
  })

  it("falls back to the selected provider's own model when it differs from the repo's provider", () => {
    const providers = [
      makeProvider({ id: 'p1', model: 'gpt-4o' }),
      makeProvider({ id: 'p2', model: 'claude-3-5-sonnet' }),
    ]
    const repo = makeRepo({ providerId: 'p1', model: 'gpt-4o-mini' })

    expect(resolveDefaultModel(repo, 'p2', providers)).toBe('claude-3-5-sonnet')
  })

  it('returns an empty string when the selected provider is unknown', () => {
    expect(resolveDefaultModel(makeRepo(), 'missing', [])).toBe('')
  })
})
