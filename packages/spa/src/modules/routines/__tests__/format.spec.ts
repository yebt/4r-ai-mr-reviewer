import { describe, expect, it } from 'vitest'
import type { RoutineRunStatus, RoutineStepStatus } from '@shared/api/types'
import { isRunActive, runStatusUi, stepStatusUi } from '@modules/routines/format'

describe('isRunActive', () => {
  it('is true only while the run is still progressing', () => {
    expect(isRunActive('pending')).toBe(true)
    expect(isRunActive('running')).toBe(true)
  })

  it('is false for the resting states', () => {
    expect(isRunActive('blocked')).toBe(false)
    expect(isRunActive('done')).toBe(false)
  })
})

describe('runStatusUi', () => {
  it('maps running to an accent spinner', () => {
    expect(runStatusUi.running).toEqual({
      icon: 'i-lucide-loader-circle',
      class: 'text-accent',
      label: 'Running',
      spin: true,
    })
  })

  it('maps blocked to a warn triangle', () => {
    expect(runStatusUi.blocked.class).toBe('text-warn')
    expect(runStatusUi.blocked.spin).toBe(false)
  })

  it('maps done to an ok check', () => {
    expect(runStatusUi.done).toEqual({
      icon: 'i-lucide-check',
      class: 'text-ok',
      label: 'Done',
      spin: false,
    })
  })

  it('covers every run status with a lucide icon', () => {
    const statuses: RoutineRunStatus[] = ['pending', 'running', 'blocked', 'done']
    for (const status of statuses) {
      expect(runStatusUi[status].icon).toMatch(/^i-lucide-/)
      expect(runStatusUi[status].label).not.toBe('')
    }
  })
})

describe('stepStatusUi', () => {
  it('maps failed to a danger x and skipped to muted', () => {
    expect(stepStatusUi.failed.class).toBe('text-danger')
    expect(stepStatusUi.failed.icon).toBe('i-lucide-x')
    expect(stepStatusUi.skipped.class).toBe('text-muted')
  })

  it('only spins the running step', () => {
    expect(stepStatusUi.running.spin).toBe(true)
    expect(stepStatusUi.pending.spin).toBe(false)
    expect(stepStatusUi.done.spin).toBe(false)
  })

  it('covers every step status with a lucide icon', () => {
    const statuses: RoutineStepStatus[] = ['pending', 'running', 'done', 'failed', 'skipped']
    for (const status of statuses) {
      expect(stepStatusUi[status].icon).toMatch(/^i-lucide-/)
      expect(stepStatusUi[status].label).not.toBe('')
    }
  })
})
