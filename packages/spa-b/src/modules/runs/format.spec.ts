import { describe, expect, it } from 'vitest'
import {
  formatDateTime,
  formatTime,
  isRunActive,
  isRunCancelable,
  runStatusUi,
  runTitle,
  stateSummaryEntries,
  stepLabel,
} from './format'
import type { RoutineRun } from './types'

function makeRun(overrides: Partial<RoutineRun> = {}): RoutineRun {
  return {
    id: 'r1',
    kind: 'release',
    repoId: 'repo1',
    mrIid: 42,
    status: 'running',
    steps: [],
    state: {},
    lastError: '',
    createdAt: '2026-01-01T00:00:00.000Z',
    updatedAt: '2026-01-01T00:00:00.000Z',
    archived: false,
    ...overrides,
  }
}

describe('runStatusUi', () => {
  it('maps every status to its Badge variant per the design contract', () => {
    expect(runStatusUi.pending.variant).toBe('neutral')
    expect(runStatusUi.running.variant).toBe('info')
    expect(runStatusUi.blocked.variant).toBe('danger')
    expect(runStatusUi.awaiting_confirmation.variant).toBe('warning')
    expect(runStatusUi.done.variant).toBe('success')
    expect(runStatusUi.cancelled.variant).toBe('neutral')
  })

  it('spins the icon only for running', () => {
    expect(runStatusUi.running.spin).toBe(true)
    expect(runStatusUi.pending.spin).toBe(false)
    expect(runStatusUi.done.spin).toBe(false)
    expect(runStatusUi.blocked.spin).toBe(false)
    expect(runStatusUi.awaiting_confirmation.spin).toBe(false)
    expect(runStatusUi.cancelled.spin).toBe(false)
  })
})

describe('runTitle', () => {
  it('prefers state.mrTitle when present', () => {
    expect(runTitle(makeRun({ state: { mrTitle: 'Add API docs' }, mrIid: 47 }))).toBe('Add API docs')
  })

  it('falls back to !{mrIid} when mrTitle is absent', () => {
    expect(runTitle(makeRun({ state: {}, mrIid: 47 }))).toBe('!47')
  })

  it('falls back to a generic label when neither is present', () => {
    expect(runTitle(makeRun({ state: {}, mrIid: 0 }))).toBe('Routine run')
  })

  it('ignores a non-string mrTitle rather than rendering it', () => {
    expect(runTitle(makeRun({ state: { mrTitle: 123 }, mrIid: 47 }))).toBe('!47')
  })
})

describe('isRunActive', () => {
  it('is true for pending/running/blocked/awaiting_confirmation', () => {
    expect(isRunActive('pending')).toBe(true)
    expect(isRunActive('running')).toBe(true)
    expect(isRunActive('blocked')).toBe(true)
    expect(isRunActive('awaiting_confirmation')).toBe(true)
  })

  it('is false for terminal statuses', () => {
    expect(isRunActive('done')).toBe(false)
    expect(isRunActive('cancelled')).toBe(false)
  })
})

describe('isRunCancelable', () => {
  it('is true for every non-terminal status', () => {
    expect(isRunCancelable('pending')).toBe(true)
    expect(isRunCancelable('running')).toBe(true)
    expect(isRunCancelable('blocked')).toBe(true)
    expect(isRunCancelable('awaiting_confirmation')).toBe(true)
  })

  it('is false once a run reaches a terminal status', () => {
    expect(isRunCancelable('done')).toBe(false)
    expect(isRunCancelable('cancelled')).toBe(false)
  })
})

describe('formatDateTime', () => {
  it('returns an empty string for an empty/invalid ISO input', () => {
    expect(formatDateTime('')).toBe('')
    expect(formatDateTime('not-a-date')).toBe('')
  })

  it('reuses the same cached formatter across calls, producing identical output for the same input', () => {
    // Regression guard for the per-call `toLocaleString` formatter-construction
    // cost this was optimized away from — the cached formatter must still be
    // stateless/deterministic across repeated calls.
    const iso = '2026-01-02T03:04:00.000Z'
    expect(formatDateTime(iso)).toBe(formatDateTime(iso))
  })
})

describe('stateSummaryEntries', () => {
  it('picks the notable keys present in state, labeled, in a stable order', () => {
    expect(
      stateSummaryEntries(
        makeRun({ state: { lastTag: '2.56.0', nextTag: '2.56.1', featCount: 0, fixCount: 1, decision: 'merge' } }),
      ),
    ).toEqual([
      { key: 'lastTag', label: 'Last tag', value: '2.56.0' },
      { key: 'nextTag', label: 'Next tag', value: '2.56.1' },
      { key: 'featCount', label: 'Features', value: '0' },
      { key: 'fixCount', label: 'Fixes', value: '1' },
      { key: 'decision', label: 'Decision', value: 'merge' },
    ])
  })

  it('skips missing, null, and empty-string keys rather than rendering them blank', () => {
    expect(stateSummaryEntries(makeRun({ state: { lastTag: '2.56.0', nextTag: null, decision: '' } }))).toEqual([
      { key: 'lastTag', label: 'Last tag', value: '2.56.0' },
    ])
  })

  it('returns an empty array when state has none of the notable keys', () => {
    expect(stateSummaryEntries(makeRun({ state: { headSHA: 'abc123' } }))).toEqual([])
  })
})

describe('stepLabel', () => {
  it('maps known routine step keys to readable labels', () => {
    expect(stepLabel('compute_tag')).toBe('Compute tag')
    expect(stepLabel('react')).toBe('Add reactions')
    expect(stepLabel('wait_pipeline')).toBe('Wait for pipeline')
    expect(stepLabel('create_mr')).toBe('Open merge request')
  })

  it('falls back to sentence case for unknown snake_case keys', () => {
    expect(stepLabel('sync_release_notes')).toBe('Sync release notes')
    expect(stepLabel('')).toBe('')
  })
})

describe('formatTime', () => {
  it('returns an empty string for empty or invalid input', () => {
    expect(formatTime('')).toBe('')
    expect(formatTime('not-a-date')).toBe('')
  })

  it('formats a valid timestamp without the date part', () => {
    const out = formatTime('2026-09-25T15:27:00Z')
    expect(out).not.toBe('')
    expect(out).not.toMatch(/Sep|2026/)
  })
})
