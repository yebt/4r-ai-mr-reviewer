/**
 * Repos feature — DTO and payload types, verified against the real backend
 * contract (`curl -s http://localhost:8082/repos`). `providerId`/`model`/
 * `defaultProfileId` of `''` mean "use default"; `webhookSecret` is `''`
 * until the webhook is enabled.
 *
 * Reviews and Runs now live in their own `modules/reviews` and
 * `modules/runs` — this file stays repo-only.
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

/**
 * A GitLab project from `GET /accounts/{id}/projects`, used to power the
 * add-repo project search (`webUrl` fills the repo URL on selection;
 * `pathWithNamespace` is the primary label, e.g. "group/project").
 */
export interface AccountProject {
  id: number
  name: string
  pathWithNamespace: string
  webUrl: string
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

/**
 * A repo's open merge request, from `GET /repos/{id}/merge-requests`
 * (`mrResp`/`toMR` server-side — see `server/internal/http/handlers.go`).
 * Live GitLab data, not persisted — backs the Flow workspace's MRs tab
 * (`modules/flow`).
 */
export interface MergeRequest {
  iid: number
  title: string
  description?: string
  state: string
  sourceBranch: string
  targetBranch: string
  webUrl: string
  author: string
}

/**
 * `POST /repos/{id}/merge-requests/generate` body — drafts a title+
 * description from the diff between two EXISTING branches, without opening
 * the merge request (read-only: diff + LLM only, no GitLab write — see
 * `handlers_mergerequests.go#generateMergeRequest`). `profileId`/
 * `providerId`/`model` are optional per-request overrides.
 */
export interface GenerateMergeRequestPayload {
  sourceBranch: string
  targetBranch: string
  profileId?: string
  providerId?: string
  model?: string
}

/** `POST /repos/{id}/merge-requests/generate` response. */
export interface GeneratedMergeRequest {
  title: string
  description: string
}

/**
 * `POST /repos/{id}/merge-requests` body — opens a REAL merge request on
 * GitLab with the (possibly edited) title/description the user reviewed
 * after `generateMergeRequest` (see
 * `handlers_mergerequests.go#createMergeRequest`).
 */
export interface CreateMergeRequestPayload {
  sourceBranch: string
  targetBranch: string
  title: string
  description: string
}

/**
 * One probed routine capability from `GET /repos/{id}/preflight` (mirrors
 * `preflightCheckResp` in `server/internal/http/handlers_routines.go`).
 * `detail` is `''` when the check has nothing extra to say.
 */
export interface PreflightCheck {
  capability: string
  label: string
  status: 'ok' | 'fail' | 'unknown'
  detail: string
}

/**
 * `GET /repos/{id}/preflight` response (`preflightResp` server-side) —
 * reports which merge-request routine actions the repo's token and access
 * level permit, from live GitLab reads (project, membership, protected
 * branches/tags). `scopesKnown` is `false` when the token type (e.g. a
 * GitLab OAuth token) doesn't expose its scopes, in which case `tokenScopes`
 * is `[]` and the UI should say "scopes could not be read" rather than
 * implying an empty scope list.
 */
export interface Preflight {
  tokenScopes: string[]
  scopesKnown: boolean
  accessLevel: number
  accessLevelName: string
  defaultBranch: string
  checks: PreflightCheck[]
}
