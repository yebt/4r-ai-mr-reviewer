// Typed HTTP client for the ai-reviewer backend. All calls go through /api,
// which Vite proxies to the server in dev (see vite.config.ts).
import type {
  Account,
  CreateReviewInput,
  FindingHumanized,
  HumanizationsResponse,
  HumanizeFindingText,
  MergeRequest,
  Profile,
  Provider,
  ProviderKind,
  Repo,
  Review,
  SummaryHumanized,
} from '@shared/api/types'

// Create/update body for a humanization profile. styleGuide* fields are
// server-managed and never sent.
export interface ProfileInput {
  name: string
  language: string
  formality: string
  emojis: boolean
  samples: string[]
}

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

  // profiles (humanization)
  listProfiles: () => request<Profile[]>('GET', '/profiles'),
  getProfile: (id: string) => request<Profile>('GET', `/profiles/${id}`),
  createProfile: (input: ProfileInput) => request<Profile>('POST', '/profiles', input),
  updateProfile: (id: string, input: ProfileInput) =>
    request<Profile>('PATCH', `/profiles/${id}`, input),
  deleteProfile: (id: string) => request<void>('DELETE', `/profiles/${id}`),
  redistillProfile: (id: string) =>
    request<{ status: string }>('POST', `/profiles/${id}/redistill`),

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
  listRepoReviews: (repoId: string, archived = false) =>
    request<Review[]>('GET', `/repos/${repoId}/reviews${archived ? '?archived=1' : ''}`),

  // reviews
  createReview: (input: CreateReviewInput) => {
    const { repoId, mrIid, mode, providerId, model } = input
    // Only send provider/model overrides when non-empty; the backend treats an
    // empty value the same as omitted (resolve from the repo/default provider).
    const body: CreateReviewInput = { repoId, mrIid, mode }
    if (providerId) body.providerId = providerId
    // Trim the free-text model so a whitespace-only value is omitted (falls back
    // to the provider default) instead of being sent as a literal model name.
    const m = model?.trim()
    if (m) body.model = m
    return request<Review>('POST', '/reviews', body)
  },
  getReview: (id: string) => request<Review>('GET', `/reviews/${id}`),
  deleteReview: (id: string) => request<void>('DELETE', `/reviews/${id}`),
  retryReview: (id: string) => request<Review>('POST', `/reviews/${id}/retry`),
  cancelReview: (id: string) => request<{ status: string }>('POST', `/reviews/${id}/cancel`),
  archiveReview: (id: string) => request<{ status: string }>('POST', `/reviews/${id}/archive`),
  unarchiveReview: (id: string) => request<{ status: string }>('POST', `/reviews/${id}/unarchive`),
  publishReview: (
    id: string,
    input: {
      all?: boolean
      indices?: number[]
      includeSummary?: boolean
      // Per-publish overrides: replace the generated summary body and/or the
      // generated body of individual findings with user-chosen (humanized) text.
      // Omitted fields keep today's behavior (generated text is posted).
      summaryOverride?: string
      findingOverrides?: HumanizeFindingText[]
    },
  ) => request<{ status: string }>('POST', `/reviews/${id}/publish`, input),
  // Humanize rewrites a single finding's issue/why/fix in the profile's voice.
  // The run is persisted server-side (see getHumanizations) and also returned so
  // the UI can append it immediately. Runs a synchronous LLM completion so it may
  // be slow. A part comes back empty when the original finding left it empty.
  humanizeFinding: (id: string, profileId: string, index: number) =>
    request<FindingHumanized>('POST', `/reviews/${id}/humanize`, {
      profileId,
      target: 'finding',
      index,
    }),
  // Humanize rewrites the review summary in the profile's voice. Persisted too.
  humanizeSummary: (id: string, profileId: string) =>
    request<SummaryHumanized>('POST', `/reviews/${id}/humanize`, { profileId, target: 'summary' }),
  // getHumanizations returns every persisted humanize run of a review, grouped so
  // the store can rehydrate its tabs after a page reload.
  getHumanizations: (id: string) =>
    request<HumanizationsResponse>('GET', `/reviews/${id}/humanizations`),

  // skills
  getSkills: () =>
    request<{ risk: string; readability: string; reliability: string; resilience: string }>(
      'GET',
      '/skills',
    ),
}
