import { request } from '@shared/api/client'
import type {
  AccountProject,
  AssignRepoPayload,
  CreateMergeRequestPayload,
  CreateRepoPayload,
  GenerateMergeRequestPayload,
  GeneratedMergeRequest,
  MergeRequest,
  Repo,
  SetWebhookPayload,
} from './types'

export function listRepos(): Promise<Repo[]> {
  return request<Repo[]>('GET', '/repos')
}

/**
 * `GET /repos/{id}/merge-requests` — the repo's open merge requests, fetched
 * live from GitLab (not persisted server-side — see
 * `handlers.go#listMergeRequests`). Backs the Flow workspace's MRs tab
 * (`modules/flow`).
 */
export function listMergeRequests(repoId: string): Promise<MergeRequest[]> {
  return request<MergeRequest[]>('GET', `/repos/${repoId}/merge-requests`)
}

/**
 * Search the GitLab projects an account's token can see, to power the
 * add-repo project search. An empty `search` returns the account's
 * most-recently-active membership projects. Non-2xx (404 unknown account,
 * 502 upstream GitLab error) throws `ApiError`.
 */
export function searchAccountProjects(accountId: string, search: string): Promise<AccountProject[]> {
  return request<AccountProject[]>(
    'GET',
    `/accounts/${accountId}/projects?search=${encodeURIComponent(search)}`,
  )
}

export function createRepo(payload: CreateRepoPayload): Promise<Repo> {
  return request<Repo>('POST', '/repos', payload)
}

/** Reassign only — `name`/`url` are not editable after create. */
export function assignRepo(id: string, payload: AssignRepoPayload): Promise<Repo> {
  return request<Repo>('PATCH', `/repos/${id}/assign`, payload)
}

export function setRepoWebhook(id: string, payload: SetWebhookPayload): Promise<Repo> {
  return request<Repo>('PATCH', `/repos/${id}/webhook`, payload)
}

export function rotateRepoWebhookSecret(id: string): Promise<Repo> {
  return request<Repo>('POST', `/repos/${id}/webhook/rotate`)
}

export function deleteRepo(id: string): Promise<void> {
  return request<void>('DELETE', `/repos/${id}`)
}

/**
 * `GET /repos/{id}/branches` — the repo's branch names, live from GitLab
 * (`listRepoBranches`/`ListBranches` server-side). Backs the Flow
 * workspace's Release-to-main and New MR dialogs' branch pickers.
 */
export function listRepoBranches(repoId: string): Promise<string[]> {
  return request<string[]>('GET', `/repos/${repoId}/branches`)
}

/**
 * `POST /repos/{id}/merge-requests/generate` — drafts a title+description
 * from the diff between two branches (read-only, no GitLab write; spends
 * LLM tokens). Backs `NewMergeRequestDialog`'s "Generate with AI".
 */
export function generateMergeRequest(
  repoId: string,
  payload: GenerateMergeRequestPayload,
): Promise<GeneratedMergeRequest> {
  return request<GeneratedMergeRequest>('POST', `/repos/${repoId}/merge-requests/generate`, payload)
}

/**
 * `POST /repos/{id}/merge-requests` — opens a REAL merge request on GitLab
 * with the (possibly AI-drafted, possibly edited) title/description. Backs
 * `NewMergeRequestDialog`'s "Create merge request".
 */
export function createMergeRequest(repoId: string, payload: CreateMergeRequestPayload): Promise<MergeRequest> {
  return request<MergeRequest>('POST', `/repos/${repoId}/merge-requests`, payload)
}
