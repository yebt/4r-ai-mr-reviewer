import { defineStore } from 'pinia'
import { useMutation } from '@pinia/colada'
import {
  createCrudResource,
  resolveErrorMessage,
  withOptimisticCreate,
  withoutItem,
  withPatchedItem,
} from '@shared/data/createCrudResource'
import * as telegramApi from './api'
import type {
  CreateTelegramTargetPayload,
  TelegramTarget,
  UpdateTelegramTargetPayload,
} from './types'

export const TELEGRAM_QUERY_KEY = ['telegram'] as const

/**
 * Pure cache-patch helpers used by the mutations' `onMutate` hooks below.
 * Exported so the optimistic-update logic can be unit-tested directly,
 * without spinning up a full Pinia Colada (Vue app + plugin) context.
 *
 * `withOptimisticCreate`, `withoutTarget` and `withPatchedTarget` are thin,
 * entity-named wrappers around the generic helpers in `createCrudResource` —
 * a newly created target is never optimistically default (`isBot`/`isDefault`
 * come straight off the payload/default), so no unflip is needed here.
 */
export { withOptimisticCreate }

export function withoutTarget(targets: TelegramTarget[], id: string): TelegramTarget[] {
  return withoutItem(targets, id)
}

export function withPatchedTarget(
  targets: TelegramTarget[],
  id: string,
  patch: Partial<Omit<TelegramTarget, 'id'>>,
): TelegramTarget[] {
  return withPatchedItem(targets, id, patch)
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
 * Telegram store, backed by @pinia/colada via `createCrudResource`. Mirrors
 * `providers/store.ts`: the `['telegram']` query is the single source of
 * truth for the list — it owns caching, request dedupe, and async loading
 * state. The base mutations (create/update/remove) patch that cache entry
 * optimistically in `onMutate` (snapshotting the previous value first), roll
 * back to the snapshot in `onError` (plus a `toast.error` for remove only),
 * and reconcile with the server in `onSettled` via `invalidateQueries` so the
 * authoritative response always wins.
 *
 * `setDefaultTarget` is a telegram-specific extra layered on top of the
 * factory: it shares the resource's `queryCache`/`toast` but follows the
 * same optimistic-patch → rollback-on-error → invalidate shape by hand,
 * since it isn't one of the 3 base CRUD mutations.
 */
export const useTelegramStore = defineStore('telegram', () => {
  const resource = createCrudResource<TelegramTarget, CreateTelegramTargetPayload, UpdateTelegramTargetPayload>({
    queryKey: TELEGRAM_QUERY_KEY,
    list: telegramApi.listTelegramTargets,
    create: telegramApi.createTelegramTarget,
    update: telegramApi.updateTelegramTarget,
    remove: telegramApi.deleteTelegramTarget,
    makeOptimistic: makeOptimisticTarget,
    toPatch: toTargetPatch,
    messages: {
      createSuccess: 'Telegram target added',
      updateSuccess: 'Telegram target saved',
      removeSuccess: 'Telegram target deleted',
      removeErrorFallback: 'Failed to delete Telegram target',
    },
  })

  const setDefaultMutation = useMutation({
    mutation: (id: string) => telegramApi.setDefaultTelegramTarget(id),
    onMutate(id) {
      resource.queryCache.cancelQueries({ key: TELEGRAM_QUERY_KEY })
      const previous = resource.queryCache.getQueryData<TelegramTarget[]>(TELEGRAM_QUERY_KEY)
      resource.queryCache.setQueryData<TelegramTarget[]>(
        TELEGRAM_QUERY_KEY,
        withDefaultFlippedTo(previous ?? [], id),
      )
      return { previous }
    },
    onSuccess() {
      resource.toast.success('Set as default')
    },
    onError(err, _id, context) {
      if (context?.previous !== undefined) {
        resource.queryCache.setQueryData(TELEGRAM_QUERY_KEY, context.previous)
      }
      resource.toast.error(resolveErrorMessage(err, 'Failed to set default Telegram target'))
    },
    onSettled() {
      resource.queryCache.invalidateQueries({ key: TELEGRAM_QUERY_KEY })
    },
  })

  /** Delegates to the API — resolves (void) on success, rejects if the target is unreachable or delivery fails. */
  function testTarget(id: string): Promise<void> {
    return telegramApi.testTelegramTarget(id)
  }

  return {
    // ['telegram'] query surface
    targets: resource.items,
    targetsState: resource.state,
    asyncStatus: resource.asyncStatus,
    isLoading: resource.isLoading,
    error: resource.error,
    refetch: resource.refetch,

    // mutations — `mutateAsync` rethrows so callers keep their existing
    // try/catch flows on top of the store-level optimistic update +
    // rollback + toast handled above.
    createTarget: resource.create,
    isCreating: resource.isCreating,
    updateTarget: resource.update,
    isUpdating: resource.isUpdating,
    removeTarget: resource.remove,
    isRemoving: resource.isRemoving,
    setDefaultTarget: setDefaultMutation.mutateAsync,
    isSettingDefault: setDefaultMutation.isLoading,
    testTarget,
  }
})
