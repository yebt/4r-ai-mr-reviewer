/**
 * Runs list filters — repo + status `Select` options and a pure client-side
 * filter, mirroring `modules/reviews/filters.ts`'s pure-function shape so the
 * filtering logic is unit-testable without driving Reka UI's `Select`
 * primitive through jsdom (portals + internal state make that brittle).
 */
import { runStatusUi } from './format'
import type { RoutineRun, RoutineRunStatus } from './types'

export const ALL_REPOS_VALUE = 'all'
export const ALL_STATUSES_VALUE = 'all'

export interface RunFilterOption {
  label: string
  value: string
}

/** Every run status, in `format.ts#runStatusUi` declaration order, as Select items — "All statuses" first. */
export const RUN_STATUS_FILTER_OPTIONS: RunFilterOption[] = [
  { label: 'All statuses', value: ALL_STATUSES_VALUE },
  ...(Object.keys(runStatusUi) as RoutineRunStatus[]).map((status) => ({
    label: runStatusUi[status].label,
    value: status,
  })),
]

export interface RepoNameSource {
  id: string
  name: string
}

function distinctRunRepos(runs: RoutineRun[]): RunFilterOption[] {
  const seen = new Map<string, string>()
  for (const run of runs) {
    if (!seen.has(run.repoId)) seen.set(run.repoId, run.repoName ?? run.repoId)
  }
  return [...seen.entries()].map(([value, label]) => ({ value, label }))
}

/**
 * Repo `Select` options, "All repositories" first. Prefers the full
 * connected-repos list (`useReposStore().repos` — every repo, not just ones
 * with a run today) when it's loaded, falling back to the distinct repos
 * actually present in `runs` (e.g. before the repos store has fetched, or if
 * it's empty).
 */
export function repoFilterOptions(runs: RoutineRun[], repos: RepoNameSource[]): RunFilterOption[] {
  const source =
    repos.length > 0 ? repos.map((repo) => ({ label: repo.name, value: repo.id })) : distinctRunRepos(runs)
  return [
    { label: 'All repositories', value: ALL_REPOS_VALUE },
    ...[...source].sort((a, b) => a.label.localeCompare(b.label)),
  ]
}

export interface RunFilters {
  repoId: string
  status: string
}

/** Pure client-side filter of the (already fetched) runs list by repo + status. `'all'` on either matches everything. */
export function filterRuns(runs: RoutineRun[], filters: RunFilters): RoutineRun[] {
  return runs.filter((run) => {
    if (filters.repoId !== ALL_REPOS_VALUE && run.repoId !== filters.repoId) return false
    if (filters.status !== ALL_STATUSES_VALUE && run.status !== filters.status) return false
    return true
  })
}
