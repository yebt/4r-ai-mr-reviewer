import { describe, expect, it } from 'vitest'
import { emptyFindingsMessage } from './findingsState'

describe('emptyFindingsMessage', () => {
  it('only claims "No findings." once the review is done', () => {
    expect(emptyFindingsMessage('done')).toBe('No findings.')
  })

  it.each(['error', 'awaiting_approval', 'cancelled', 'running', 'pending'] as const)(
    'never says "No findings." for %s',
    (status) => {
      expect(emptyFindingsMessage(status)).not.toMatch(/no findings/i)
      expect(emptyFindingsMessage(status).length).toBeGreaterThan(10)
    },
  )

  it('explains each non-done state', () => {
    expect(emptyFindingsMessage('awaiting_approval')).toMatch(/approv/i)
    expect(emptyFindingsMessage('error')).toMatch(/fail/i)
    expect(emptyFindingsMessage('cancelled')).toMatch(/cancel/i)
  })
})
