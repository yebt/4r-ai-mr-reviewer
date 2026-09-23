/**
 * Repos feature — DTO and payload types, verified against the real backend
 * contract (`curl -s http://localhost:8082/repos`). `providerId`/`model`/
 * `defaultProfileId` of `''` mean "use default"; `webhookSecret` is `''`
 * until the webhook is enabled.
 *
 * Preflight/branches/reviews/routines belong to a future Reviews/Runs
 * module — out of scope here.
 */

export interface Repo {
  id: string
  name: string
  url: string
  accountId: string
  providerId: string
  model: string
  defaultProfileId: string
  webhookEnabled: boolean
  webhookRequireConfirmation: boolean
  webhookSecret: string
  webhookPath: string
  createdAt: string
}

/** `POST /repos` body — `accountId` is required; `providerId`/`model`/`profileId` are optional. */
export interface CreateRepoPayload {
  name: string
  url: string
  accountId: string
  providerId: string
  model: string
  profileId: string
}

/**
 * `PATCH /repos/{id}/assign` body — reassign only. `name`/`url` are not
 * editable after create, so they never appear here.
 */
export interface AssignRepoPayload {
  providerId: string
  model: string
  accountId: string
  profileId: string
}

/** `PATCH /repos/{id}/webhook` body. */
export interface SetWebhookPayload {
  enabled: boolean
  requireConfirmation: boolean
}
