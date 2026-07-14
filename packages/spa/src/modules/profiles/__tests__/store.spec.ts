import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('@shared/api/client', () => ({
  errorMessage: (e: unknown) => (e instanceof Error ? e.message : String(e)),
  api: {
    listProfiles: vi.fn(),
    getProfile: vi.fn(),
    createProfile: vi.fn(),
    updateProfile: vi.fn(),
    deleteProfile: vi.fn(),
    redistillProfile: vi.fn(),
  },
}))

import { api } from '@shared/api/client'
import type { Profile, ProfileStyleStatus } from '@shared/api/types'
import { useProfilesStore } from '@modules/profiles/store'

const mocked = api as unknown as {
  listProfiles: ReturnType<typeof vi.fn>
  getProfile: ReturnType<typeof vi.fn>
  createProfile: ReturnType<typeof vi.fn>
  updateProfile: ReturnType<typeof vi.fn>
  deleteProfile: ReturnType<typeof vi.fn>
  redistillProfile: ReturnType<typeof vi.fn>
}

const profile = (id: string, over: Partial<Profile> = {}): Profile => ({
  id,
  name: `voice-${id}`,
  language: 'en',
  formality: 'neutral',
  emojis: false,
  samples: [],
  styleGuide: '',
  styleGuideStatus: '' as ProfileStyleStatus,
  styleGuideError: '',
  createdAt: '',
  updatedAt: '',
  ...over,
})

describe('profiles store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('fetchAll populates items', async () => {
    mocked.listProfiles.mockResolvedValue([profile('1'), profile('2')])
    const store = useProfilesStore()
    await store.fetchAll()
    expect(store.items.map((p) => p.id)).toEqual(['1', '2'])
  })

  it('create prepends the new profile', async () => {
    mocked.createProfile.mockResolvedValue(profile('2', { styleGuideStatus: 'pending' }))
    const store = useProfilesStore()
    store.items = [profile('1')]
    await store.create({
      name: 'voice-2',
      language: 'en',
      formality: '',
      emojis: false,
      samples: ['x'],
    })
    expect(store.items.map((p) => p.id)).toEqual(['2', '1'])
    expect(mocked.createProfile).toHaveBeenCalledOnce()
  })

  it('update mutates the matching profile in place', async () => {
    mocked.updateProfile.mockResolvedValue(profile('1', { name: 'renamed' }))
    const store = useProfilesStore()
    store.items = [profile('1'), profile('2')]
    await store.update('1', {
      name: 'renamed',
      language: 'en',
      formality: '',
      emojis: false,
      samples: [],
    })
    expect(store.items.find((p) => p.id === '1')?.name).toBe('renamed')
    expect(store.items.map((p) => p.id)).toEqual(['1', '2'])
  })

  it('remove purges the profile', async () => {
    mocked.deleteProfile.mockResolvedValue(undefined)
    const store = useProfilesStore()
    store.items = [profile('1'), profile('2')]
    await store.remove('1')
    expect(store.items.map((p) => p.id)).toEqual(['2'])
  })

  it('redistill calls the api then refreshes the row in place', async () => {
    mocked.redistillProfile.mockResolvedValue({ status: 'pending' })
    mocked.getProfile.mockResolvedValue(profile('1', { styleGuideStatus: 'pending' }))
    const store = useProfilesStore()
    store.items = [profile('1'), profile('2')]
    await store.redistill('1')
    expect(mocked.redistillProfile).toHaveBeenCalledWith('1')
    expect(store.items.find((p) => p.id === '1')?.styleGuideStatus).toBe('pending')
  })

  it('refreshOne updates a single profile without reordering', async () => {
    mocked.getProfile.mockResolvedValue(
      profile('2', { styleGuideStatus: 'ready', styleGuide: 'g' }),
    )
    const store = useProfilesStore()
    store.items = [profile('1'), profile('2')]
    await store.refreshOne('2')
    expect(store.items.map((p) => p.id)).toEqual(['1', '2'])
    expect(store.items.find((p) => p.id === '2')?.styleGuide).toBe('g')
  })
})
