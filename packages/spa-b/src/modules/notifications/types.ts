/**
 * Notifications (rules) feature — DTO and payload types, verified against
 * the real backend contract (`GET /notifications/events` and
 * `GET /notifications/rules` curled against localhost:8082, auth off).
 */

export type NotifierKind = 'telegram'

export interface NotificationRule {
  id: string
  event: string
  notifierKind: NotifierKind
  notifierId: string
  /** '' = ALL repositories (global scope); otherwise a repo id. */
  repoId: string
  enabled: boolean
  createdAt: string
}

/** `POST /notifications/rules` body. 404 if `repoId` is set to an unknown repo; 409 on a duplicate tuple. */
export interface CreateNotificationRulePayload {
  event: string
  notifierId: string
  notifierKind?: NotifierKind
  repoId: string
}

/** `GET /notifications/events` response shape. */
export interface NotificationEventsResponse {
  events: string[]
}
