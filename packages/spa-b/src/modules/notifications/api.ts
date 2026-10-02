import { request } from '@shared/api/client'
import type { CreateNotificationRulePayload, NotificationEventsResponse, NotificationRule } from './types'

export function listNotificationEvents(): Promise<NotificationEventsResponse> {
  return request<NotificationEventsResponse>('GET', '/notifications/events')
}

export function listNotificationRules(): Promise<NotificationRule[]> {
  return request<NotificationRule[]>('GET', '/notifications/rules')
}

export function createNotificationRule(payload: CreateNotificationRulePayload): Promise<NotificationRule> {
  return request<NotificationRule>('POST', '/notifications/rules', payload)
}

/** `PATCH /notifications/rules/{id}` — the only mutable field on a rule. */
export function setNotificationRuleEnabled(id: string, enabled: boolean): Promise<NotificationRule> {
  return request<NotificationRule>('PATCH', `/notifications/rules/${id}`, { enabled })
}

export function deleteNotificationRule(id: string): Promise<void> {
  return request<void>('DELETE', `/notifications/rules/${id}`)
}
