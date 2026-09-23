import { describe, expect, it } from 'vitest'
import { isRunActive, isRunCancelable, runStatusUi, runTitle } from './format'
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
