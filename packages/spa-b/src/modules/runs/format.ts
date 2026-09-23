/**
 * Pure, side-effect-free formatting/status helpers for the runs list — ported
 * from the old app's `packages/spa/src/modules/routines/format.ts` and
 * adapted to this app's `Badge`/`Icon` vocabulary (a `BadgeStatus` instead of
 * a bespoke `class`/`icon` pair). Kept pure so they can drive both the
 * template and the polling-gate logic, and be unit-tested directly.
 */
import type { BadgeStatus } from '@shared/ui/design-system'
import type { RoutineKind, RoutineRun, RoutineRunStatus, RoutineStepStatus } from './types'

/**
 * A run is "active" for THIS list page while it is still progressing on the
 * server *or* waiting on a human to unblock it — pending/running (mid-flight)
 * plus blocked/awaiting_confirmation (resting, but can flip the moment
 * someone acts elsewhere). All four are non-terminal, so the live-polling
 * gate (`useRunsStore`) keeps refetching while any row is in one of them;
 * only `done`/`cancelled` stop it. This is intentionally broader than the
 * old app's `isRunActive`, which only covered pending/running — that one
 * gated a single run's own detail poller, not a list-wide "is anything here
 * still moving" indicator.
 */
export function isRunActive(status: RoutineRunStatus): boolean {
  return status === 'pending' || status === 'running' || status === 'blocked' || status === 'awaiting_confirmation'
}

/**
 * A run is cancelable while it has not reached a terminal state (`done` or
 * `cancelled`): pending/running are aborted mid-flight, and blocked/
 * awaiting_confirmation are aborted from their resting pause.
 */
export function isRunCancelable(status: RoutineRunStatus): boolean {
  return status !== 'done' && status !== 'cancelled'
}

/** Visual mapping for a run's overall status, in this app's Badge/Icon vocabulary. */
export interface RunStatusUi {
  label: string
  variant: BadgeStatus
  icon: string
  spin: boolean
}

export const runStatusUi: Record<RoutineRunStatus, RunStatusUi> = {
  pending: { label: 'Pending', variant: 'neutral', icon: 'circle-dashed', spin: false },
  running: { label: 'Running', variant: 'info', icon: 'loader-circle', spin: true },
  blocked: { label: 'Blocked', variant: 'danger', icon: 'triangle-alert', spin: false },
  awaiting_confirmation: {
    label: 'Awaiting confirmation',
    variant: 'warning',
    icon: 'circle-pause',
    spin: false,
  },
  done: { label: 'Done', variant: 'success', icon: 'check', spin: false },
  cancelled: { label: 'Cancelled', variant: 'neutral', icon: 'ban', spin: false },
}

/** Visual mapping for a single step's status. Same contract as `RunStatusUi`. */
export interface StepStatusUi {
  label: string
  variant: BadgeStatus
  icon: string
  spin: boolean
}

export const stepStatusUi: Record<RoutineStepStatus, StepStatusUi> = {
  pending: { label: 'Pending', variant: 'neutral', icon: 'circle', spin: false },
  running: { label: 'Running', variant: 'info', icon: 'loader-circle', spin: true },
  done: { label: 'Done', variant: 'success', icon: 'check', spin: false },
  failed: { label: 'Failed', variant: 'danger', icon: 'x', spin: false },
  skipped: { label: 'Skipped', variant: 'neutral', icon: 'minus', spin: false },
}

export const routineKindLabel: Record<RoutineKind, string> = {
  release: 'Release',
  approve_and_tag: 'Approve & tag',
}

/**
 * Primary human label for a run: the merge-request title once captured (the
 * row's prominent identifier), otherwise the `!{mrIid}` fallback for a run
 * that just started or predates title capture, and finally a generic label
 * when neither is present.
 */
export function runTitle(run: RoutineRun): string {
  const mrTitle = run.state.mrTitle
  if (typeof mrTitle === 'string' && mrTitle) return mrTitle
  if (run.mrIid) return `!${run.mrIid}`
  return 'Routine run'
}

/** Human label for a run's release flow, or null when absent/unrecognized. */
export function flowLabel(flow: string | undefined): string | null {
  if (flow === 'main') return 'Main release'
  if (flow === 'development') return 'Dev release'
  return null
}

/** Compact date/time formatter for run rows. */
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
