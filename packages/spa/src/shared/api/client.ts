// Typed HTTP client for the ai-reviewer backend. All calls go through /api,
// which Vite proxies to the server in dev (see vite.config.ts).
import type {
  Account,
  MergeRequest,
  Provider,
  ProviderKind,
  Repo,
  Review,
} from '@shared/api/types'

const BASE = import.meta.env.VITE_API_URL ?? '/api'

export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
  ) {
    super(message)
    this.name = 'ApiError'
  }
}

/** Extracts a human-readable message from any thrown value. */
export function errorMessage(e: unknown): string {
  if (e instanceof Error) return e.message
  return String(e)
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const res = await fetch(BASE + path, {
    method,
    headers: body ? { 'Content-Type': 'application/json' } : undefined,
    body: body ? JSON.stringify(body) : undefined,
  })

  if (res.status === 204) return undefined as T

  const text = await res.text()
  const data = text ? JSON.parse(text) : undefined

  if (!res.ok) {
    const message = (data as { error?: string })?.error ?? `request failed (${res.status})`
    throw new ApiError(res.status, message)
  }
  return data as T
}

export const api = {
  // accounts
  listAccounts: () => request<Account[]>('GET', '/accounts'),
  createAccount: (input: { name: string; baseUrl: string; token: string }) =>
    request<Account>('POST', '/accounts', input),
  deleteAccount: (id: string) => request<void>('DELETE', `/accounts/${id}`),

  // providers
  listProviders: () => request<Provider[]>('GET', '/providers'),
  createProvider: (input: {
    name: string
    kind: ProviderKind
    baseUrl: string
    model: string
    apiKey: string
    makeDefault: boolean
    temperature: number | null
    models: string[]
  }) => request<Provider>('POST', '/providers', input),
  updateProvider: (
    id: string,
    input: {
      name: string
      kind: ProviderKind
      baseUrl: string
      model: string
      apiKey: string
      temperature: number | null
      models: string[]
    },
  ) => request<Provider>('PATCH', `/providers/${id}`, input),
  setDefaultProvider: (id: string) => request<void>('POST', `/providers/${id}/default`),
  deleteProvider: (id: string) => request<void>('DELETE', `/providers/${id}`),

  // repos
  listRepos: () => request<Repo[]>('GET', '/repos'),
  createRepo: (input: {
    name: string
    url: string
    accountId: string
    providerId: string
    model: string
  }) => request<Repo>('POST', '/repos', input),
  assignRepo: (id: string, input: { providerId: string; model: string }) =>
    request<Repo>('PATCH', `/repos/${id}/assign`, input),
  deleteRepo: (id: string) => request<void>('DELETE', `/repos/${id}`),
  listMergeRequests: (repoId: string) =>
    request<MergeRequest[]>('GET', `/repos/${repoId}/merge-requests`),
  listRepoReviews: (repoId: string) =>
    request<Review[]>('GET', `/repos/${repoId}/reviews`),

  // reviews
  createReview: (input: { repoId: string; mrIid: number; mode: string }) =>
    request<Review>('POST', '/reviews', input),
  getReview: (id: string) => request<Review>('GET', `/reviews/${id}`),
  retryReview: (id: string) => request<Review>('POST', `/reviews/${id}/retry`),
  publishReview: (id: string, input: { all?: boolean; indices?: number[] }) =>
    request<{ status: string }>('POST', `/reviews/${id}/publish`, input),
}
