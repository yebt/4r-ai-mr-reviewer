import { request } from '@shared/api/client'
import type { AssignRepoPayload, CreateRepoPayload, Repo, SetWebhookPayload } from './types'

export function listRepos(): Promise<Repo[]> {
  return request<Repo[]>('GET', '/repos')
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
