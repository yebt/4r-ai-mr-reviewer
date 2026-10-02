import { defineStore } from 'pinia'
import { useMutation } from '@pinia/colada'
import {
  createCrudResource,
  resolveErrorMessage,
  withOptimisticCreate,
  withoutItem,
  withPatchedItem,
} from '@shared/data/createCrudResource'
import * as profilesApi from './api'
import type { CreateProfilePayload, Profile, UpdateProfilePayload } from './types'

export const PROFILES_QUERY_KEY = ['profiles'] as const

/**
 * Pure cache-patch helpers used by the mutations' `onMutate` hooks below.
 * Exported so the optimistic-update logic can be unit-tested directly,
 * without spinning up a full Pinia Colada (Vue app + plugin) context.
 *
 * `withOptimisticCreate`, `withoutProfile` and `withPatchedProfile` are thin,
 * entity-named wrappers around the generic helpers in `createCrudResource` —
 * profiles have no entity-specific create/patch behavior, so they just
 * delegate.
 */
export { withOptimisticCreate }

export function withoutProfile(profiles: Profile[], id: string): Profile[] {
  return withoutItem(profiles, id)
}

export function withPatchedProfile(
  profiles: Profile[],
  id: string,
  patch: Partial<Omit<Profile, 'id'>>,
): Profile[] {
  return withPatchedItem(profiles, id, patch)
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

export { resolveErrorMessage }

/**
 * Profiles store, backed by @pinia/colada via `createCrudResource`. The
 * `['profiles']` query is the single source of truth for the list — it owns
 * caching, request dedupe, and async loading state. The base mutations
 * (create/update/remove) patch that cache entry optimistically in `onMutate`
 * (snapshotting the previous value first), roll back to the snapshot in
 * `onError` (plus a `toast.error` for remove only), and reconcile with the
 * server in `onSettled` via `invalidateQueries` so the authoritative response
 * always wins.
 *
 * `redistillProfile` is a profiles-specific extra layered on top of the
 * factory: it shares the resource's `queryCache`/`toast` but follows the same
 * optimistic-patch → rollback-on-error → invalidate shape by hand, since it
 * isn't one of the 3 base CRUD mutations.
 */
export const useProfilesStore = defineStore('profiles', () => {
  const resource = createCrudResource<Profile, CreateProfilePayload, UpdateProfilePayload>({
    queryKey: PROFILES_QUERY_KEY,
    list: profilesApi.listProfiles,
    create: profilesApi.createProfile,
    update: profilesApi.updateProfile,
    remove: profilesApi.deleteProfile,
    makeOptimistic: makeOptimisticProfile,
    toPatch: toProfilePatch,
    messages: {
      createSuccess: 'Profile added',
      updateSuccess: 'Profile saved',
      removeSuccess: 'Profile deleted',
      removeErrorFallback: 'Failed to delete profile',
    },
  })

  const redistillMutation = useMutation({
    mutation: (id: string) => profilesApi.redistillProfile(id),
    onMutate(id) {
      resource.queryCache.cancelQueries({ key: PROFILES_QUERY_KEY })
      const previous = resource.queryCache.getQueryData<Profile[]>(PROFILES_QUERY_KEY)
      resource.queryCache.setQueryData<Profile[]>(
        PROFILES_QUERY_KEY,
        withPatchedProfile(previous ?? [], id, { styleGuideStatus: 'pending' }),
      )
      return { previous }
    },
    onSuccess(updated) {
      const current = resource.queryCache.getQueryData<Profile[]>(PROFILES_QUERY_KEY) ?? []
      resource.queryCache.setQueryData<Profile[]>(
        PROFILES_QUERY_KEY,
        withPatchedProfile(current, updated.id, updated),
      )
      resource.toast.success('Redistilling…')
    },
    onError(err, _id, context) {
      if (context?.previous !== undefined) {
        resource.queryCache.setQueryData(PROFILES_QUERY_KEY, context.previous)
      }
      resource.toast.error(resolveErrorMessage(err, 'Failed to redistill profile'))
    },
    onSettled() {
      resource.queryCache.invalidateQueries({ key: PROFILES_QUERY_KEY })
    },
  })

  return {
    // ['profiles'] query surface
    profiles: resource.items,
    profilesState: resource.state,
    asyncStatus: resource.asyncStatus,
    isLoading: resource.isLoading,
    error: resource.error,
    refetch: resource.refetch,

    // mutations — `mutateAsync` rethrows so callers keep their existing
    // try/catch flows on top of the store-level optimistic update +
    // rollback + toast handled above.
    createProfile: resource.create,
    isCreating: resource.isCreating,
    updateProfile: resource.update,
    isUpdating: resource.isUpdating,
    removeProfile: resource.remove,
    isRemoving: resource.isRemoving,
    redistillProfile: redistillMutation.mutateAsync,
    isRedistilling: redistillMutation.isLoading,
  }
})
