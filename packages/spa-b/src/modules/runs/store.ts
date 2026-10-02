import { computed, ref, watch } from 'vue'
import { defineStore } from 'pinia'
import { useDocumentVisibility, useIntervalFn } from '@vueuse/core'
import { useToast } from '@shared/composables/useToast'
import { resolveErrorMessage } from '@shared/api/resolveErrorMessage'
import { useInfiniteList } from '@shared/data/useInfiniteList'
import * as runsApi from './api'
import { isRunActive } from './format'
import type { RoutineRun } from './types'

export { resolveErrorMessage }

const POLL_INTERVAL_MS = 2500

/**
 * Kept for `composables/useRunDetail.ts`, which still invalidates this key
 * via `@pinia/colada`'s `useQueryCache()` after a detail-page mutation
 * (resume/skip/confirm/cancel) settles. The runs list itself no longer runs
 * through `@pinia/colada` (see `useRunsStore` below, backed by
 * `useInfiniteList` instead) — that invalidation call is now an inert no-op
 * against this list, and the list still self-updates via its own mutations
 * and polling. Not touching the detail composable's cross-module contract
 * here; only the constant it imports.
 */
export const RUNS_QUERY_ROOT = ['routines'] as const

/**
 * Pure gate for the polling loop: true while at least one currently-loaded
 * run is non-terminal (see `format.ts#isRunActive`). Kept standalone so it
 * can be unit-tested as a plain function instead of asserting on the
 * interval timer itself.
 */
export function shouldPoll(runs: RoutineRun[]): boolean {
  return runs.some((run) => isRunActive(run.status))
}

/**
 * Runs store — the global routine-run list (every repo, newest first),
 * cursor-paginated via `useInfiniteList` (see `@shared/data/useInfiniteList`)
 * over `runsApi.listRecentRoutines`. Not a `createCrudResource` — this
 * resource has no `create` and its mutations are custom action endpoints
 * (archive/unarchive/cancel/remove) rather than the generic create/update/
 * remove shape — so the list + mutations are hand-rolled here.
 *
 * `archived` is a plain ref toggle: flipping it resets the accumulated list
 * and reloads its first page from scratch, since active/archived are
 * disjoint result sets, not different pages of the same one.
 *
 * Live polling: `useIntervalFn` calls `list.refreshFirstPage()` every 2.5s
 * (a *merge* into the accumulated list by id — see `useInfiniteList` — so an
 * active run's status updates in place and brand-new runs appear at the top
 * WITHOUT dropping already-loaded older pages or resetting scroll), started
 * only while `runs.some(isRunActive)` AND the document is visible
 * (`useDocumentVisibility` — a backgrounded tab has no reason to keep
 * hitting the API every 2.5s) — a `watch` on that combined `canPoll` boolean
 * (immediate, so a page load that already has an active run starts polling
 * right away) drives `resume()`/`pause()`. `pausePolling` is exposed so a
 * consuming component can stop the interval on `onUnmounted` — `useIntervalFn`'s
 * own cleanup only fires when the *component that called it* unmounts, which
 * is this store's setup function, not necessarily the page that renders it
 * (Pinia stores outlive the component that first instantiated them).
 */
export const useRunsStore = defineStore('runs', () => {
  const toast = useToast()

  const archived = ref(false)

  const list = useInfiniteList<RoutineRun>((cursor) => runsApi.listRecentRoutines(cursor, archived.value))

  const runs = computed(() => list.items.value)

  // Normalizes `list.error` (`unknown`, since `useInfiniteList` is generic
  // and catches any thrown value) down to `Error | null` — every consumer
  // (`RunsListSection.vue`, `src/pages/index.vue`) reads `.message` off it,
  // and the API layer only ever throws `ApiError extends Error` anyway.
  const error = computed<Error | null>(() => (list.error.value instanceof Error ? list.error.value : null))

  // `{ status }` shape kept for `src/pages/index.vue`, an out-of-scope
  // consumer that still branches on `runsState.status` the way `@pinia/
  // colada`'s `query.state` used to — this store no longer runs through
  // colada (see the class doc above), so it's reconstructed here instead of
  // changing that page's contract.
  const runsState = computed<{ status: 'pending' | 'error' | 'success' }>(() => {
    if (error.value) return { status: 'error' }
    if (list.isLoading.value) return { status: 'pending' }
    return { status: 'success' }
  })

  const hasActiveRun = computed(() => shouldPoll(runs.value))

  const documentVisibility = useDocumentVisibility()
  const canPoll = computed(() => hasActiveRun.value && documentVisibility.value === 'visible')

  const { pause, resume, isActive: isPolling } = useIntervalFn(() => list.refreshFirstPage(), POLL_INTERVAL_MS, {
    immediate: false,
  })

  watch(canPoll, (active) => (active ? resume() : pause()), { immediate: true })

  // Archived is a disjoint result set from active, not a further page of it —
  // toggling it starts the accumulated list over from its own first page.
  watch(archived, () => {
    list.reset()
    void list.loadInitial()
  })

  void list.loadInitial()

  /** Reset + reload from the first page — used by the section's error-retry button. */
  function refetch(): Promise<void> {
    list.reset()
    return list.loadInitial()
  }

  /** Drops a run from the accumulated list by id, for mutations that change list membership (archive/unarchive/delete). */
  function removeRunLocally(id: string): void {
    list.items.value = list.items.value.filter((run) => run.id !== id)
  }

  const isArchiving = ref(false)
  async function archiveRun(id: string): Promise<void> {
    isArchiving.value = true
    try {
      await runsApi.archiveRoutine(id)
      removeRunLocally(id)
      toast.success('Run archived')
    } catch (err) {
      toast.error(resolveErrorMessage(err, 'Failed to archive run'))
      throw err
    } finally {
      isArchiving.value = false
    }
  }

  const isUnarchiving = ref(false)
  async function unarchiveRun(id: string): Promise<void> {
    isUnarchiving.value = true
    try {
      await runsApi.unarchiveRoutine(id)
      removeRunLocally(id)
      toast.success('Run unarchived')
    } catch (err) {
      toast.error(resolveErrorMessage(err, 'Failed to unarchive run'))
      throw err
    } finally {
      isUnarchiving.value = false
    }
  }

  const isCancelling = ref(false)
  async function cancelRun(id: string): Promise<void> {
    isCancelling.value = true
    try {
      await runsApi.cancelRoutine(id)
      toast.success('Run cancelled')
      // Cancelling keeps the run in view (just changes its status) — merge
      // the refreshed first page so the row picks up its new status in place.
      await list.refreshFirstPage()
    } catch (err) {
      toast.error(resolveErrorMessage(err, 'Failed to cancel run'))
      throw err
    } finally {
      isCancelling.value = false
    }
  }

  const isRemoving = ref(false)
  async function removeRun(id: string): Promise<void> {
    isRemoving.value = true
    try {
      await runsApi.deleteRoutine(id)
      removeRunLocally(id)
      toast.success('Run deleted')
    } catch (err) {
      toast.error(resolveErrorMessage(err, 'Failed to delete run'))
      throw err
    } finally {
      isRemoving.value = false
    }
  }

  return {
    // Infinite-list surface backing the runs list.
    runs,
    runsState,
    isLoading: list.isLoading,
    isLoadingMore: list.isLoadingMore,
    hasMore: list.hasMore,
    error,
    loadMore: list.loadMore,
    refetch,
    archived,

    // live-polling status, for the "live" indicator, and a manual pause the
    // list page calls on unmount so navigating away actually stops the poll.
    isPolling,
    pausePolling: pause,

    // mutations — each rethrows so callers keep their own try/catch (e.g.
    // clearing a row-local "pending" id in `finally`); the store already
    // toasted the error before rethrowing.
    archiveRun,
    isArchiving,
    unarchiveRun,
    isUnarchiving,
    cancelRun,
    isCancelling,
    removeRun,
    isRemoving,
  }
})
