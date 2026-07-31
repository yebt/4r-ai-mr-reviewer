// Typed HTTP client for the ai-reviewer backend. All calls go through /api,
// which Vite proxies to the server in dev (see vite.config.ts).
import type {
  Account,
  AuthStatus,
  ConfirmDecision,
  CreateReviewInput,
  FindingHumanized,
  HumanizationsResponse,
  HumanizeFindingText,
  MainReleaseInput,
  MergeRequest,
  NotificationRule,
  Preflight,
  Profile,
  Provider,
  ProviderKind,
  ReleaseInput,
  Repo,
  ResolvedChat,
  Review,
  RoutineRun,
  SummaryHumanized,
  TelegramTarget,
  TelegramTargetInput,
  TelegramTargetUpdateInput,
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

// Listeners notified whenever an authenticated API call comes back 401 (session
// missing or expired). Lets the auth store clear its state and the router send
// the user to the login page without every caller having to special-case 401.
type UnauthorizedHandler = () => void
const unauthorizedHandlers = new Set<UnauthorizedHandler>()

/**
 * Register a handler fired on any 401 from an authenticated route. Returns an
 * unsubscribe function. 401s from the /auth/* endpoints (e.g. a wrong password
 * on login) are NOT reported here — those are handled inline by the caller.
 */
export function onUnauthorized(handler: UnauthorizedHandler): () => void {
  unauthorizedHandlers.add(handler)
  return () => unauthorizedHandlers.delete(handler)
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const res = await fetch(BASE + path, {
    method,
    headers: body ? { 'Content-Type': 'application/json' } : undefined,
    body: body ? JSON.stringify(body) : undefined,
    // Send the httpOnly `air_session` cookie so authenticated routes work once
    // password auth is enabled. Same-origin in prod; the Vite dev proxy keeps
    // /api same-origin too.
    credentials: 'same-origin',
  })

  // A 401 from an authenticated route means the session is missing or expired.
  // Notify listeners (auth store + router) but still throw so the caller sees it.
  // Skip /auth/* — its 401s (bad password) are expected and handled inline.
  if (res.status === 401 && !path.startsWith('/auth/')) {
    for (const handler of unauthorizedHandlers) handler()
  }

  if (res.status === 204) return undefined as T

  const text = await res.text()

  let data: unknown
  try {
    data = text ? JSON.parse(text) : undefined
  } catch {
    // Non-JSON body — e.g. a plain-text "404 page not found" from the router or a
    // proxy/gateway error. Surface the status and raw text instead of a cryptic
    // JSON.parse failure.
    if (!res.ok) throw new ApiError(res.status, text.trim() || `request failed (${res.status})`)
    throw new Error(`unexpected non-JSON response (${res.status})`)
  }

  if (!res.ok) {
    const message = (data as { error?: string })?.error ?? `request failed (${res.status})`
    throw new ApiError(res.status, message)
  }
  return data as T
}

export const api = {
  // auth (optional password auth; /auth/* never requires a session)
  authStatus: () => request<AuthStatus>('GET', '/auth/status'),
  login: (password: string) =>
    request<{ authenticated: boolean }>('POST', '/auth/login', { password }),
  logout: () => request<void>('POST', '/auth/logout'),

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

  // telegram (notification targets)
  listTelegram: () => request<TelegramTarget[]>('GET', '/telegram'),
  createTelegram: (input: TelegramTargetInput) => {
    // Always send name/botToken/chatId; only include the optional threadId and
    // isDefault when meaningful so empty values aren't persisted as literals.
    const body: TelegramTargetInput = {
      name: input.name,
      botToken: input.botToken,
      chatId: input.chatId,
    }
    const thread = input.threadId?.trim()
    if (thread) body.threadId = thread
    if (input.isDefault) body.isDefault = true
    return request<TelegramTarget>('POST', '/telegram', body)
  },
  updateTelegram: (id: string, input: TelegramTargetUpdateInput) => {
    // Always send name/chatId; include threadId/isDefault only when meaningful,
    // and botToken only when the user typed a new one (blank keeps the current
    // token server-side) so an empty string never rotates the token.
    const body: TelegramTargetUpdateInput = { name: input.name, chatId: input.chatId }
    const thread = input.threadId?.trim()
    if (thread) body.threadId = thread
    if (input.isDefault) body.isDefault = true
    const token = input.botToken?.trim()
    if (token) body.botToken = token
    return request<TelegramTarget>('PUT', `/telegram/${id}`, body)
  },
  duplicateTelegram: (id: string) => request<TelegramTarget>('POST', `/telegram/${id}/duplicate`),
  // Discover chats/threads the bot has recently seen so the user can pick an id
  // instead of copying it by hand. A bad token yields a non-2xx -> ApiError.
  resolveTelegram: (botToken: string) =>
    request<{ chats: ResolvedChat[] }>('POST', '/telegram/resolve', { botToken }),
  setDefaultTelegram: (id: string) =>
    request<{ status: string }>('POST', `/telegram/${id}/default`),
  setBotTelegram: (id: string) => request<{ status: string }>('POST', `/telegram/${id}/bot`),
  testTelegram: (id: string) => request<{ status: string }>('POST', `/telegram/${id}/test`),
  deleteTelegram: (id: string) => request<void>('DELETE', `/telegram/${id}`),

  // notifications
  listNotificationEvents: () =>
    request<{ events: string[] }>('GET', '/notifications/events'),
  listNotificationRules: () => request<NotificationRule[]>('GET', '/notifications/rules'),
  createNotificationRule: (input: { event: string; notifierId: string }) =>
    request<NotificationRule>('POST', '/notifications/rules', input),
  setNotificationRuleEnabled: (id: string, enabled: boolean) =>
    request<NotificationRule>('PATCH', `/notifications/rules/${id}`, { enabled }),
  deleteNotificationRule: (id: string) => request<void>('DELETE', `/notifications/rules/${id}`),

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
  // Preflight the repo's token scopes and project access to learn which
  // automated actions are permitted before running any routine. Non-2xx (404
  // unknown repo, 502 upstream GitLab error) throws ApiError.
  preflightRepo: (id: string) => request<Preflight>('GET', `/repos/${id}/preflight`),
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

  // routines (automated GitLab actions)
  // Dev-flow release: requires the MR target to be `development` (the backend
  // rejects other targets). Non-2xx (400 bad target, 409 duplicate) throws.
  createRelease: (repoId: string, input: ReleaseInput) =>
    request<RoutineRun>('POST', `/repos/${repoId}/routines/release`, input),
  // Main-flow release: defaults source=development, target=main when omitted.
  createMainRelease: (repoId: string, input: MainReleaseInput) =>
    request<RoutineRun>('POST', `/repos/${repoId}/routines/release-main`, input),
  getRoutineRun: (id: string) => request<RoutineRun>('GET', `/routines/${id}`),
  listRepoRoutines: (repoId: string) =>
    request<RoutineRun[]>('GET', `/repos/${repoId}/routines`),
  // Recent routine runs across all repos, newest first. Each item carries a
  // best-effort repoName. The optional limit is clamped server-side.
  listRecentRoutines: (limit?: number) =>
    request<RoutineRun[]>('GET', `/routines${limit != null ? `?limit=${limit}` : ''}`),
  resumeRoutine: (id: string) => request<RoutineRun>('POST', `/routines/${id}/resume`),
  // Answer a release run's confirmation gate. 409 when the run is not
  // awaiting_confirmation.
  confirmRoutine: (id: string, decision: ConfirmDecision) =>
    request<RoutineRun>('POST', `/routines/${id}/confirm`, { decision }),
  // Cancel a routine run and get the updated (cancelled) run back. 404 when
  // unknown, 409 when the run is already terminal (done/cancelled).
  cancelRoutine: (id: string) => request<RoutineRun>('POST', `/routines/${id}/cancel`),
  // Delete a routine run. 404 when unknown, 409 when the run is still running
  // (a running action can't be deleted) — both throw ApiError for the caller.
  deleteRoutine: (id: string) => request<void>('DELETE', `/routines/${id}`),

  // skills
  getSkills: () =>
    request<{ risk: string; readability: string; reliability: string; resilience: string }>(
      'GET',
      '/skills',
    ),
}
