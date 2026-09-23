import { computed, ref, watch } from 'vue'
import { defineStore } from 'pinia'
import { useMutation, useQuery, useQueryCache } from '@pinia/colada'
import { useIntervalFn } from '@vueuse/core'
import { useToast } from '@shared/composables/useToast'
import { resolveErrorMessage } from '@shared/api/resolveErrorMessage'
import * as runsApi from './api'
import { isRunActive } from './format'
import type { RoutineRun } from './types'

export { resolveErrorMessage }

const DEFAULT_LIMIT = 30
const POLL_INTERVAL_MS = 2500

export const RUNS_QUERY_ROOT = ['routines'] as const

/**
 * Pure query-key builder, exported so tests (and any future consumer) can
 * assert the exact key shape without reaching into the store's internals.
 */
export function runsQueryKey(archived: boolean): readonly [string, { archived: boolean }] {
  return ['routines', { archived }]
}

/**
 * Pure gate for the polling loop: true while at least one run in the list
 * is non-terminal (see `format.ts#isRunActive`). Kept standalone so it can
 * be unit-tested as a plain function instead of asserting on the interval
 * timer itself.
 */
export function shouldPoll(runs: RoutineRun[]): boolean {
  return runs.some((run) => isRunActive(run.status))
}

/**
 * Runs store — the global routine-run list (every repo, newest first). Not
 * a `createCrudResource` — this resource has no `create` and its mutations
 * are custom action endpoints (archive/unarchive/cancel/remove) rather than
 * the generic create/update/remove shape — so the query + mutations are
 * hand-rolled here, sharing one `useQueryCache()`/`useToast()` pair the way
 * `createCrudResource` does internally.
 *
 * `archived` is a plain ref toggle: flipping it changes the `['routines',
 * { archived }]` query key, so Pinia Colada fetches (and caches) the
 * archived list independently from the active one.
 *
 * Live polling: `useIntervalFn` refetches every 2.5s, started only while
 * `runs.some(isRunActive)` and paused otherwise — a `watch` on that
 * derived boolean (immediate, so a page load that already has an active
 * run starts polling right away) drives `resume()`/`pause()`.
 */
export const useRunsStore = defineStore('runs', () => {
  const queryCache = useQueryCache()
  const toast = useToast()

  const archived = ref(false)

  const query = useQuery({
    key: () => runsQueryKey(archived.value),
    query: () => runsApi.listRecentRoutines(DEFAULT_LIMIT, archived.value),
  })

  const runs = computed(() => query.data.value ?? [])
  const hasActiveRun = computed(() => shouldPoll(runs.value))

  const { pause, resume, isActive: isPolling } = useIntervalFn(() => query.refetch(), POLL_INTERVAL_MS, {
    immediate: false,
  })

  watch(hasActiveRun, (active) => (active ? resume() : pause()), { immediate: true })

  function invalidate() {
    queryCache.invalidateQueries({ key: RUNS_QUERY_ROOT })
  }

  const archiveMutation = useMutation({
    mutation: (id: string) => runsApi.archiveRoutine(id),
    onSuccess: () => toast.success('Run archived'),
    onError: (err) => toast.error(resolveErrorMessage(err, 'Failed to archive run')),
    onSettled: invalidate,
  })

  const unarchiveMutation = useMutation({
    mutation: (id: string) => runsApi.unarchiveRoutine(id),
    onSuccess: () => toast.success('Run unarchived'),
    onError: (err) => toast.error(resolveErrorMessage(err, 'Failed to unarchive run')),
    onSettled: invalidate,
  })

  const cancelMutation = useMutation({
    mutation: (id: string) => runsApi.cancelRoutine(id),
    onSuccess: () => toast.success('Run cancelled'),
    onError: (err) => toast.error(resolveErrorMessage(err, 'Failed to cancel run')),
    onSettled: invalidate,
  })

  const removeMutation = useMutation({
    mutation: (id: string) => runsApi.deleteRoutine(id),
    onSuccess: () => toast.success('Run deleted'),
    onError: (err) => toast.error(resolveErrorMessage(err, 'Failed to delete run')),
    onSettled: invalidate,
  })

  return {
    // ['routines', { archived }] query surface
    runs,
    runsState: query.state,
    isLoading: query.isLoading,
    error: query.error,
    refetch: query.refetch,
    archived,

    // live-polling status, for the "live" indicator
    isPolling,

    // mutations — `mutateAsync` rethrows so callers keep their own
    // try/catch (e.g. clearing a row-local "pending" id in `finally`).
    archiveRun: archiveMutation.mutateAsync,
    isArchiving: archiveMutation.isLoading,
    unarchiveRun: unarchiveMutation.mutateAsync,
    isUnarchiving: unarchiveMutation.isLoading,
    cancelRun: cancelMutation.mutateAsync,
    isCancelling: cancelMutation.isLoading,
    removeRun: removeMutation.mutateAsync,
    isRemoving: removeMutation.isLoading,
  }
})
