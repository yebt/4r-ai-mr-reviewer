import { request } from '@shared/api/client'
import type { AccountProject, AssignRepoPayload, CreateRepoPayload, Repo, SetWebhookPayload } from './types'

export function listRepos(): Promise<Repo[]> {
  return request<Repo[]>('GET', '/repos')
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
