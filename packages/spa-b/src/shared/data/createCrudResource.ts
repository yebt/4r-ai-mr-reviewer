import { computed } from 'vue'
import { useMutation, useQuery, useQueryCache, type EntryKey } from '@pinia/colada'
import { useToast } from '@shared/composables/useToast'
import { resolveErrorMessage } from '@shared/api/resolveErrorMessage'

export { resolveErrorMessage }

/**
 * Generic, entity-agnostic cache-patch helpers shared by every CRUD resource
 * built with `createCrudResource`. Each `modules/*\/store.ts` re-exports (or
 * thinly wraps, when it needs an entity-specific export name) these so its
 * `store.spec.ts` can keep importing pure helpers straight from `./store`
 * without any test rewrites.
 */
export function withOptimisticCreate<T>(items: T[], optimistic: T): T[] {
  return [...items, optimistic]
}

export function withoutItem<T extends { id: string }>(items: T[], id: string): T[] {
  return items.filter((item) => item.id !== id)
}

export function withPatchedItem<T extends { id: string }>(
  items: T[],
  id: string,
  patch: Partial<Omit<T, 'id'>>,
): T[] {
  return items.map((item) => (item.id === id ? { ...item, ...patch } : item))
}

export interface CrudResourceMessages {
  createSuccess: string
  updateSuccess: string
  removeSuccess: string
  removeErrorFallback: string
}

export interface CrudResourceConfig<T extends { id: string }, TCreate, TUpdate> {
  /** The `useQuery`/cache key for the list, e.g. `['providers']`. */
  queryKey: EntryKey
  list: () => Promise<T[]>
  create: (payload: TCreate) => Promise<T>
  update: (id: string, payload: TUpdate) => Promise<T>
  remove: (id: string) => Promise<void>
  /** Builds the temp-id row appended to the cache before the create request resolves. */
  makeOptimistic: (payload: TCreate) => T
  /** Builds the optimistic patch for an update, excluding write-only fields. */
  toPatch: (payload: TUpdate) => Partial<Omit<T, 'id'>>
  /**
   * How the optimistic temp row is merged into the current list. Defaults to
   * a plain append (`withOptimisticCreate`); pass a custom function for
   * entity-specific placement (e.g. providers unflipping other rows'
   * `isDefault` when the new row is created as the default).
   */
  applyOptimisticCreate?: (items: T[], optimistic: T) => T[]
  messages: CrudResourceMessages
}

/**
 * Encapsulates the `[queryKey]` list query + the 3 base mutations
 * (create/update/remove) shared by every settings-module store: optimistic
 * update in `onMutate` (snapshotting the previous cache value first),
 * rollback to the snapshot in `onError`, and reconciliation with the server
 * in `onSettled` via `invalidateQueries`.
 *
 * Preserves the stores' existing error-toast policy exactly: create/update
 * never toast on error — only rollback, since the open form's inline
 * `formError` banner is the contextual surface for those failures — while
 * remove DOES toast on error.
 *
 * Returns the query's `queryCache` and `toast` alongside the base surface so
 * a store's own extra mutations (setDefault, redistill, …) can share this
 * resource's cache and toast instance instead of re-injecting their own.
 */
export function createCrudResource<T extends { id: string }, TCreate, TUpdate>(
  config: CrudResourceConfig<T, TCreate, TUpdate>,
) {
  const queryCache = useQueryCache()
  const toast = useToast()
  const { queryKey } = config
  const applyOptimisticCreate = config.applyOptimisticCreate ?? withOptimisticCreate

  const query = useQuery({
    key: queryKey,
    query: config.list,
  })

  const items = computed(() => query.data.value ?? [])

  const createMutation = useMutation({
    mutation: (payload: TCreate) => config.create(payload),
    onMutate(payload) {
      queryCache.cancelQueries({ key: queryKey })
      const previous = queryCache.getQueryData<T[]>(queryKey)
      const optimistic = config.makeOptimistic(payload)
      queryCache.setQueryData<T[]>(queryKey, applyOptimisticCreate(previous ?? [], optimistic))
      return { previous, tempId: optimistic.id }
    },
    onSuccess(created, _payload, { tempId }) {
      const current = queryCache.getQueryData<T[]>(queryKey) ?? []
      queryCache.setQueryData<T[]>(
        queryKey,
        current.map((item) => (item.id === tempId ? created : item)),
      )
      toast.success(config.messages.createSuccess)
    },
    // No `toast.error` here: the open form's inline `formError` banner is
    // the contextual surface for create/update failures. Toasting here too
    // would show the same error twice.
    onError(_err, _payload, context) {
      if (context?.previous !== undefined) {
        queryCache.setQueryData(queryKey, context.previous)
      }
    },
    onSettled() {
      queryCache.invalidateQueries({ key: queryKey })
    },
  })

  const updateMutation = useMutation({
    mutation: (vars: { id: string; payload: TUpdate }) => config.update(vars.id, vars.payload),
    onMutate(vars) {
      queryCache.cancelQueries({ key: queryKey })
      const previous = queryCache.getQueryData<T[]>(queryKey)
      queryCache.setQueryData<T[]>(
        queryKey,
        withPatchedItem(previous ?? [], vars.id, config.toPatch(vars.payload)),
      )
      return { previous }
    },
    onSuccess(updated) {
      const current = queryCache.getQueryData<T[]>(queryKey) ?? []
      queryCache.setQueryData<T[]>(queryKey, withPatchedItem(current, updated.id, updated))
      toast.success(config.messages.updateSuccess)
    },
    // No `toast.error` here — see createMutation.onError above.
    onError(_err, _vars, context) {
      if (context?.previous !== undefined) {
        queryCache.setQueryData(queryKey, context.previous)
      }
    },
    onSettled() {
      queryCache.invalidateQueries({ key: queryKey })
    },
  })

  const removeMutation = useMutation({
    mutation: (id: string) => config.remove(id),
    onMutate(id) {
      queryCache.cancelQueries({ key: queryKey })
      const previous = queryCache.getQueryData<T[]>(queryKey)
      queryCache.setQueryData<T[]>(queryKey, withoutItem(previous ?? [], id))
      return { previous }
    },
    onSuccess() {
      toast.success(config.messages.removeSuccess)
    },
    onError(err, _id, context) {
      if (context?.previous !== undefined) {
        queryCache.setQueryData(queryKey, context.previous)
      }
      toast.error(resolveErrorMessage(err, config.messages.removeErrorFallback))
    },
    onSettled() {
      queryCache.invalidateQueries({ key: queryKey })
    },
  })

  return {
    // Escape hatches for a store's own extra mutations (setDefault,
    // redistill, …) layered on top of this resource.
    queryCache,
    toast,

    // [queryKey] query surface
    items,
    state: query.state,
    asyncStatus: query.asyncStatus,
    isLoading: query.isLoading,
    error: query.error,
    refetch: query.refetch,

    // mutations — `mutateAsync` rethrows so callers keep their existing
    // try/catch flows (e.g. a form's inline `formError`) on top of the
    // optimistic update + rollback + toast handled above.
    create: createMutation.mutateAsync,
    isCreating: createMutation.isLoading,
    update: (id: string, payload: TUpdate) => updateMutation.mutateAsync({ id, payload }),
    isUpdating: updateMutation.isLoading,
    remove: removeMutation.mutateAsync,
    isRemoving: removeMutation.isLoading,
  }
}
