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

/** One notable `state` key rendered in the detail page's summary. */
export interface StateSummaryEntry {
  key: string
  label: string
  value: string
}

/**
 * Notable `state` keys the detail page knows how to label, in the order
 * they should render. `state` is a free-form per-run scratch bag (see
 * `types.ts#RoutineRun`), so this is best-effort: any key that's absent,
 * `null`, or an empty string is silently skipped rather than rendered blank.
 */
const STATE_SUMMARY_FIELDS: { key: string; label: string }[] = [
  { key: 'lastTag', label: 'Last tag' },
  { key: 'nextTag', label: 'Next tag' },
  { key: 'featCount', label: 'Features' },
  { key: 'fixCount', label: 'Fixes' },
  { key: 'decision', label: 'Decision' },
]

/** Builds the detail page's best-effort `state` summary — see `STATE_SUMMARY_FIELDS`. */
export function stateSummaryEntries(run: RoutineRun): StateSummaryEntry[] {
  const entries: StateSummaryEntry[] = []
  for (const field of STATE_SUMMARY_FIELDS) {
    const value = field.key === 'nextTag' ? releaseTag(run) : run.state[field.key]
    if (value === undefined || value === null || value === '') continue
    entries.push({ key: field.key, label: field.label, value: String(value) })
  }
  return entries
}

// `Intl.DateTimeFormat` construction is the expensive part of formatting a
// date (it resolves the runtime's locale data), not the eventual `.format()`
// call — building one per `formatDateTime` call measured at ~10ms of main-
// thread time across a warm runs-page navigation (30+ rows). Cache one
// module-level instance and reuse it; `undefined` locale + these options
// never change, so there is nothing to invalidate.
const runDateTimeFormatter = new Intl.DateTimeFormat(undefined, {
  month: 'short',
  day: '2-digit',
  hour: '2-digit',
  minute: '2-digit',
})

/** Compact date/time formatter for run rows. */
export function formatDateTime(iso: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  return runDateTimeFormatter.format(d)
}

const runTimeFormatter = new Intl.DateTimeFormat(undefined, { hour: '2-digit', minute: '2-digit' })

/** Time-of-day only (e.g. step completion times inside one run). */
export function formatTime(iso: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  return runTimeFormatter.format(d)
}

// Readable names for the backend's routine step keys (see the step ledgers in
// packages/server/internal/app/routines/service.go). Unknown keys fall back to
// sentence case so a new backend step still renders sensibly.
const STEP_LABELS: Record<string, string> = {
  verify: 'Verify merge request',
  react: 'Add reactions',
  comment: 'Comment',
  approve: 'Approve merge request',
  compute_tag: 'Compute tag',
  create_mr: 'Open merge request',
  wait_pipeline: 'Wait for pipeline',
  confirm: 'Confirmation',
  merge: 'Merge',
  tag: 'Push tag',
  notify: 'Notify',
}

export function stepLabel(name: string): string {
  if (!name) return ''
  const known = STEP_LABELS[name]
  if (known) return known
  const words = name.replace(/[_-]+/g, ' ').trim()
  return words.charAt(0).toUpperCase() + words.slice(1)
}

/**
 * The merge request a run acts on: `run.mrIid` for the dev flow (the MR
 * exists when the run starts), else the MR the main flow created mid-run and
 * recorded in `state.mrIid` (server `effectiveMRIID`). `null` before the main
 * flow has created it.
 */
export function runMergeRequestIid(run: RoutineRun): number | null {
  if (run.mrIid > 0) return run.mrIid
  const created = run.state.mrIid
  return typeof created === 'number' && created > 0 ? created : null
}

/** GitLab's web URL for merge request `iid` of the project at `repoUrl`; `null` without a repo URL. */
export function mergeRequestUrl(repoUrl: string, iid: number): string | null {
  if (!repoUrl) return null
  return `${repoUrl.replace(/\/+$/, '')}/-/merge_requests/${iid}`
}

/**
 * The tag a run pushes. The server stores the computed version bare in
 * `state.nextTag` and appends the "-dev" prerelease suffix only when tagging
 * a release run outside the main flow (server `tagSuffix`, which also treats
 * a release run with no flow as the dev flow). `null` before it's computed.
 */
export function releaseTag(run: RoutineRun): string | null {
  const next = run.state.nextTag
  if (typeof next !== 'string' || !next) return null
  return run.kind === 'release' && run.flow !== 'main' ? `${next}-dev` : next
}
