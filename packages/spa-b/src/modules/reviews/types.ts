/**
 * Reviews feature — DTO types, per the contract handed to this milestone.
 * The list is backed by the global `GET /reviews` endpoint (every repo,
 * cursor-paginated newest-first — see `api.ts#listRecentReviews` and
 * `store.ts`). `GET /reviews/{id}` returns the full `Review` below,
 * including `findings`/`reasonings` (see `api.ts#getReview`, used by the
 * detail page).
 */

export type ReviewStatus = 'awaiting_approval' | 'pending' | 'running' | 'done' | 'error' | 'cancelled'

export type ReviewContextMode = 'fast' | 'deep'

export type ReviewRecommendation = 'approve' | 'request_changes' | 'comment'

export type ReviewFindingDimension = 'risk' | 'readability' | 'reliability' | 'resilience'

export type ReviewFindingSeverity = 'high' | 'medium' | 'low'

/** One reviewer finding — per the detail-page contract. */
export interface Finding {
  index: number
  dimension: ReviewFindingDimension
  severity: ReviewFindingSeverity
  file: string
  line: number
  issue: string
  why: string
  fix: string
  blocking: boolean
  published: boolean
}

/** One phase of the reviewer's chain-of-thought — per the detail-page contract. */
export interface Reasoning {
  phase: string
  content: string
}

export interface Review {
  id: string
  repoId: string
  mrIid: number
  contextMode: ReviewContextMode
  model?: string
  sourceBranch?: string
  targetBranch?: string
  status: ReviewStatus
  phase: string
  archived: boolean
  summaryPublished: boolean
  summary: string
  recommendation: ReviewRecommendation
  score: number
  error?: string
  inputTokens: number
  outputTokens: number
  findings: Finding[]
  reasonings: Reasoning[]
  createdAt: string
  updatedAt: string
}

/** A `Review` with its owning repo's name attached — the list's fan-out shape. */
export type ReviewWithRepo = Review & { repoName: string }

/**
 * `POST /reviews` body — launches a review for one MR. `providerId`/`model`
 * are optional: an empty/omitted `providerId` resolves to the repo's own
 * provider server-side (see `Service.Create` in
 * `packages/server/internal/app/reviews/service.go`), same for `model`.
 */
export interface CreateReviewInput {
  repoId: string
  mrIid: number
  mode: ReviewContextMode
  providerId?: string
  model?: string
}

/**
 * Humanize (slice 2) — a single profile-voiced rewrite of a finding's
 * issue/why/fix, per `POST /reviews/{id}/humanize` with `target:"finding"`.
 */
export interface FindingHumanized {
  issue: string
  why: string
  fix: string
}

/** A single profile-voiced rewrite of the review summary. */
export interface SummaryHumanized {
  summary: string
}

/**
 * `GET /reviews/{id}/humanizations` — every past humanize run for a review,
 * in run order (= tab order). `findings` is keyed by the finding's `index`
 * as a string (JSON object keys are always strings).
 */
export interface Humanizations {
  summary: SummaryHumanized[]
  findings: Record<string, FindingHumanized[]>
}
