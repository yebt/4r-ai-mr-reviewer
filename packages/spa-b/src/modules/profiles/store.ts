import { computed } from 'vue'
import { defineStore } from 'pinia'
import { useMutation, useQuery, useQueryCache } from '@pinia/colada'
import { useToast } from '@shared/composables/useToast'
import * as profilesApi from './api'
import type { CreateProfilePayload, Profile, UpdateProfilePayload } from './types'

export const PROFILES_QUERY_KEY = ['profiles'] as const

/**
 * Pure cache-patch helpers used by the mutations' `onMutate` hooks below.
 * Exported so the optimistic-update logic can be unit-tested directly,
 * without spinning up a full Pinia Colada (Vue app + plugin) context.
 */
export function withOptimisticCreate(profiles: Profile[], optimistic: Profile): Profile[] {
  return [...profiles, optimistic]
}

export function withoutProfile(profiles: Profile[], id: string): Profile[] {
  return profiles.filter((profile) => profile.id !== id)
}

export function withPatchedProfile(
  profiles: Profile[],
  id: string,
  patch: Partial<Omit<Profile, 'id'>>,
): Profile[] {
  return profiles.map((profile) => (profile.id === id ? { ...profile, ...patch } : profile))
}

export function toProfilePatch(payload: UpdateProfilePayload): Partial<Omit<Profile, 'id'>> {
  return {
    name: payload.name,
    language: payload.language,
    formality: payload.formality,
    emojis: payload.emojis,
    samples: payload.samples,
  }
}

export function makeOptimisticProfile(payload: CreateProfilePayload): Profile {
  const now = new Date().toISOString()
  return {
    id: `temp-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
    name: payload.name,
    language: payload.language,
    formality: payload.formality,
    emojis: payload.emojis,
    samples: payload.samples,
    styleGuide: '',
    styleGuideStatus: 'pending',
    styleGuideError: '',
    createdAt: now,
    updatedAt: now,
  }
}

export function resolveErrorMessage(err: unknown, fallback: string): string {
  return err instanceof Error ? err.message : fallback
}

/**
 * Profiles store, backed by @pinia/colada. The `['profiles']` query is the
 * single source of truth for the list — it owns caching, request dedupe, and
 * async loading state. Every mutation (create/update/remove/redistill)
 * patches that cache entry optimistically in `onMutate` (snapshotting the
 * previous value first), rolls back to the snapshot in `onError` (plus a
 * `toast.error`), and reconciles with the server in `onSettled` via
 * `invalidateQueries` so the authoritative response always wins.
 */
export const useProfilesStore = defineStore('profiles', () => {
  const queryCache = useQueryCache()
  const toast = useToast()

  const profilesQuery = useQuery({
    key: PROFILES_QUERY_KEY,
    query: profilesApi.listProfiles,
  })

  const profiles = computed(() => profilesQuery.data.value ?? [])

  const createMutation = useMutation({
    mutation: (payload: CreateProfilePayload) => profilesApi.createProfile(payload),
    onMutate(payload) {
      queryCache.cancelQueries({ key: PROFILES_QUERY_KEY })
      const previous = queryCache.getQueryData<Profile[]>(PROFILES_QUERY_KEY)
      const optimistic = makeOptimisticProfile(payload)
      queryCache.setQueryData<Profile[]>(
        PROFILES_QUERY_KEY,
        withOptimisticCreate(previous ?? [], optimistic),
      )
      return { previous, tempId: optimistic.id }
    },
    onSuccess(created, _payload, { tempId }) {
      const current = queryCache.getQueryData<Profile[]>(PROFILES_QUERY_KEY) ?? []
      queryCache.setQueryData<Profile[]>(
        PROFILES_QUERY_KEY,
        current.map((profile) => (profile.id === tempId ? created : profile)),
      )
      toast.success('Profile added')
    },
    onError(err, _payload, context) {
      if (context?.previous !== undefined) {
        queryCache.setQueryData(PROFILES_QUERY_KEY, context.previous)
      }
      toast.error(resolveErrorMessage(err, 'Failed to add profile'))
    },
    onSettled() {
      queryCache.invalidateQueries({ key: PROFILES_QUERY_KEY })
    },
  })

  const updateMutation = useMutation({
    mutation: (vars: { id: string; payload: UpdateProfilePayload }) =>
      profilesApi.updateProfile(vars.id, vars.payload),
    onMutate(vars) {
      queryCache.cancelQueries({ key: PROFILES_QUERY_KEY })
      const previous = queryCache.getQueryData<Profile[]>(PROFILES_QUERY_KEY)
      queryCache.setQueryData<Profile[]>(
        PROFILES_QUERY_KEY,
        withPatchedProfile(previous ?? [], vars.id, toProfilePatch(vars.payload)),
      )
      return { previous }
    },
    onSuccess(updated) {
      const current = queryCache.getQueryData<Profile[]>(PROFILES_QUERY_KEY) ?? []
      queryCache.setQueryData<Profile[]>(
        PROFILES_QUERY_KEY,
        withPatchedProfile(current, updated.id, updated),
      )
      toast.success('Saved')
    },
    onError(err, _vars, context) {
      if (context?.previous !== undefined) {
        queryCache.setQueryData(PROFILES_QUERY_KEY, context.previous)
      }
      toast.error(resolveErrorMessage(err, 'Failed to save profile'))
    },
    onSettled() {
      queryCache.invalidateQueries({ key: PROFILES_QUERY_KEY })
    },
  })

  const removeMutation = useMutation({
    mutation: (id: string) => profilesApi.deleteProfile(id),
    onMutate(id) {
      queryCache.cancelQueries({ key: PROFILES_QUERY_KEY })
      const previous = queryCache.getQueryData<Profile[]>(PROFILES_QUERY_KEY)
      queryCache.setQueryData<Profile[]>(PROFILES_QUERY_KEY, withoutProfile(previous ?? [], id))
      return { previous }
    },
    onSuccess() {
      toast.success('Profile deleted')
    },
    onError(err, _id, context) {
      if (context?.previous !== undefined) {
        queryCache.setQueryData(PROFILES_QUERY_KEY, context.previous)
      }
      toast.error(resolveErrorMessage(err, 'Failed to delete profile'))
    },
    onSettled() {
      queryCache.invalidateQueries({ key: PROFILES_QUERY_KEY })
    },
  })

  const redistillMutation = useMutation({
    mutation: (id: string) => profilesApi.redistillProfile(id),
    onMutate(id) {
      queryCache.cancelQueries({ key: PROFILES_QUERY_KEY })
      const previous = queryCache.getQueryData<Profile[]>(PROFILES_QUERY_KEY)
      queryCache.setQueryData<Profile[]>(
        PROFILES_QUERY_KEY,
        withPatchedProfile(previous ?? [], id, { styleGuideStatus: 'pending' }),
      )
      return { previous }
    },
    onSuccess(updated) {
      const current = queryCache.getQueryData<Profile[]>(PROFILES_QUERY_KEY) ?? []
      queryCache.setQueryData<Profile[]>(
        PROFILES_QUERY_KEY,
        withPatchedProfile(current, updated.id, updated),
      )
      toast.success('Style guide redistillation started')
    },
    onError(err, _id, context) {
      if (context?.previous !== undefined) {
        queryCache.setQueryData(PROFILES_QUERY_KEY, context.previous)
      }
      toast.error(resolveErrorMessage(err, 'Failed to redistill profile'))
    },
    onSettled() {
      queryCache.invalidateQueries({ key: PROFILES_QUERY_KEY })
    },
  })

  return {
    // ['profiles'] query surface
    profiles,
    profilesState: profilesQuery.state,
    asyncStatus: profilesQuery.asyncStatus,
    isLoading: profilesQuery.isLoading,
    error: profilesQuery.error,
    refetch: profilesQuery.refetch,

    // mutations — `mutateAsync` rethrows so callers keep their existing
    // try/catch flows on top of the store-level optimistic update +
    // rollback + toast handled above.
    createProfile: createMutation.mutateAsync,
    isCreating: createMutation.isLoading,
    updateProfile: (id: string, payload: UpdateProfilePayload) =>
      updateMutation.mutateAsync({ id, payload }),
    isUpdating: updateMutation.isLoading,
    removeProfile: removeMutation.mutateAsync,
    isRemoving: removeMutation.isLoading,
    redistillProfile: redistillMutation.mutateAsync,
    isRedistilling: redistillMutation.isLoading,
  }
})
