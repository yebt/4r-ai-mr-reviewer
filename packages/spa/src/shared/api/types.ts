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

// Ephemeral humanized rewrite of a review, keyed back to findings by index.
// Nothing is persisted server-side — these are preview-only variants.
export interface HumanizeFindingText {
  index: number
  text: string
}

export interface HumanizeVariant {
  summary: string
  findings: HumanizeFindingText[]
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
