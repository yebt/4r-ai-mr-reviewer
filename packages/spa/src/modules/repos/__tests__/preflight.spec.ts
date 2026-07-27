import { describe, expect, it } from 'vitest'
import type { PreflightCheck } from '@shared/api/types'
import { checkStatusUi } from '@modules/repos/preflight'

describe('checkStatusUi', () => {
  it('maps ok to a success icon and color', () => {
    expect(checkStatusUi.ok).toEqual({ icon: 'i-lucide-check', class: 'text-ok', srLabel: 'Allowed' })
  })

  it('maps fail to a danger icon and color', () => {
    expect(checkStatusUi.fail).toEqual({ icon: 'i-lucide-x', class: 'text-danger', srLabel: 'Blocked' })
  })

  it('maps unknown to a warn icon and color', () => {
    expect(checkStatusUi.unknown).toEqual({
      icon: 'i-lucide-triangle-alert',
      class: 'text-warn',
      srLabel: 'Unknown',
    })
  })

  it('covers every possible status', () => {
    const statuses: PreflightCheck['status'][] = ['ok', 'fail', 'unknown']
    for (const status of statuses) {
      expect(checkStatusUi[status]).toBeDefined()
      expect(checkStatusUi[status].icon).toMatch(/^i-lucide-/)
    }
  })
})
