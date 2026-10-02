import { computed, toValue, watch, type MaybeRefOrGetter } from 'vue'
import { useMutation, useQuery, useQueryCache } from '@pinia/colada'
import { useDocumentVisibility, useIntervalFn } from '@vueuse/core'
import { useToast } from '@shared/composables/useToast'
import { resolveErrorMessage } from '@shared/api/resolveErrorMessage'
import * as runsApi from '../api'
import { isRunActive } from '../format'
import { RUNS_QUERY_ROOT } from '../store'
import type { RoutineConfirmDecision } from '../types'

const POLL_INTERVAL_MS = 2500

/**
 * Pure query-key builder for a single run, exported so tests can assert its
 * exact shape without reaching into the composable's internals.
 */
export function runDetailQueryKey(id: string): readonly [string, string] {
  return ['routine', id]
}

/**
 * Run detail — one run's `GET /routines/{id}` query plus its 4 status-gated
 * actions (resume/skip/confirm/cancel). Page-local composable rather than a
 * Pinia store: unlike `useRunsStore` (one global list, one instance for the
 * whole app), a run detail is scoped to whichever id the route currently
 * holds, so it's called straight from `src/pages/runs/[id].vue`'s setup —
 * @pinia/colada's `useQuery`/`useMutation` only need an active Pinia Colada
 * plugin context, not a `defineStore` wrapper.
 *
 * Live polling mirrors `useRunsStore`'s leak-safe pattern exactly:
 * `useIntervalFn` refetches every 2.5s, gated by `isRunActive(run.status)` AND
 * the document being visible (`useDocumentVisibility`), driven by an
 * `{ immediate: true }` watch so an already-active run starts polling on
 * load. `pausePolling` is exposed for the page to call on `onUnmounted` —
 * navigating away must stop the interval, since `useIntervalFn`'s own
 * cleanup only fires when the composable's own owner (the page) unmounts,
 * and by then the interval may already have fired once more.
 */
export function useRunDetail(id: MaybeRefOrGetter<string>) {
  const queryCache = useQueryCache()
  const toast = useToast()

  const query = useQuery({
    key: () => runDetailQueryKey(toValue(id)),
    query: () => runsApi.getRoutine(toValue(id)),
  })

  const run = computed(() => query.data.value ?? null)
  const isActive = computed(() => (run.value ? isRunActive(run.value.status) : false))

  const documentVisibility = useDocumentVisibility()
  const canPoll = computed(() => isActive.value && documentVisibility.value === 'visible')

  const { pause, resume, isActive: isPolling } = useIntervalFn(() => query.refetch(), POLL_INTERVAL_MS, {
    immediate: false,
  })

  watch(canPoll, (active) => (active ? resume() : pause()), { immediate: true })

  function invalidate() {
    queryCache.invalidateQueries({ key: runDetailQueryKey(toValue(id)) })
    queryCache.invalidateQueries({ key: RUNS_QUERY_ROOT })
  }

  const resumeMutation = useMutation({
    mutation: () => runsApi.resumeRoutine(toValue(id)),
    onSuccess: () => toast.success('Run resumed'),
    onError: (err) => toast.error(resolveErrorMessage(err, 'Failed to resume run')),
    onSettled: invalidate,
  })

  const skipMutation = useMutation({
    mutation: () => runsApi.skipRoutine(toValue(id)),
    onSuccess: () => toast.success('Step skipped'),
    onError: (err) => toast.error(resolveErrorMessage(err, 'Failed to skip step')),
    onSettled: invalidate,
  })

  const confirmMutation = useMutation({
    mutation: (decision: RoutineConfirmDecision) => runsApi.confirmRoutine(toValue(id), decision),
    onSuccess: () => toast.success('Confirmation recorded'),
    onError: (err) => toast.error(resolveErrorMessage(err, 'Failed to confirm run')),
    onSettled: invalidate,
  })

  const cancelMutation = useMutation({
    mutation: () => runsApi.cancelRoutine(toValue(id)),
    onSuccess: () => toast.success('Run cancelled'),
    onError: (err) => toast.error(resolveErrorMessage(err, 'Failed to cancel run')),
    onSettled: invalidate,
  })

  return {
    // ['routine', id] query surface
    run,
    state: query.state,
    isLoading: query.isLoading,
    error: query.error,
    refetch: query.refetch,

    // live-polling status + the manual pause the detail page calls on unmount.
    isPolling,
    pausePolling: pause,

    // mutations — `mutateAsync` rethrows so the page keeps its own try/catch.
    resumeRun: resumeMutation.mutateAsync,
    isResuming: resumeMutation.isLoading,
    skipRun: skipMutation.mutateAsync,
    isSkipping: skipMutation.isLoading,
    confirmRun: confirmMutation.mutateAsync,
    isConfirming: confirmMutation.isLoading,
    cancelRun: cancelMutation.mutateAsync,
    isCancelling: cancelMutation.isLoading,
  }
}
