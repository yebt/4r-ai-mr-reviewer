import { defineStore } from 'pinia'
import {
  createCrudResource,
  resolveErrorMessage,
  withOptimisticCreate,
  withoutItem,
  withPatchedItem,
} from '@shared/data/createCrudResource'
import * as accountsApi from './api'
import type { Account, CreateAccountPayload, UpdateAccountPayload } from './types'

export const ACCOUNTS_QUERY_KEY = ['accounts'] as const

/**
 * Pure cache-patch helpers used by the mutations' `onMutate` hooks below.
 * Exported so the optimistic-update logic can be unit-tested directly,
 * without spinning up a full Pinia Colada (Vue app + plugin) context.
 *
 * `withOptimisticCreate`, `withoutAccount` and `withPatchedAccount` are thin,
 * entity-named wrappers around the generic helpers in `createCrudResource` —
 * accounts have no entity-specific create/patch behavior (unlike providers'
 * `isDefault` unflip), so they just delegate.
 */
export { withOptimisticCreate }

export function withoutAccount(accounts: Account[], id: string): Account[] {
  return withoutItem(accounts, id)
}

export function withPatchedAccount(
  accounts: Account[],
  id: string,
  patch: Partial<Omit<Account, 'id'>>,
): Account[] {
  return withPatchedItem(accounts, id, patch)
}

/**
 * `token` is write-only and never lives on `Account` — never spread the raw
 * update payload onto the cache, or it leaks a stray `token` field.
 */
export function toAccountPatch(payload: UpdateAccountPayload): Partial<Omit<Account, 'id'>> {
  return {
    name: payload.name,
    baseUrl: payload.baseUrl,
  }
}

export function makeOptimisticAccount(payload: CreateAccountPayload): Account {
  return {
    id: `temp-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
    name: payload.name,
    baseUrl: payload.baseUrl,
    createdAt: new Date().toISOString(),
  }
}

export { resolveErrorMessage }

/**
 * Accounts store, backed by @pinia/colada via `createCrudResource`. The
 * `['accounts']` query is the single source of truth for the list — it owns
 * caching, request dedupe, and async loading state. Every mutation
 * (create/update/remove) patches that cache entry optimistically in
 * `onMutate` (snapshotting the previous value first), rolls back to the
 * snapshot in `onError` (plus a `toast.error` for remove only), and
 * reconciles with the server in `onSettled` via `invalidateQueries` so the
 * authoritative response always wins.
 */
export const useAccountsStore = defineStore('accounts', () => {
  const resource = createCrudResource<Account, CreateAccountPayload, UpdateAccountPayload>({
    queryKey: ACCOUNTS_QUERY_KEY,
    list: accountsApi.listAccounts,
    create: accountsApi.createAccount,
    update: accountsApi.updateAccount,
    remove: accountsApi.deleteAccount,
    makeOptimistic: makeOptimisticAccount,
    toPatch: toAccountPatch,
    messages: {
      createSuccess: 'Account added',
      updateSuccess: 'Account saved',
      removeSuccess: 'Account deleted',
      removeErrorFallback: 'Failed to delete account',
    },
  })

  return {
    // ['accounts'] query surface
    accounts: resource.items,
    accountsState: resource.state,
    asyncStatus: resource.asyncStatus,
    isLoading: resource.isLoading,
    error: resource.error,
    refetch: resource.refetch,

    // mutations — `mutateAsync` rethrows so callers keep their existing
    // try/catch flows (e.g. AccountForm's inline `formError`) on top of the
    // store-level optimistic update + rollback + toast handled above.
    createAccount: resource.create,
    isCreating: resource.isCreating,
    updateAccount: resource.update,
    isUpdating: resource.isUpdating,
    removeAccount: resource.remove,
    isRemoving: resource.isRemoving,
  }
})
