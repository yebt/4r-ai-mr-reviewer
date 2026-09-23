/**
 * Reviews feature — DTO types, per the contract handed to this milestone
 * (no live backend reviews to verify against yet: this backend has never
 * run one). There is no global reviews endpoint — reviews are always
 * fetched per-repo (`GET /repos/{id}/reviews`), which is why the list store
 * fans out over every repo (see `store.ts`).
 */

export type ReviewStatus = 'awaiting_approval' | 'pending' | 'running' | 'done' | 'error' | 'cancelled'

export type ReviewContextMode = 'fast' | 'deep'

export type ReviewRecommendation = 'approve' | 'request_changes' | 'comment'

/**
 * Finding/Reasoning shapes were not part of the contract for this list-only
 * milestone (the list never renders them) — kept as loose, forward-compat
 * stand-ins with an `id` plus an index signature rather than asserting an
 * unverified field list. Tighten these when the detail view lands and the
 * real shape is confirmed against the backend.
 */
export interface Finding {
  id: string
  [key: string]: unknown
}

export interface Reasoning {
  id: string
  [key: string]: unknown
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
