import { describe, expect, it } from 'vitest'
import type { NotificationRule } from '@shared/api/types'
import { isEventRouted, unroutedEvents } from '@modules/notifications/events'

const rule = (event: string, enabled = true): NotificationRule => ({
  id: `${event}-${enabled}`,
  event,
  notifierKind: 'telegram',
  notifierId: 'tg-1',
  repoId: '',
  enabled,
  createdAt: '',
})

describe('isEventRouted', () => {
  it('is true when an enabled rule targets the event', () => {
    expect(isEventRouted('review.finished', [rule('review.finished')])).toBe(true)
  })

  it('is false when the only matching rule is disabled', () => {
    expect(isEventRouted('review.finished', [rule('review.finished', false)])).toBe(false)
  })

  it('is false when no rule matches the event', () => {
    expect(isEventRouted('release.finished', [rule('review.finished')])).toBe(false)
  })
})

describe('unroutedEvents', () => {
  it('returns events with no enabled rule routing them', () => {
    const events = ['review.finished', 'release.finished']
    const rules = [rule('review.finished'), rule('release.finished', false)]
    expect(unroutedEvents(events, rules)).toEqual(['release.finished'])
  })

  it('returns all events when there are no rules', () => {
    const events = ['review.finished', 'release.finished']
    expect(unroutedEvents(events, [])).toEqual(events)
  })

  it('returns an empty array when every event is routed', () => {
    const events = ['review.finished']
    expect(unroutedEvents(events, [rule('review.finished')])).toEqual([])
  })
})
