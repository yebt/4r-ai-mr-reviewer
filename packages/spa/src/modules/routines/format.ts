import type {
  RoutineRun,
  RoutineRunStatus,
  RoutineStepName,
  RoutineStepStatus,
} from '@shared/api/types'

// A run is "active" while it is still progressing on the server, so the UI must
// keep polling it. Pure and side-effect free so it can drive both the polling
// loop and its tests. `blocked` and `done` are the resting states.
export function isRunActive(status: RoutineRunStatus): boolean {
  return status === 'pending' || status === 'running'
}

// Visual mapping for a run's overall status. `spin` marks the icon that should
// animate (the running spinner); `label` is shown as text so state is never
// conveyed by color alone.
export interface RunStatusUi {
  icon: string
  class: string
  label: string
  spin: boolean
}

export const runStatusUi: Record<RoutineRunStatus, RunStatusUi> = {
  pending: { icon: 'i-lucide-circle-dashed', class: 'text-muted', label: 'Pending', spin: false },
  running: { icon: 'i-lucide-loader-circle', class: 'text-accent', label: 'Running', spin: true },
  blocked: { icon: 'i-lucide-triangle-alert', class: 'text-warn', label: 'Blocked', spin: false },
  done: { icon: 'i-lucide-check', class: 'text-ok', label: 'Done', spin: false },
}

// Visual mapping for a single step's status. Same contract as RunStatusUi.
export interface StepStatusUi {
  icon: string
  class: string
  label: string
  spin: boolean
}

export const stepStatusUi: Record<RoutineStepStatus, StepStatusUi> = {
  pending: { icon: 'i-lucide-circle', class: 'text-muted', label: 'Pending', spin: false },
  running: { icon: 'i-lucide-loader-circle', class: 'text-accent', label: 'Running', spin: true },
  done: { icon: 'i-lucide-check', class: 'text-ok', label: 'Done', spin: false },
  failed: { icon: 'i-lucide-x', class: 'text-danger', label: 'Failed', spin: false },
  skipped: { icon: 'i-lucide-minus', class: 'text-muted', label: 'Skipped', spin: false },
}

export const routineKindLabel: Record<RoutineRun['kind'], string> = {
  approve_and_tag: 'Approve & tag',
}

export const stepLabel: Record<RoutineStepName, string> = {
  react: 'React',
  comment: 'Comment',
  tag: 'Tag',
}

// Compact date/time formatter for run rows. Self-contained so the routines
// module stays independent of other feature modules.
export function formatDateTime(iso: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  return d.toLocaleString(undefined, {
    month: 'short',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}
