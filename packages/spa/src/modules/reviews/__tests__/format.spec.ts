import { describe, expect, it } from 'vitest'
import type { Dimension } from '@shared/api/types'
import { phaseLabel } from '@modules/reviews/format'

describe('phaseLabel', () => {
  it('maps each 4R phase to its short lens label', () => {
    const expected: Record<Dimension, string> = {
      risk: 'Risk',
      readability: 'Readability',
      reliability: 'Reliability',
      resilience: 'Resilience',
    }
    for (const [phase, label] of Object.entries(expected)) {
      expect(phaseLabel(phase)).toBe(label)
    }
  })

  it('capitalizes an unknown phase instead of leaking a raw key', () => {
    expect(phaseLabel('security')).toBe('Security')
  })

  it('returns an empty string for an empty phase', () => {
    expect(phaseLabel('')).toBe('')
  })
})
