import { computed } from 'vue'
import { defineStore } from 'pinia'
import { useQuery } from '@pinia/colada'
import { createCrudResource, resolveErrorMessage, withoutItem } from '@shared/data/createCrudResource'
import * as notificationsApi from './api'
import type { CreateNotificationRulePayload, NotificationRule } from './types'

export const NOTIFICATION_RULES_QUERY_KEY = ['notification-rules'] as const
export const NOTIFICATION_EVENTS_QUERY_KEY = ['notification-events'] as const

// The event catalog is effectively static (defined by the backend build),
// unlike the rules list which changes on every add/remove/toggle — so it
// gets a long staleTime instead of the default `staleTime: 0` (main.ts),
// avoiding a re-fetch on every mount/window-focus.
const EVENTS_STALE_TIME_MS = 10 * 60 * 1000

/**
 * Pure cache-patch / mapping helpers used by the mutations' `onMutate` hooks
 * below (and the coverage panel). Exported so this logic can be unit-tested
 * directly, without spinning up a full Pinia Colada (Vue app + plugin)
 * context — mirrors providers/telegram's `store.ts`.
 */

/** An event is "routed" when at least one ENABLED rule targets a notifier for it. */
export function isEventRouted(event: string, rules: NotificationRule[]): boolean {
  return rules.some((rule) => rule.enabled && rule.event === event)
}

/** Events with no enabled rule routing them — their notifications won't be delivered. */
export function unroutedEvents(events: string[], rules: NotificationRule[]): string[] {
  return events.filter((event) => !isEventRouted(event, rules))
}

export function withoutRule(rules: NotificationRule[], id: string): NotificationRule[] {
  return withoutItem(rules, id)
}

export function makeOptimisticRule(payload: CreateNotificationRulePayload): NotificationRule {
  return {
    id: `temp-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
    event: payload.event,
    notifierKind: payload.notifierKind ?? 'telegram',
    notifierId: payload.notifierId,
    repoId: payload.repoId,
    enabled: true,
    createdAt: new Date().toISOString(),
  }
}

/**
 * A rule has no general "update" — `PATCH /notifications/rules/{id}` only
 * ever changes `enabled`. This is `createCrudResource`'s `toPatch` for that
 * narrowed `TUpdate` shape.
 */
export function toEnabledPatch(payload: { enabled: boolean }): Partial<Omit<NotificationRule, 'id'>> {
  return { enabled: payload.enabled }
}

export { resolveErrorMessage }

/**
 * Notification rules store, backed by @pinia/colada via `createCrudResource`
 * for create/remove (mirrors providers/telegram). Rules have no general
 * "update" endpoint — the only mutable field is `enabled`
 * (`PATCH /notifications/rules/{id}`) — so the factory's `TUpdate` is
 * narrowed to `{ enabled: boolean }` and surfaced as a dedicated
 * `setEnabled(id, enabled)` instead of a generic `updateRule`. It still goes
 * through the resource's `update` mutation, so an enable/disable toggle gets
 * the same optimistic-patch → rollback-on-error → invalidate behavior as
 * every other field update in this codebase.
 *
 * `['notification-events']` is a second, independent query — a near-static
 * catalog the "Add rule" form and the "Unrouted events" coverage panel both
 * read, kept separate from the frequently-changing rules list so its own
 * long `staleTime` doesn't affect rules' freshness.
 */
export const useNotificationsStore = defineStore('notifications', () => {
  const resource = createCrudResource<NotificationRule, CreateNotificationRulePayload, { enabled: boolean }>({
    queryKey: NOTIFICATION_RULES_QUERY_KEY,
    list: notificationsApi.listNotificationRules,
    create: notificationsApi.createNotificationRule,
    update: (id, payload) => notificationsApi.setNotificationRuleEnabled(id, payload.enabled),
    remove: notificationsApi.deleteNotificationRule,
    makeOptimistic: makeOptimisticRule,
    toPatch: toEnabledPatch,
    messages: {
      createSuccess: 'Notification rule added',
      updateSuccess: 'Notification rule updated',
      removeSuccess: 'Notification rule deleted',
      removeErrorFallback: 'Failed to delete notification rule',
    },
  })

  const eventsQuery = useQuery({
    key: NOTIFICATION_EVENTS_QUERY_KEY,
    query: async () => (await notificationsApi.listNotificationEvents()).events,
    staleTime: EVENTS_STALE_TIME_MS,
  })
  const events = computed(() => eventsQuery.data.value ?? [])

  function setEnabled(id: string, enabled: boolean) {
    return resource.update(id, { enabled })
  }

  return {
    // ['notification-rules'] query surface
    rules: resource.items,
    rulesState: resource.state,
    asyncStatus: resource.asyncStatus,
    isLoading: resource.isLoading,
    error: resource.error,
    refetch: resource.refetch,

    // ['notification-events'] query surface
    events,
    eventsState: eventsQuery.state,
    eventsLoading: eventsQuery.isLoading,
    eventsError: eventsQuery.error,
    refetchEvents: eventsQuery.refetch,

    // mutations — `mutateAsync` rethrows so callers keep their existing
    // try/catch flows on top of the store-level optimistic update +
    // rollback + toast handled above.
    createRule: resource.create,
    isCreating: resource.isCreating,
    removeRule: resource.remove,
    isRemoving: resource.isRemoving,
    setEnabled,
    isSettingEnabled: resource.isUpdating,
  }
})
