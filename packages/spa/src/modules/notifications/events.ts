import type { NotificationRule } from '@shared/api/types'

// An event is "routed" when at least one ENABLED rule targets a notifier for it.
// A disabled rule does not count: notifications for that event won't be delivered.
export function isEventRouted(event: string, rules: NotificationRule[]): boolean {
  return rules.some((r) => r.enabled && r.event === event)
}

// Available events that have no enabled rule routing them to any notifier.
// Notifications for these events won't be delivered until a rule is added/enabled.
export function unroutedEvents(events: string[], rules: NotificationRule[]): string[] {
  return events.filter((event) => !isEventRouted(event, rules))
}
