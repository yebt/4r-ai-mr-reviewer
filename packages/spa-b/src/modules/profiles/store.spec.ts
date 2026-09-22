import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { PiniaColada } from '@pinia/colada'
import { defineComponent } from 'vue'
import * as profilesApi from './api'
import {
  makeOptimisticProfile,
  resolveErrorMessage,
  toProfilePatch,
  useProfilesStore,
  withOptimisticCreate,
  withPatchedProfile,
  withoutProfile,
} from './store'
import type { Profile } from './types'

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

const mockedListProfiles = vi.mocked(profilesApi.listProfiles)
const mockedCreateProfile = vi.mocked(profilesApi.createProfile)
const mockedUpdateProfile = vi.mocked(profilesApi.updateProfile)
const mockedDeleteProfile = vi.mocked(profilesApi.deleteProfile)
const mockedRedistillProfile = vi.mocked(profilesApi.redistillProfile)

function makeProfile(overrides: Partial<Profile> = {}): Profile {
  return {
    id: 'p1',
    name: 'Edu',
    language: 'es',
    formality: 'neutral',
    emojis: false,
    samples: ['sample one'],
    styleGuide: '',
    styleGuideStatus: 'ready',
    styleGuideError: '',
    createdAt: '2026-01-01T00:00:00.000Z',
    updatedAt: '2026-01-01T00:00:00.000Z',
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
 * Mounts a throwaway component that just instantiates `useProfilesStore()`
 * under a real Pinia + Pinia Colada app context — required because
 * `useQuery`/`useMutation`/`useQueryCache` need an active injection context,
 * exactly like any other Vue composable.
 */
function mountStore() {
  let store!: ReturnType<typeof useProfilesStore>
  const Harness = defineComponent({
    setup() {
      store = useProfilesStore()
      return () => null
    },
  })
  const wrapper = mount(Harness, {
    global: { plugins: [createPinia(), PiniaColada] },
  })
  return { wrapper, store }
}

describe('pure optimistic-patch helpers', () => {
  it('withOptimisticCreate appends a temp row', () => {
    const existing = [makeProfile({ id: 'p1' })]
    const optimistic = makeProfile({ id: 'temp-1' })

    expect(withOptimisticCreate(existing, optimistic)).toEqual([existing[0], optimistic])
  })

  it('withoutProfile drops the matching row only', () => {
    const existing = [makeProfile({ id: 'p1' }), makeProfile({ id: 'p2' })]
    expect(withoutProfile(existing, 'p1')).toEqual([existing[1]])
  })

  it('withPatchedProfile merges a partial patch onto the matching row only', () => {
    const existing = [makeProfile({ id: 'p1', name: 'Old' }), makeProfile({ id: 'p2', name: 'Other' })]
    const patched = withPatchedProfile(existing, 'p1', { name: 'New' })
    expect(patched[0]).toEqual({ ...existing[0], name: 'New' })
    expect(patched[1]).toEqual(existing[1])
  })

  it('toProfilePatch maps only the editable fields', () => {
    const patch = toProfilePatch({
      name: 'N',
      language: 'en',
      formality: 'casual',
      emojis: true,
      samples: ['s1'],
    })
    expect(patch).toEqual({ name: 'N', language: 'en', formality: 'casual', emojis: true, samples: ['s1'] })
  })

  it('makeOptimisticProfile builds a temp-id row carrying the create payload with a pending style guide', () => {
    const optimistic = makeOptimisticProfile({
      name: 'N',
      language: 'es',
      formality: 'neutral',
      emojis: false,
      samples: [],
    })
    expect(optimistic.id).toMatch(/^temp-/)
    expect(optimistic.name).toBe('N')
    expect(optimistic.styleGuideStatus).toBe('pending')
  })

  it('resolveErrorMessage prefers the Error message and falls back otherwise', () => {
    expect(resolveErrorMessage(new Error('boom'), 'fallback')).toBe('boom')
    expect(resolveErrorMessage('not an error', 'fallback')).toBe('fallback')
  })
})

describe('useProfilesStore (@pinia/colada)', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('the profiles query maps the mocked listProfiles() result into store.profiles', async () => {
    const profiles = [makeProfile()]
    mockedListProfiles.mockResolvedValueOnce(profiles)

    const { store } = mountStore()
    await flushPromises()

    expect(mockedListProfiles).toHaveBeenCalledTimes(1)
    expect(store.profiles).toEqual(profiles)
    expect(store.isLoading).toBe(false)
    expect(store.profilesState.status).toBe('success')
  })

  it('createProfile appends an optimistic temp row, then reconciles with the server id on success', async () => {
    mockedListProfiles.mockResolvedValueOnce([])
    const { store } = mountStore()
    await flushPromises()

    const created = makeProfile({ id: 'server-1' })
    const { promise, resolve } = deferred<Profile>()
    mockedCreateProfile.mockReturnValueOnce(promise)
    mockedListProfiles.mockResolvedValueOnce([created])

    const call = store.createProfile({
      name: created.name,
      language: created.language,
      formality: created.formality,
      emojis: created.emojis,
      samples: created.samples,
    })
    await Promise.resolve()
    await Promise.resolve()

    expect(store.profiles).toHaveLength(1)
    expect(store.profiles[0]!.id).toMatch(/^temp-/)

    resolve(created)
    await call
    await flushPromises()

    expect(store.profiles.some((p) => p.id === 'server-1')).toBe(true)
    expect(mockToastSuccess).toHaveBeenCalledWith('Profile added')
  })

  it('createProfile rolls back and toasts on error', async () => {
    mockedListProfiles.mockResolvedValueOnce([])
    const { store } = mountStore()
    await flushPromises()

    const { promise, reject } = deferred<Profile>()
    mockedCreateProfile.mockReturnValueOnce(promise)
    mockedListProfiles.mockResolvedValueOnce([])

    const call = store
      .createProfile({ name: 'N', language: 'es', formality: 'neutral', emojis: false, samples: [] })
      .catch(() => undefined)
    await Promise.resolve()
    await Promise.resolve()

    expect(store.profiles).toHaveLength(1)

    reject(new Error('Create failed'))
    await call
    await flushPromises()

    expect(store.profiles).toHaveLength(0)
    expect(mockToastError).toHaveBeenCalledWith('Create failed')
  })

  it('removeProfile optimistically drops the row and rolls back + toasts on error', async () => {
    mockedListProfiles.mockResolvedValueOnce([makeProfile({ id: 'p1' }), makeProfile({ id: 'p2' })])
    const { store } = mountStore()
    await flushPromises()

    const { promise, reject } = deferred<void>()
    mockedDeleteProfile.mockReturnValueOnce(promise)
    mockedListProfiles.mockResolvedValueOnce([makeProfile({ id: 'p1' }), makeProfile({ id: 'p2' })])

    const call = store.removeProfile('p1').catch(() => undefined)
    await Promise.resolve()
    await Promise.resolve()

    expect(store.profiles.map((p) => p.id)).toEqual(['p2'])

    reject(new Error('Delete failed'))
    await call
    await flushPromises()

    expect(store.profiles.map((p) => p.id).sort()).toEqual(['p1', 'p2'])
    expect(mockToastError).toHaveBeenCalledWith('Delete failed')
  })

  it('updateProfile optimistically patches fields before the request resolves', async () => {
    mockedListProfiles.mockResolvedValueOnce([makeProfile({ id: 'p1', name: 'Old name' })])
    const { store } = mountStore()
    await flushPromises()

    const updated = makeProfile({ id: 'p1', name: 'New name' })
    const { promise, resolve } = deferred<Profile>()
    mockedUpdateProfile.mockReturnValueOnce(promise)
    mockedListProfiles.mockResolvedValueOnce([updated])

    const call = store.updateProfile('p1', {
      name: 'New name',
      language: 'es',
      formality: 'neutral',
      emojis: false,
      samples: ['sample one'],
    })
    await Promise.resolve()
    await Promise.resolve()

    expect(store.profiles[0]!.name).toBe('New name')

    resolve(updated)
    await call
    await flushPromises()

    expect(mockToastSuccess).toHaveBeenCalledWith('Saved')
  })

  it('redistillProfile optimistically marks the style guide pending, then reconciles on success', async () => {
    mockedListProfiles.mockResolvedValueOnce([makeProfile({ id: 'p1', styleGuideStatus: 'ready' })])
    const { store } = mountStore()
    await flushPromises()

    const redistilled = makeProfile({ id: 'p1', styleGuideStatus: 'ready', styleGuide: 'new guide' })
    const { promise, resolve } = deferred<Profile>()
    mockedRedistillProfile.mockReturnValueOnce(promise)
    mockedListProfiles.mockResolvedValueOnce([redistilled])

    const call = store.redistillProfile('p1')
    await Promise.resolve()
    await Promise.resolve()

    expect(store.profiles[0]!.styleGuideStatus).toBe('pending')

    resolve(redistilled)
    await call
    await flushPromises()

    expect(store.profiles[0]!.styleGuide).toBe('new guide')
    expect(mockToastSuccess).toHaveBeenCalledWith('Style guide redistillation started')
  })

  it('redistillProfile rolls back and toasts on error', async () => {
    mockedListProfiles.mockResolvedValueOnce([makeProfile({ id: 'p1', styleGuideStatus: 'ready' })])
    const { store } = mountStore()
    await flushPromises()

    const { promise, reject } = deferred<Profile>()
    mockedRedistillProfile.mockReturnValueOnce(promise)
    mockedListProfiles.mockResolvedValueOnce([makeProfile({ id: 'p1', styleGuideStatus: 'ready' })])

    const call = store.redistillProfile('p1').catch(() => undefined)
    await Promise.resolve()
    await Promise.resolve()

    expect(store.profiles[0]!.styleGuideStatus).toBe('pending')

    reject(new Error('Redistill failed'))
    await call
    await flushPromises()

    expect(store.profiles[0]!.styleGuideStatus).toBe('ready')
    expect(mockToastError).toHaveBeenCalledWith('Redistill failed')
  })
})
