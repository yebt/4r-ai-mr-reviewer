import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('@shared/api/client', () => ({
  errorMessage: (e: unknown) => (e instanceof Error ? e.message : String(e)),
  api: {
    listRepos: vi.fn(),
    createRepo: vi.fn(),
    assignRepo: vi.fn(),
    deleteRepo: vi.fn(),
  },
}))

import { api } from '@shared/api/client'
import { useReposStore } from '@modules/repos/store'

const mocked = api as unknown as {
  listRepos: ReturnType<typeof vi.fn>
  createRepo: ReturnType<typeof vi.fn>
  assignRepo: ReturnType<typeof vi.fn>
  deleteRepo: ReturnType<typeof vi.fn>
}

const repo = (id: string, over: Partial<Record<string, unknown>> = {}) => ({
  id,
  name: `repo-${id}`,
  url: 'https://gitlab.com/g/p',
  accountId: 'acc',
  providerId: '',
  model: '',
  createdAt: '',
  ...over,
})

describe('repos store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('fetchAll populates items', async () => {
    mocked.listRepos.mockResolvedValue([repo('1')])
    const store = useReposStore()
    await store.fetchAll()
    expect(store.items).toHaveLength(1)
    expect(store.error).toBeNull()
  })

  it('add appends the created repo', async () => {
    mocked.createRepo.mockResolvedValue(repo('2'))
    const store = useReposStore()
    await store.add({ name: 'repo-2', url: 'u', accountId: 'acc', providerId: '', model: '' })
    expect(store.items.map((r) => r.id)).toContain('2')
  })

  it('assign replaces the repo in place', async () => {
    mocked.assignRepo.mockResolvedValue(repo('1', { providerId: 'p9', model: 'm9' }))
    const store = useReposStore()
    store.items = [repo('1')]
    await store.assign('1', { providerId: 'p9', model: 'm9' })
    expect(store.items[0]!.providerId).toBe('p9')
    expect(store.items[0]!.model).toBe('m9')
  })

  it('remove drops the repo', async () => {
    mocked.deleteRepo.mockResolvedValue(undefined)
    const store = useReposStore()
    store.items = [repo('1'), repo('2')]
    await store.remove('1')
    expect(store.items.map((r) => r.id)).toEqual(['2'])
  })
})
