import { describe, expect, it } from 'vitest'
import { checkStatusUi, summarizePreflightChecks } from './preflight'
import type { PreflightCheck } from './types'

describe('checkStatusUi', () => {
  it('maps ok to the success badge status', () => {
    expect(checkStatusUi.ok).toEqual({ badgeStatus: 'success', icon: 'check', srLabel: 'Allowed' })
  })

  it('maps fail to the danger badge status', () => {
    expect(checkStatusUi.fail).toEqual({ badgeStatus: 'danger', icon: 'x', srLabel: 'Blocked' })
  })

  it('maps unknown to the warning badge status', () => {
    expect(checkStatusUi.unknown).toEqual({
      badgeStatus: 'warning',
      icon: 'triangle-alert',
      srLabel: 'Unknown',
    })
  })

  it('covers every possible status', () => {
    const statuses: PreflightCheck['status'][] = ['ok', 'fail', 'unknown']
    for (const status of statuses) {
      expect(checkStatusUi[status]).toBeDefined()
    }
  })
})

describe('summarizePreflightChecks', () => {
  function check(status: PreflightCheck['status']): PreflightCheck {
    return { capability: `cap-${status}`, label: status, status, detail: '' }
  }

  it('counts zero checks as all zero', () => {
    expect(summarizePreflightChecks([])).toEqual({ ok: 0, fail: 0, unknown: 0 })
  })

  it('tallies each status independently', () => {
    const checks = [check('ok'), check('ok'), check('fail'), check('unknown'), check('ok')]
    expect(summarizePreflightChecks(checks)).toEqual({ ok: 3, fail: 1, unknown: 1 })
  })
})
