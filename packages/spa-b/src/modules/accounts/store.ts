import { computed } from 'vue'
import { defineStore } from 'pinia'
import { useMutation, useQuery, useQueryCache } from '@pinia/colada'
import { useToast } from '@shared/composables/useToast'
import { resolveErrorMessage } from '@shared/api/resolveErrorMessage'
import * as accountsApi from './api'
import type { Account, CreateAccountPayload, UpdateAccountPayload } from './types'

export const ACCOUNTS_QUERY_KEY = ['accounts'] as const

/**
 * Pure cache-patch helpers used by the mutations' `onMutate` hooks below.
 * Exported so the optimistic-update logic can be unit-tested directly,
 * without spinning up a full Pinia Colada (Vue app + plugin) context.
 */
export function withOptimisticCreate(accounts: Account[], optimistic: Account): Account[] {
  return [...accounts, optimistic]
}

export function withoutAccount(accounts: Account[], id: string): Account[] {
  return accounts.filter((account) => account.id !== id)
}

export function withPatchedAccount(
  accounts: Account[],
  id: string,
  patch: Partial<Omit<Account, 'id'>>,
): Account[] {
  return accounts.map((account) => (account.id === id ? { ...account, ...patch } : account))
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
 * Accounts store, backed by @pinia/colada. The `['accounts']` query is the
 * single source of truth for the list — it owns caching, request dedupe, and
 * async loading state. Every mutation (create/update/remove) patches that
 * cache entry optimistically in `onMutate` (snapshotting the previous value
 * first), rolls back to the snapshot in `onError` (plus a `toast.error`),
 * and reconciles with the server in `onSettled` via `invalidateQueries` so
 * the authoritative response always wins.
 */
export const useAccountsStore = defineStore('accounts', () => {
  const queryCache = useQueryCache()
  const toast = useToast()

  const accountsQuery = useQuery({
    key: ACCOUNTS_QUERY_KEY,
    query: accountsApi.listAccounts,
  })

  const accounts = computed(() => accountsQuery.data.value ?? [])

  const createMutation = useMutation({
    mutation: (payload: CreateAccountPayload) => accountsApi.createAccount(payload),
    onMutate(payload) {
      queryCache.cancelQueries({ key: ACCOUNTS_QUERY_KEY })
      const previous = queryCache.getQueryData<Account[]>(ACCOUNTS_QUERY_KEY)
      const optimistic = makeOptimisticAccount(payload)
      queryCache.setQueryData<Account[]>(
        ACCOUNTS_QUERY_KEY,
        withOptimisticCreate(previous ?? [], optimistic),
      )
      return { previous, tempId: optimistic.id }
    },
    onSuccess(created, _payload, { tempId }) {
      const current = queryCache.getQueryData<Account[]>(ACCOUNTS_QUERY_KEY) ?? []
      queryCache.setQueryData<Account[]>(
        ACCOUNTS_QUERY_KEY,
        current.map((account) => (account.id === tempId ? created : account)),
      )
      toast.success('Account added')
    },
    // No `toast.error` here: the open form's inline `formError` banner is
    // the contextual surface for create/update failures. Toasting here too
    // would show the same error twice.
    onError(_err, _payload, context) {
      if (context?.previous !== undefined) {
        queryCache.setQueryData(ACCOUNTS_QUERY_KEY, context.previous)
      }
    },
    onSettled() {
      queryCache.invalidateQueries({ key: ACCOUNTS_QUERY_KEY })
    },
  })

  const updateMutation = useMutation({
    mutation: (vars: { id: string; payload: UpdateAccountPayload }) =>
      accountsApi.updateAccount(vars.id, vars.payload),
    onMutate(vars) {
      queryCache.cancelQueries({ key: ACCOUNTS_QUERY_KEY })
      const previous = queryCache.getQueryData<Account[]>(ACCOUNTS_QUERY_KEY)
      queryCache.setQueryData<Account[]>(
        ACCOUNTS_QUERY_KEY,
        withPatchedAccount(previous ?? [], vars.id, toAccountPatch(vars.payload)),
      )
      return { previous }
    },
    onSuccess(updated) {
      const current = queryCache.getQueryData<Account[]>(ACCOUNTS_QUERY_KEY) ?? []
      queryCache.setQueryData<Account[]>(
        ACCOUNTS_QUERY_KEY,
        withPatchedAccount(current, updated.id, updated),
      )
      toast.success('Account saved')
    },
    // No `toast.error` here: the open form's inline `formError` banner is
    // the contextual surface for create/update failures. Toasting here too
    // would show the same error twice.
    onError(_err, _vars, context) {
      if (context?.previous !== undefined) {
        queryCache.setQueryData(ACCOUNTS_QUERY_KEY, context.previous)
      }
    },
    onSettled() {
      queryCache.invalidateQueries({ key: ACCOUNTS_QUERY_KEY })
    },
  })

  const removeMutation = useMutation({
    mutation: (id: string) => accountsApi.deleteAccount(id),
    onMutate(id) {
      queryCache.cancelQueries({ key: ACCOUNTS_QUERY_KEY })
      const previous = queryCache.getQueryData<Account[]>(ACCOUNTS_QUERY_KEY)
      queryCache.setQueryData<Account[]>(ACCOUNTS_QUERY_KEY, withoutAccount(previous ?? [], id))
      return { previous }
    },
    onSuccess() {
      toast.success('Account deleted')
    },
    onError(err, _id, context) {
      if (context?.previous !== undefined) {
        queryCache.setQueryData(ACCOUNTS_QUERY_KEY, context.previous)
      }
      toast.error(resolveErrorMessage(err, 'Failed to delete account'))
    },
    onSettled() {
      queryCache.invalidateQueries({ key: ACCOUNTS_QUERY_KEY })
    },
  })

  return {
    // ['accounts'] query surface
    accounts,
    accountsState: accountsQuery.state,
    asyncStatus: accountsQuery.asyncStatus,
    isLoading: accountsQuery.isLoading,
    error: accountsQuery.error,
    refetch: accountsQuery.refetch,

    // mutations — `mutateAsync` rethrows so callers keep their existing
    // try/catch flows (e.g. AccountForm's inline `formError`) on top of the
    // store-level optimistic update + rollback + toast handled above.
    createAccount: createMutation.mutateAsync,
    isCreating: createMutation.isLoading,
    updateAccount: (id: string, payload: UpdateAccountPayload) =>
      updateMutation.mutateAsync({ id, payload }),
    isUpdating: updateMutation.isLoading,
    removeAccount: removeMutation.mutateAsync,
    isRemoving: removeMutation.isLoading,
  }
})
