import { computed } from 'vue'
import { defineStore } from 'pinia'
import { useMutation, useQuery, useQueryCache } from '@pinia/colada'
import { useToast } from '@shared/composables/useToast'
import { resolveErrorMessage } from '@shared/api/resolveErrorMessage'
import * as telegramApi from './api'
import type {
  CreateTelegramTargetPayload,
  TelegramTarget,
  TestTelegramTargetResult,
  UpdateTelegramTargetPayload,
} from './types'

export const TELEGRAM_QUERY_KEY = ['telegram'] as const

/**
 * Pure cache-patch helpers used by the mutations' `onMutate` hooks below.
 * Exported so the optimistic-update logic can be unit-tested directly,
 * without spinning up a full Pinia Colada (Vue app + plugin) context.
 */
export function withOptimisticCreate(targets: TelegramTarget[], optimistic: TelegramTarget): TelegramTarget[] {
  return [...targets, optimistic]
}

export function withoutTarget(targets: TelegramTarget[], id: string): TelegramTarget[] {
  return targets.filter((target) => target.id !== id)
}

export function withPatchedTarget(
  targets: TelegramTarget[],
  id: string,
  patch: Partial<Omit<TelegramTarget, 'id'>>,
): TelegramTarget[] {
  return targets.map((target) => (target.id === id ? { ...target, ...patch } : target))
}

export function withDefaultFlippedTo(targets: TelegramTarget[], id: string): TelegramTarget[] {
  return targets.map((target) => ({ ...target, isDefault: target.id === id }))
}

/**
 * Default-first ordering for the list UI (`TelegramSection`): `Array#sort`
 * is stable, so this only ever moves the default row to the top — every
 * other row keeps its relative order, which is what makes "Set default"
 * read as an obvious, single-item reorder rather than a full reshuffle.
 */
export function sortDefaultFirst(targets: TelegramTarget[]): TelegramTarget[] {
  return [...targets].sort((a, b) => Number(b.isDefault) - Number(a.isDefault))
}

/**
 * `botToken` is write-only and never lives on `TelegramTarget` — never
 * spread the raw update payload onto the cache, or it leaks a stray
 * `botToken` field.
 */
export function toTargetPatch(payload: UpdateTelegramTargetPayload): Partial<Omit<TelegramTarget, 'id'>> {
  return {
    name: payload.name,
    chatId: payload.chatId,
    threadId: payload.threadId,
    isBot: payload.isBot ?? false,
  }
}

export function makeOptimisticTarget(payload: CreateTelegramTargetPayload): TelegramTarget {
  return {
    id: `temp-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
    name: payload.name,
    chatId: payload.chatId,
    threadId: payload.threadId,
    isDefault: false,
    isBot: payload.isBot ?? false,
    createdAt: new Date().toISOString(),
  }
}

export { resolveErrorMessage }

/**
 * Telegram store, backed by @pinia/colada. Mirrors `providers/store.ts`:
 * the `['telegram']` query is the single source of truth for the list — it
 * owns caching, request dedupe, and async loading state. Every mutation
 * (create/update/remove/setDefault) patches that cache entry optimistically
 * in `onMutate` (snapshotting the previous value first), rolls back to the
 * snapshot in `onError` (plus a `toast.error`), and reconciles with the
 * server in `onSettled` via `invalidateQueries` so the authoritative
 * response always wins.
 */
export const useTelegramStore = defineStore('telegram', () => {
  const queryCache = useQueryCache()
  const toast = useToast()

  const targetsQuery = useQuery({
    key: TELEGRAM_QUERY_KEY,
    query: telegramApi.listTelegramTargets,
  })

  const targets = computed(() => targetsQuery.data.value ?? [])

  const createMutation = useMutation({
    mutation: (payload: CreateTelegramTargetPayload) => telegramApi.createTelegramTarget(payload),
    onMutate(payload) {
      queryCache.cancelQueries({ key: TELEGRAM_QUERY_KEY })
      const previous = queryCache.getQueryData<TelegramTarget[]>(TELEGRAM_QUERY_KEY)
      const optimistic = makeOptimisticTarget(payload)
      queryCache.setQueryData<TelegramTarget[]>(
        TELEGRAM_QUERY_KEY,
        withOptimisticCreate(previous ?? [], optimistic),
      )
      return { previous, tempId: optimistic.id }
    },
    onSuccess(created, _payload, { tempId }) {
      const current = queryCache.getQueryData<TelegramTarget[]>(TELEGRAM_QUERY_KEY) ?? []
      queryCache.setQueryData<TelegramTarget[]>(
        TELEGRAM_QUERY_KEY,
        current.map((target) => (target.id === tempId ? created : target)),
      )
      toast.success('Telegram target added')
    },
    // No `toast.error` here: the open form's inline `formError` banner is
    // the contextual surface for create/update failures. Toasting here too
    // would show the same error twice.
    onError(_err, _payload, context) {
      if (context?.previous !== undefined) {
        queryCache.setQueryData(TELEGRAM_QUERY_KEY, context.previous)
      }
    },
    onSettled() {
      queryCache.invalidateQueries({ key: TELEGRAM_QUERY_KEY })
    },
  })

  const updateMutation = useMutation({
    mutation: (vars: { id: string; payload: UpdateTelegramTargetPayload }) =>
      telegramApi.updateTelegramTarget(vars.id, vars.payload),
    onMutate(vars) {
      queryCache.cancelQueries({ key: TELEGRAM_QUERY_KEY })
      const previous = queryCache.getQueryData<TelegramTarget[]>(TELEGRAM_QUERY_KEY)
      queryCache.setQueryData<TelegramTarget[]>(
        TELEGRAM_QUERY_KEY,
        withPatchedTarget(previous ?? [], vars.id, toTargetPatch(vars.payload)),
      )
      return { previous }
    },
    onSuccess(updated) {
      const current = queryCache.getQueryData<TelegramTarget[]>(TELEGRAM_QUERY_KEY) ?? []
      queryCache.setQueryData<TelegramTarget[]>(
        TELEGRAM_QUERY_KEY,
        withPatchedTarget(current, updated.id, updated),
      )
      toast.success('Telegram target saved')
    },
    // No `toast.error` here: the open form's inline `formError` banner is
    // the contextual surface for create/update failures. Toasting here too
    // would show the same error twice.
    onError(_err, _vars, context) {
      if (context?.previous !== undefined) {
        queryCache.setQueryData(TELEGRAM_QUERY_KEY, context.previous)
      }
    },
    onSettled() {
      queryCache.invalidateQueries({ key: TELEGRAM_QUERY_KEY })
    },
  })

  const removeMutation = useMutation({
    mutation: (id: string) => telegramApi.deleteTelegramTarget(id),
    onMutate(id) {
      queryCache.cancelQueries({ key: TELEGRAM_QUERY_KEY })
      const previous = queryCache.getQueryData<TelegramTarget[]>(TELEGRAM_QUERY_KEY)
      queryCache.setQueryData<TelegramTarget[]>(TELEGRAM_QUERY_KEY, withoutTarget(previous ?? [], id))
      return { previous }
    },
    onSuccess() {
      toast.success('Telegram target deleted')
    },
    onError(err, _id, context) {
      if (context?.previous !== undefined) {
        queryCache.setQueryData(TELEGRAM_QUERY_KEY, context.previous)
      }
      toast.error(resolveErrorMessage(err, 'Failed to delete Telegram target'))
    },
    onSettled() {
      queryCache.invalidateQueries({ key: TELEGRAM_QUERY_KEY })
    },
  })

  const setDefaultMutation = useMutation({
    mutation: (id: string) => telegramApi.setDefaultTelegramTarget(id),
    onMutate(id) {
      queryCache.cancelQueries({ key: TELEGRAM_QUERY_KEY })
      const previous = queryCache.getQueryData<TelegramTarget[]>(TELEGRAM_QUERY_KEY)
      queryCache.setQueryData<TelegramTarget[]>(
        TELEGRAM_QUERY_KEY,
        withDefaultFlippedTo(previous ?? [], id),
      )
      return { previous }
    },
    onSuccess() {
      toast.success('Set as default')
    },
    onError(err, _id, context) {
      if (context?.previous !== undefined) {
        queryCache.setQueryData(TELEGRAM_QUERY_KEY, context.previous)
      }
      toast.error(resolveErrorMessage(err, 'Failed to set default Telegram target'))
    },
    onSettled() {
      queryCache.invalidateQueries({ key: TELEGRAM_QUERY_KEY })
    },
  })

  /** Never throws for a reachable-but-rejecting target — resolves `{ ok, error? }`. */
  function testTarget(id: string): Promise<TestTelegramTargetResult> {
    return telegramApi.testTelegramTarget(id)
  }

  return {
    // ['telegram'] query surface
    targets,
    targetsState: targetsQuery.state,
    asyncStatus: targetsQuery.asyncStatus,
    isLoading: targetsQuery.isLoading,
    error: targetsQuery.error,
    refetch: targetsQuery.refetch,

    // mutations — `mutateAsync` rethrows so callers keep their existing
    // try/catch flows on top of the store-level optimistic update +
    // rollback + toast handled above.
    createTarget: createMutation.mutateAsync,
    isCreating: createMutation.isLoading,
    updateTarget: (id: string, payload: UpdateTelegramTargetPayload) =>
      updateMutation.mutateAsync({ id, payload }),
    isUpdating: updateMutation.isLoading,
    removeTarget: removeMutation.mutateAsync,
    isRemoving: removeMutation.isLoading,
    setDefaultTarget: setDefaultMutation.mutateAsync,
    isSettingDefault: setDefaultMutation.isLoading,
    testTarget,
  }
})
