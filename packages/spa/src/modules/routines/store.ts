import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { api, errorMessage } from '@shared/api/client'
import type {
  ConfirmDecision,
  MainReleaseInput,
  ReleaseInput,
  RoutineRun,
} from '@shared/api/types'

export const useRoutinesStore = defineStore('routines', () => {
  // Runs cached per repo (newest first, as the backend returns them) so
  // revisiting a repo shows its previous runs immediately.
  const runsByRepo = ref<Record<string, RoutineRun[]>>({})
  // Run-detail cache keyed by run id, kept in sync with the per-repo lists so
  // the live step ledger and the runs list observe the same object.
  const runsById = ref<Record<string, RoutineRun>>({})
  const listLoading = ref(false)
  const listError = ref<string | null>(null)
  // Ordered ids of the global "recent runs" list (newest first). The runs
  // themselves live in runsById, so the live poller's refresh() updates the same
  // objects the recent list observes.
  const recentRunIds = ref<string[]>([])

  function runsFor(repoId: string): RoutineRun[] {
    return runsByRepo.value[repoId] ?? []
  }

  // recentRuns resolves the ordered recent ids against runsById so the list stays
  // in sync with polling refreshes (which upsert runsById).
  const recentRuns = computed<RoutineRun[]>(() =>
    recentRunIds.value
      .map((id) => runsById.value[id])
      .filter((r): r is RoutineRun => r != null),
  )

  function runById(id: string): RoutineRun | null {
    return runsById.value[id] ?? null
  }

  // cacheRun upserts a run into both caches, prepending it to its repo list when
  // it is new (so a freshly created run appears at the top).
  function cacheRun(run: RoutineRun) {
    runsById.value[run.id] = run
    const list = runsByRepo.value[run.repoId]
    if (list) {
      const exists = list.some((r) => r.id === run.id)
      runsByRepo.value = {
        ...runsByRepo.value,
        [run.repoId]: exists ? list.map((r) => (r.id === run.id ? run : r)) : [run, ...list],
      }
    } else {
      runsByRepo.value = { ...runsByRepo.value, [run.repoId]: [run] }
    }
  }

  async function list(repoId: string) {
    listLoading.value = true
    listError.value = null
    try {
      const data = await api.listRepoRoutines(repoId)
      runsByRepo.value = { ...runsByRepo.value, [repoId]: data }
      for (const run of data) runsById.value[run.id] = run
    } catch (e) {
      listError.value = errorMessage(e)
    } finally {
      listLoading.value = false
    }
  }

  // listRecent loads the global recent-runs list (across all repos). It stores
  // the runs in runsById and records their order in recentRunIds, so the recent
  // list and the live poller share the same run objects.
  async function listRecent(limit?: number) {
    listLoading.value = true
    listError.value = null
    try {
      const data = await api.listRecentRoutines(limit)
      recentRunIds.value = data.map((r) => r.id)
      for (const run of data) runsById.value[run.id] = run
    } catch (e) {
      listError.value = errorMessage(e)
    } finally {
      listLoading.value = false
    }
  }

  // createRelease starts a dev-flow release run for a merge request whose target
  // is `development`. Optional fields are only sent when meaningful so the
  // backend applies its own defaults. Rethrows so callers can surface a 400 (bad
  // target) / 409 (duplicate active run) via a toast.
  async function createRelease(repoId: string, input: ReleaseInput) {
    const body: ReleaseInput = { mrIid: input.mrIid }
    if (input.bump) body.bump = input.bump
    if (input.emojis && input.emojis.length > 0) body.emojis = input.emojis
    if (input.removeSourceBranch != null) body.removeSourceBranch = input.removeSourceBranch
    if (input.mergeWhenPipelineSucceeds != null)
      body.mergeWhenPipelineSucceeds = input.mergeWhenPipelineSucceeds
    const created = await api.createRelease(repoId, body)
    cacheRun(created)
    return created
  }

  // createMainRelease starts a main-flow release run. Branch names are only sent
  // when non-empty so the backend applies its defaults (development → main).
  async function createMainRelease(repoId: string, input: MainReleaseInput) {
    const body: MainReleaseInput = {}
    if (input.bump) body.bump = input.bump
    const source = input.sourceBranch?.trim()
    if (source) body.sourceBranch = source
    const target = input.targetBranch?.trim()
    if (target) body.targetBranch = target
    if (input.emojis && input.emojis.length > 0) body.emojis = input.emojis
    if (input.removeSourceBranch != null) body.removeSourceBranch = input.removeSourceBranch
    if (input.mergeWhenPipelineSucceeds != null)
      body.mergeWhenPipelineSucceeds = input.mergeWhenPipelineSucceeds
    const created = await api.createMainRelease(repoId, body)
    cacheRun(created)
    return created
  }

  // refresh silently pulls the latest run state (used while polling a live run).
  async function refresh(id: string) {
    const run = await api.getRoutineRun(id)
    cacheRun(run)
    return run
  }

  // resume re-drives a blocked run. Rethrows (409 when the run is not blocked) so
  // the caller can toast.
  async function resume(id: string) {
    const run = await api.resumeRoutine(id)
    cacheRun(run)
    return run
  }

  // skip drops a blocked run's failed (non-critical) step and re-queues it.
  // Rethrows (409 when the step is essential / the run is not blocked) so the
  // caller can toast.
  async function skip(id: string) {
    const run = await api.skipRoutine(id)
    cacheRun(run)
    return run
  }

  // confirm answers a release run's confirmation gate. The run flips back to
  // running (decision 'merge') or continues to its resting state, so the caller
  // restarts polling. Rethrows (409 when the run is not awaiting_confirmation).
  async function confirm(id: string, decision: ConfirmDecision) {
    const run = await api.confirmRoutine(id, decision)
    cacheRun(run)
    return run
  }

  // cancel aborts a run on the server and upserts the returned (cancelled) run
  // into every cache, so all views observe the terminal state at once. Rethrows
  // (404 unknown, 409 already terminal) so the caller can toast.
  async function cancel(id: string) {
    const run = await api.cancelRoutine(id)
    cacheRun(run)
    return run
  }

  // remove deletes a run on the server, then drops it from every cache it lives
  // in (the by-id map, each per-repo list, and the recent-runs order) so all
  // views forget it at once. Rethrows (404 unknown, 409 running) so the caller
  // can toast; the caches are only pruned after the delete succeeds.
  async function remove(id: string) {
    await api.deleteRoutine(id)
    const { [id]: _removed, ...restById } = runsById.value
    runsById.value = restById
    const nextByRepo: Record<string, RoutineRun[]> = {}
    for (const [repoId, list] of Object.entries(runsByRepo.value)) {
      nextByRepo[repoId] = list.filter((r) => r.id !== id)
    }
    runsByRepo.value = nextByRepo
    recentRunIds.value = recentRunIds.value.filter((rid) => rid !== id)
  }

  return {
    runsByRepo,
    runsById,
    listLoading,
    listError,
    recentRunIds,
    recentRuns,
    runsFor,
    runById,
    list,
    listRecent,
    createRelease,
    createMainRelease,
    refresh,
    resume,
    skip,
    confirm,
    cancel,
    remove,
  }
})
