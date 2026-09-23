import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { PiniaColada } from '@pinia/colada'
import { defineComponent } from 'vue'
import * as reposApi from './api'
import {
  makeOptimisticRepo,
  toRepoPatch,
  useReposStore,
  withPatchedRepo,
  withoutRepo,
} from './store'
import type { Repo } from './types'

vi.mock('./api')

// `vi.mock` factories are hoisted above regular top-level statements, so the
// spies they close over must be created via `vi.hoisted` — a plain `const`
// declared below would still be in its temporal dead zone when the factory
// first runs.
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
const mockedCreateRepo = vi.mocked(reposApi.createRepo)
const mockedAssignRepo = vi.mocked(reposApi.assignRepo)
const mockedDeleteRepo = vi.mocked(reposApi.deleteRepo)
const mockedSetRepoWebhook = vi.mocked(reposApi.setRepoWebhook)

function makeRepo(overrides: Partial<Repo> = {}): Repo {
  return {
    id: 'r1',
    name: 'my-repo',
    url: 'https://gitlab.com/group/my-repo',
    accountId: 'acc1',
    providerId: '',
    model: '',
    defaultProfileId: '',
    webhookEnabled: false,
    webhookRequireConfirmation: false,
    webhookSecret: '',
    webhookPath: '/webhooks/gitlab/r1',
    createdAt: '2026-01-01T00:00:00.000Z',
    ...overrides,
  }
}

/** A promise plus its resolve/reject, so a test can assert the optimistic
 * cache state before the underlying request settles. */
function deferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (reason: unknown) => void
  const promise = new Promise<T>((res, rej) => {
    resolve = res
    reject = rej
  })
  return { promise, resolve, reject }
}

/**
 * Mounts a throwaway component that just instantiates `useReposStore()`
 * under a real Pinia + Pinia Colada app context — required because
 * `useQuery`/`useMutation`/`useQueryCache` need an active injection context.
 */
function mountStore() {
  let store!: ReturnType<typeof useReposStore>
  const Harness = defineComponent({
    setup() {
      store = useReposStore()
      return () => null
    },
  })
  const wrapper = mount(Harness, {
    global: { plugins: [createPinia(), PiniaColada] },
  })
  return { wrapper, store }
}

describe('pure optimistic-patch helpers', () => {
  it('withoutRepo drops the matching row only', () => {
    const existing = [makeRepo({ id: 'r1' }), makeRepo({ id: 'r2' })]
    expect(withoutRepo(existing, 'r1')).toEqual([existing[1]])
  })

  it('withPatchedRepo merges a partial patch onto the matching row only', () => {
    const existing = [makeRepo({ id: 'r1', name: 'Old' }), makeRepo({ id: 'r2', name: 'Other' })]
    const patched = withPatchedRepo(existing, 'r1', { name: 'New' })
    expect(patched[0]).toEqual({ ...existing[0], name: 'New' })
    expect(patched[1]).toEqual(existing[1])
  })

  it('toRepoPatch maps the assign payload onto the repo fields it can update (no name/url)', () => {
    const patch = toRepoPatch({
      accountId: 'acc2',
      providerId: 'p1',
      model: 'gpt-4o',
      profileId: 'prof1',
    })
    expect(patch).toEqual({
      accountId: 'acc2',
      providerId: 'p1',
      model: 'gpt-4o',
      defaultProfileId: 'prof1',
    })
  })

  it('makeOptimisticRepo builds a temp-id row carrying the create payload, webhook unset', () => {
    const optimistic = makeOptimisticRepo({
      name: 'my-repo',
      url: 'https://gitlab.com/group/my-repo',
      accountId: 'acc1',
      providerId: '',
      model: '',
      profileId: '',
    })
    expect(optimistic.id).toMatch(/^temp-/)
    expect(optimistic.name).toBe('my-repo')
    expect(optimistic.webhookEnabled).toBe(false)
    expect(optimistic.webhookSecret).toBe('')
  })
})

describe('useReposStore (@pinia/colada)', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('the repos query maps the mocked listRepos() result into store.repos', async () => {
    const repos = [makeRepo()]
    mockedListRepos.mockResolvedValueOnce(repos)

    const { store } = mountStore()
    await flushPromises()

    expect(mockedListRepos).toHaveBeenCalledTimes(1)
    expect(store.repos).toEqual(repos)
    expect(store.isLoading).toBe(false)
    expect(store.reposState.status).toBe('success')
  })

  it('createRepo appends an optimistic temp row, then reconciles with the server id on success', async () => {
    mockedListRepos.mockResolvedValueOnce([])
    const { store } = mountStore()
    await flushPromises()

    const created = makeRepo({ id: 'server-1' })
    const { promise, resolve } = deferred<Repo>()
    mockedCreateRepo.mockReturnValueOnce(promise)
    mockedListRepos.mockResolvedValueOnce([created])

    const call = store.createRepo({
      name: created.name,
      url: created.url,
      accountId: created.accountId,
      providerId: '',
      model: '',
      profileId: '',
    })
    await Promise.resolve()
    await Promise.resolve()

    expect(store.repos).toHaveLength(1)
    expect(store.repos[0]!.id).toMatch(/^temp-/)

    resolve(created)
    await call
    await flushPromises()

    expect(store.repos.some((repo) => repo.id === 'server-1')).toBe(true)
    expect(mockToastSuccess).toHaveBeenCalledWith('Repository added')
  })

  it('assignRepo (the factory update slot) optimistically patches the row before the request resolves', async () => {
    mockedListRepos.mockResolvedValueOnce([makeRepo({ id: 'r1', providerId: '', model: '' })])
    const { store } = mountStore()
    await flushPromises()

    const updated = makeRepo({ id: 'r1', providerId: 'p1', model: 'gpt-4o' })
    const { promise, resolve } = deferred<Repo>()
    mockedAssignRepo.mockReturnValueOnce(promise)
    mockedListRepos.mockResolvedValueOnce([updated])

    const call = store.assignRepo('r1', {
      accountId: 'acc1',
      providerId: 'p1',
      model: 'gpt-4o',
      profileId: '',
    })
    await Promise.resolve()
    await Promise.resolve()

    expect(store.repos[0]!.providerId).toBe('p1')
    expect(store.repos[0]!.model).toBe('gpt-4o')

    resolve(updated)
    await call
    await flushPromises()

    expect(mockedAssignRepo).toHaveBeenCalledWith('r1', {
      accountId: 'acc1',
      providerId: 'p1',
      model: 'gpt-4o',
      profileId: '',
    })
    expect(mockToastSuccess).toHaveBeenCalledWith('Repository reassigned')
  })

  it('removeRepo optimistically drops the row and rolls back + toasts on error', async () => {
    mockedListRepos.mockResolvedValueOnce([makeRepo({ id: 'r1' }), makeRepo({ id: 'r2' })])
    const { store } = mountStore()
    await flushPromises()

    const { promise, reject } = deferred<void>()
    mockedDeleteRepo.mockReturnValueOnce(promise)
    mockedListRepos.mockResolvedValueOnce([makeRepo({ id: 'r1' }), makeRepo({ id: 'r2' })])

    const call = store.removeRepo('r1').catch(() => undefined)
    await Promise.resolve()
    await Promise.resolve()

    expect(store.repos.map((repo) => repo.id)).toEqual(['r2'])

    reject(new Error('Delete failed'))
    await call
    await flushPromises()

    // Rolled back to the pre-mutation snapshot.
    expect(store.repos.map((repo) => repo.id).sort()).toEqual(['r1', 'r2'])
    expect(mockToastError).toHaveBeenCalledWith('Delete failed')
  })

  it('setWebhook patches the row from the server response (no optimistic guess at the rotated secret)', async () => {
    mockedListRepos.mockResolvedValueOnce([makeRepo({ id: 'r1', webhookEnabled: false })])
    const { store } = mountStore()
    await flushPromises()

    const updated = makeRepo({
      id: 'r1',
      webhookEnabled: true,
      webhookSecret: 'shhh',
    })
    mockedSetRepoWebhook.mockResolvedValueOnce(updated)

    await store.setWebhook('r1', { enabled: true, requireConfirmation: false })
    await flushPromises()

    expect(mockedSetRepoWebhook).toHaveBeenCalledWith('r1', { enabled: true, requireConfirmation: false })
    expect(store.repos[0]).toEqual(updated)
    expect(mockToastSuccess).toHaveBeenCalledWith('Webhook updated')
  })
})
