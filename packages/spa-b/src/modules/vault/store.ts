import { computed } from 'vue'
import { defineStore } from 'pinia'
import { useMutation, useQuery, useQueryCache } from '@pinia/colada'
import { ApiError } from '@shared/api/client'
import { resolveErrorMessage } from '@shared/api/resolveErrorMessage'
import * as vaultApi from './api'
import type {
  ChangeVaultPasswordPayload,
  ChangeVaultPasswordResult,
  VaultStatus,
  VaultStatusResult,
  VaultUnavailable,
} from './types'

export const VAULT_STATUS_QUERY_KEY = ['vault-status'] as const

const UNAVAILABLE: VaultUnavailable = { unavailable: true }

export function isVaultUnavailable(result: VaultStatusResult | undefined): result is VaultUnavailable {
  return result !== undefined && 'unavailable' in result
}

/**
 * Fetches vault status, translating a 501 response (vault management
 * unavailable on this build) into the `{ unavailable: true }` sentinel
 * instead of letting the `ApiError` propagate — 501 here is a legitimate,
 * successful query outcome (hide the whole card), never a query error state
 * (error banner + retry).
 */
export async function fetchVaultStatusOrUnavailable(): Promise<VaultStatusResult> {
  try {
    return await vaultApi.fetchVaultStatus()
  } catch (err) {
    if (err instanceof ApiError && err.status === 501) return UNAVAILABLE
    throw err
  }
}

/** Maps a `changeVaultPassword` failure to a user-facing message per the documented status codes. */
export function resolveVaultErrorMessage(err: unknown): string {
  if (err instanceof ApiError) {
    if (err.status === 401) return 'Current password is incorrect'
    if (err.status === 409) return 'Vault is not initialized'
    if (err.status === 501) return "Vault management isn't available on this instance."
  }
  return resolveErrorMessage(err, 'Failed to change the master key')
}

/**
 * Vault (security) store. The `['vault-status']` query is the single source
 * of truth for the current mode; its `query` function
 * (`fetchVaultStatusOrUnavailable`) is what turns a 501 into the
 * `{ unavailable: true }` sentinel above, so `unavailable`/`status` below
 * are plain derived reads — no separate error-shaped state to reconcile.
 *
 * `changePassword` deliberately has NO optimistic update: this is a
 * security-sensitive mutation (a password/key-file change), so the UI waits
 * for the real server response before reflecting a new mode. On success it
 * only invalidates the status query so the next read is authoritative; the
 * caller (`SecurityCard.vue`) is responsible for the toast + persistent
 * `result.warning` alert, since that copy depends on the submitted payload
 * (contextual "password changed" vs "switched to key-file mode" wording)
 * that this store doesn't otherwise carry.
 */
export const useVaultStore = defineStore('vault', () => {
  const queryCache = useQueryCache()

  const statusQuery = useQuery({
    key: VAULT_STATUS_QUERY_KEY,
    query: fetchVaultStatusOrUnavailable,
  })

  const unavailable = computed(() => isVaultUnavailable(statusQuery.data.value))
  const status = computed<VaultStatus | null>(() => {
    const data = statusQuery.data.value
    return data && !isVaultUnavailable(data) ? data : null
  })

  const changePasswordMutation = useMutation({
    mutation: (payload: ChangeVaultPasswordPayload) => vaultApi.changeVaultPassword(payload),
    onSuccess() {
      queryCache.invalidateQueries({ key: VAULT_STATUS_QUERY_KEY })
    },
  })

  // `mutateAsync` rethrows on error — the caller maps it via
  // `resolveVaultErrorMessage` for its inline form error.
  function changePassword(payload: ChangeVaultPasswordPayload): Promise<ChangeVaultPasswordResult> {
    return changePasswordMutation.mutateAsync(payload)
  }

  return {
    // ['vault-status'] query surface
    status,
    unavailable,
    statusState: statusQuery.state,
    isLoading: statusQuery.isLoading,
    error: statusQuery.error,
    refetch: statusQuery.refetch,

    // mutation
    changePassword,
    isChangingPassword: changePasswordMutation.isLoading,
  }
})
