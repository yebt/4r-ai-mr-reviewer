import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api, errorMessage } from '@shared/api/client'
import type { ApproveAndTagInput, RoutineRun } from '@shared/api/types'

export const useRoutinesStore = defineStore('routines', () => {
  // Runs cached per repo (newest first, as the backend returns them) so
  // revisiting a repo shows its previous runs immediately.
  const runsByRepo = ref<Record<string, RoutineRun[]>>({})
  // Run-detail cache keyed by run id, kept in sync with the per-repo lists so
  // the live step ledger and the runs list observe the same object.
  const runsById = ref<Record<string, RoutineRun>>({})
  const listLoading = ref(false)
  const listError = ref<string | null>(null)

  function runsFor(repoId: string): RoutineRun[] {
    return runsByRepo.value[repoId] ?? []
  }

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

  // create starts an approve_and_tag run. Optional fields are only sent when
  // meaningful so the backend applies its own defaults for the rest. Rethrows so
  // callers can surface a 409 (duplicate active run) / 400 via a toast.
  async function create(repoId: string, input: ApproveAndTagInput) {
    const body: ApproveAndTagInput = { mrIid: input.mrIid }
    if (input.bump) body.bump = input.bump
    const comment = input.comment?.trim()
    if (comment) body.comment = comment
    if (input.emojis && input.emojis.length > 0) body.emojis = input.emojis
    const created = await api.createApproveAndTag(repoId, body)
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

  return {
    runsByRepo,
    runsById,
    listLoading,
    listError,
    runsFor,
    runById,
    list,
    create,
    refresh,
    resume,
  }
})
