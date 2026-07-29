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
  // The single target whose bot token drives interactive bot commands.
  isBot: boolean
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

// Notification rule: routes an event (e.g. "review.finished") to a notifier
// target (currently always a Telegram target, identified by notifierId).
export interface NotificationRule {
  id: string
  event: string
  notifierKind: string
  notifierId: string
  enabled: boolean
  createdAt: string
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

// One capability probed by the repo preflight: whether the configured token +
// project access permit a given automated action. status is 'unknown' when the
// prerequisite (e.g. branch protection) could not be read.
export interface PreflightCheck {
  capability: string
  label: string
  status: 'ok' | 'fail' | 'unknown'
  detail: string
}

// Token-scope + project-permission preflight for a repo. Reports which
// automated GitLab actions the configured token/access allow before any
// routine runs. scopesKnown is false when the token's scopes could not be read.
export interface Preflight {
  tokenScopes: string[]
  scopesKnown: boolean
  accessLevel: number
  accessLevelName: string
  defaultBranch: string
  checks: PreflightCheck[]
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

// --- Routines (automated GitLab actions driven by a step ledger) ---

// Status of a single step within a routine run.
export type RoutineStepStatus = 'pending' | 'running' | 'done' | 'failed' | 'skipped'

// Overall status of a routine run. A `blocked` run stopped on a recoverable
// condition and can be resumed; `done` is terminal.
export type RoutineRunStatus = 'pending' | 'running' | 'blocked' | 'done'

// The approve_and_tag routine executes these steps, in this fixed order.
export type RoutineStepName = 'react' | 'comment' | 'tag'

export interface RoutineStep {
  name: RoutineStepName
  status: RoutineStepStatus
  detail: string
  updatedAt: string
}

// A routine run. `state` is a decoded object (may be `{}`); `nextTag` is the
// most useful field for approve_and_tag and is surfaced in the UI.
export interface RoutineRun {
  id: string
  kind: 'approve_and_tag'
  repoId: string
  mrIid: number
  status: RoutineRunStatus
  steps: RoutineStep[]
  state: { nextTag?: string; [key: string]: unknown }
  lastError: string
  createdAt: string
  updatedAt: string
}

// Create body for an approve_and_tag run. Only mrIid is required; the backend
// fills defaults (emojis: thumbsup/seedling, comment: "LGFM", bump: patch) when
// the optional fields are omitted.
export interface ApproveAndTagInput {
  mrIid: number
  emojis?: string[]
  comment?: string
  bump?: 'major' | 'minor' | 'patch'
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

// Model reasoning (chain-of-thought) captured per 4R phase during the run. The
// backend emits one entry as each lens completes (non-streaming, so per-phase
// granularity), ordered by capture time. `reasonings` is always an array —
// empty when nothing was captured (reasoning budget off or model exposes none).
export interface ReviewReasoning {
  phase: string
  content: string
}

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
  reasonings: ReviewReasoning[]
  createdAt: string
  updatedAt: string
}
