// Types mirroring the backend API DTOs (see packages/server + docs/API.md).

export interface Account {
  id: string
  name: string
  baseUrl: string
  createdAt: string
}

export type ProviderKind = 'openai-compat' | 'anthropic'

export interface Provider {
  id: string
  name: string
  kind: ProviderKind
  baseUrl: string
  model: string
  isDefault: boolean
  temperature: number | null
  models: string[]
  createdAt: string
}

// Telegram notification target. The bot token is write-only: it is sent on
// create and never returned by the backend.
export interface TelegramTarget {
  id: string
  name: string
  chatId: string
  threadId: string
  isDefault: boolean
  createdAt: string
}

// Create body for a Telegram target. botToken is required and write-only;
// threadId/isDefault are optional and omitted when empty/false.
export interface TelegramTargetInput {
  name: string
  botToken: string
  chatId: string
  threadId?: string
  isDefault?: boolean
}

// Discovered Telegram thread (forum topic) the bot has recently seen.
export interface ResolvedThread {
  threadId: string
  name: string
}

// Discovered Telegram chat the bot has recently seen, with any known threads.
// chats and threads are always arrays but may be empty.
export interface ResolvedChat {
  chatId: string
  title: string
  type: string
  threads: ResolvedThread[]
}

// Style-guide distillation state for a humanization profile. Empty string means
// no samples were provided (nothing to distill).
export type ProfileStyleStatus = '' | 'pending' | 'ready' | 'error'

export interface Profile {
  id: string
  name: string
  language: string
  formality: string
  emojis: boolean
  samples: string[]
  styleGuide: string
  styleGuideStatus: ProfileStyleStatus
  styleGuideError: string
  createdAt: string
  updatedAt: string
}

export interface Repo {
  id: string
  name: string
  url: string
  accountId: string
  providerId: string
  model: string
  createdAt: string
}

export interface MergeRequest {
  iid: number
  title: string
  state: string
  sourceBranch: string
  targetBranch: string
  webUrl: string
  author: string
}

export type Dimension = 'risk' | 'readability' | 'reliability' | 'resilience'
export type Severity = 'high' | 'medium' | 'low'

export interface Finding {
  index: number
  dimension: Dimension
  severity: Severity
  file: string
  line: number
  issue: string
  why: string
  fix: string
  blocking: boolean
  published: boolean
}

// Per-publish override for a single finding: replaces that finding's generated
// body with user-chosen (humanized) text. Keyed back to the finding by index.
export interface HumanizeFindingText {
  index: number
  text: string
}

// Structured humanization of one finding, each part rewritten in the author's
// voice. A part is empty when the original finding left it empty.
export interface FindingHumanized {
  issue: string
  why: string
  fix: string
}

// Structured humanization of the review summary.
export interface SummaryHumanized {
  summary: string
}

// Every persisted humanize run for a review, grouped for tab rehydration on
// load. `summary` holds the summary rewrite tabs in order; `findings` maps a
// finding index to its ordered rewrite tabs. The server is the source of truth.
export interface HumanizationsResponse {
  summary: SummaryHumanized[]
  // Object keys are stringified finding indices (JSON object keys are strings).
  findings: Record<string, FindingHumanized[]>
}

// Create-review request body. providerId/model are optional overrides: an empty
// or omitted value tells the backend to resolve from the repo/default provider.
export interface CreateReviewInput {
  repoId: string
  mrIid: number
  mode: string
  providerId?: string
  model?: string
}

export type ReviewStatus = 'pending' | 'running' | 'done' | 'error' | 'cancelled'
export type ContextMode = 'fast' | 'deep'
export type Recommendation = 'approve' | 'request_changes' | 'comment'

export interface Review {
  id: string
  repoId: string
  mrIid: number
  contextMode: ContextMode
  status: ReviewStatus
  phase: string
  archived: boolean
  summaryPublished: boolean
  summary: string
  recommendation: Recommendation
  score: number
  error?: string
  inputTokens: number
  outputTokens: number
  findings: Finding[]
  createdAt: string
  updatedAt: string
}
