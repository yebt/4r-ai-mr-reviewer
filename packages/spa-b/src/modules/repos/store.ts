import { useMutation } from '@pinia/colada'
import { defineStore } from 'pinia'
import {
  createCrudResource,
  resolveErrorMessage,
  withOptimisticCreate,
  withoutItem,
  withPatchedItem,
} from '@shared/data/createCrudResource'
import * as reposApi from './api'
import type { AssignRepoPayload, CreateRepoPayload, Repo, SetWebhookPayload } from './types'

export const REPOS_QUERY_KEY = ['repos'] as const

/**
 * Pure cache-patch helpers used by the mutations' `onMutate`/`onSuccess`
 * hooks below. Exported so the optimistic-update logic can be unit-tested
 * directly, without spinning up a full Pinia Colada (Vue app + plugin)
 * context. Repos have no entity-specific create placement (unlike
 * providers' `isDefault` unflip), so `withOptimisticCreate` just delegates.
 */
export { withOptimisticCreate }

export function withoutRepo(repos: Repo[], id: string): Repo[] {
  return withoutItem(repos, id)
}

export function withPatchedRepo(repos: Repo[], id: string, patch: Partial<Omit<Repo, 'id'>>): Repo[] {
  return withPatchedItem(repos, id, patch)
}

/**
 * `AssignRepoPayload` has no `name`/`url` (not editable after create), so
 * the optimistic patch built from it only ever touches the assignment
 * fields — `defaultProfileId` is the DTO's name for the payload's `profileId`.
 */
export function toRepoPatch(payload: AssignRepoPayload): Partial<Omit<Repo, 'id'>> {
  return {
    accountId: payload.accountId,
    providerId: payload.providerId,
    model: payload.model,
    defaultProfileId: payload.profileId,
  }
}

export function makeOptimisticRepo(payload: CreateRepoPayload): Repo {
  return {
    id: `temp-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
    name: payload.name,
    url: payload.url,
    accountId: payload.accountId,
    providerId: payload.providerId,
    model: payload.model,
    defaultProfileId: payload.profileId,
    webhookEnabled: false,
    webhookRequireConfirmation: false,
    webhookSecret: '',
    webhookPath: '',
    createdAt: new Date().toISOString(),
  }
}

export { resolveErrorMessage }

/**
 * Repos store, backed by @pinia/colada via `createCrudResource`. The
 * `['repos']` query is the single source of truth for the list. The base
 * mutations (create/remove) and the factory's "update" slot — mapped to
 * `PATCH /repos/{id}/assign`, since repos have no general update endpoint —
 * patch that cache entry optimistically in `onMutate`, roll back on error,
 * and reconcile with the server in `onSettled` via `invalidateQueries`,
 * exactly like providers/accounts/profiles.
 *
 * `setWebhook`/`rotateWebhookSecret` are repo-specific extras layered on top
 * of the factory: unlike assign, their result (in particular a freshly
 * rotated `webhookSecret`) can't be guessed client-side, so they skip the
 * optimistic `onMutate` patch and instead patch the cache row directly from
 * the server response in `onSuccess` — while still sharing the resource's
 * `queryCache`/`toast`.
 */
export const useReposStore = defineStore('repos', () => {
  const resource = createCrudResource<Repo, CreateRepoPayload, AssignRepoPayload>({
    queryKey: REPOS_QUERY_KEY,
    list: reposApi.listRepos,
    create: reposApi.createRepo,
    update: reposApi.assignRepo,
    remove: reposApi.deleteRepo,
    makeOptimistic: makeOptimisticRepo,
    toPatch: toRepoPatch,
    messages: {
      createSuccess: 'Repository added',
      updateSuccess: 'Repository reassigned',
      removeSuccess: 'Repository deleted',
      removeErrorFallback: 'Failed to delete repository',
    },
  })

  const setWebhookMutation = useMutation({
    mutation: (vars: { id: string; payload: SetWebhookPayload }) =>
      reposApi.setRepoWebhook(vars.id, vars.payload),
    onSuccess(updated) {
      const current = resource.queryCache.getQueryData<Repo[]>(REPOS_QUERY_KEY) ?? []
      resource.queryCache.setQueryData<Repo[]>(
        REPOS_QUERY_KEY,
        withPatchedRepo(current, updated.id, updated),
      )
      resource.toast.success('Webhook updated')
    },
    onError(err) {
      resource.toast.error(resolveErrorMessage(err, 'Failed to update webhook'))
    },
  })

  function setWebhook(id: string, payload: SetWebhookPayload) {
    return setWebhookMutation.mutateAsync({ id, payload })
  }

  const rotateWebhookSecretMutation = useMutation({
    mutation: (id: string) => reposApi.rotateRepoWebhookSecret(id),
    onSuccess(updated) {
      const current = resource.queryCache.getQueryData<Repo[]>(REPOS_QUERY_KEY) ?? []
      resource.queryCache.setQueryData<Repo[]>(
        REPOS_QUERY_KEY,
        withPatchedRepo(current, updated.id, updated),
      )
      resource.toast.success('Webhook secret rotated')
    },
    onError(err) {
      resource.toast.error(resolveErrorMessage(err, 'Failed to rotate webhook secret'))
    },
  })

  function rotateWebhookSecret(id: string) {
    return rotateWebhookSecretMutation.mutateAsync(id)
  }

  return {
    // ['repos'] query surface
    repos: resource.items,
    reposState: resource.state,
    asyncStatus: resource.asyncStatus,
    isLoading: resource.isLoading,
    error: resource.error,
    refetch: resource.refetch,

    // mutations — `mutateAsync` rethrows so callers keep their existing
    // try/catch flows (e.g. RepoForm's inline `formError`) on top of the
    // store-level optimistic update + rollback + toast handled above.
    createRepo: resource.create,
    isCreating: resource.isCreating,
    assignRepo: resource.update,
    isAssigning: resource.isUpdating,
    removeRepo: resource.remove,
    isRemoving: resource.isRemoving,

    setWebhook,
    isSettingWebhook: setWebhookMutation.isLoading,
    rotateWebhookSecret,
    isRotatingWebhookSecret: rotateWebhookSecretMutation.isLoading,
  }
})
