// Package review models a code review produced by the 4R engine: a set of
// located findings plus a deterministic score and recommendation.
package review

import "time"

// Dimension is one of the four review lenses.
type Dimension string

const (
	Risk        Dimension = "risk"
	Readability Dimension = "readability"
	Reliability Dimension = "reliability"
	Resilience  Dimension = "resilience"
)

// Valid reports whether d is a known dimension.
func (d Dimension) Valid() bool {
	switch d {
	case Risk, Readability, Reliability, Resilience:
		return true
	default:
		return false
	}
}

// Severity ranks a finding's impact.
type Severity string

const (
	SeverityHigh   Severity = "high"
	SeverityMedium Severity = "medium"
	SeverityLow    Severity = "low"
)

// Recommendation is the overall merge guidance.
type Recommendation string

const (
	Approve        Recommendation = "approve"
	RequestChanges Recommendation = "request_changes"
	Comment        Recommendation = "comment"
)

// Status is the lifecycle state of a review.
type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusDone      Status = "done"
	StatusError     Status = "error"
	StatusCancelled Status = "cancelled"
)

// Terminal reports whether the status is a final lifecycle state that no longer
// changes on its own (done, error or cancelled).
func (s Status) Terminal() bool {
	switch s {
	case StatusDone, StatusError, StatusCancelled:
		return true
	default:
		return false
	}
}

// ContextMode selects how much material the engine sees: Fast is the diff plus
// touched files; Deep clones the repo for surrounding context.
type ContextMode string

const (
	ModeFast ContextMode = "fast"
	ModeDeep ContextMode = "deep"
)

// Finding is a single located issue. Line 0 means the finding is not tied to a
// specific line (file-level or general).
type Finding struct {
	Dimension Dimension
	Severity  Severity
	File      string
	Line      int
	Issue     string
	Why       string
	Fix       string
	Blocking  bool
	Published bool
}

// Review is a completed (or in-flight) review of one merge request.
type Review struct {
	ID          string
	RepoID      string
	MRIID       int
	ContextMode ContextMode
	// ProviderID and Model optionally pin the review to a specific provider and
	// model chosen at launch. Empty means resolve from the repo, then the default
	// provider (the default, backward-compatible behavior).
	ProviderID string
	Model      string
	Status     Status
	// Phase reports fine-grained progress while running (e.g. the current 4R
	// lens in a multi-pass review). Empty when not running.
	Phase string
	// Archived soft-hides the review from the main list while keeping its
	// history. Managed independently of the review lifecycle.
	Archived bool
	// SummaryPublished reports whether the summary/score header has been posted
	// to the merge request. Set on the first publish so subsequent publishes do
	// not re-post it unless explicitly requested.
	SummaryPublished bool
	Summary          string
	Findings         []Finding
	Recommendation   Recommendation
	Score            int
	Error            string
	InputTokens      int
	OutputTokens     int
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// Score weights per finding, subtracted from a perfect 100.
const (
	penaltyHigh     = 15
	penaltyMedium   = 5
	penaltyLow      = 1
	penaltyBlocking = 10
)

// Evaluate derives the recommendation and a 0..100 score from findings,
// deterministically — the model is never asked to grade itself.
func Evaluate(findings []Finding) (Recommendation, int) {
	score := 100
	blocking := false
	anyHigh := false

	for _, f := range findings {
		switch f.Severity {
		case SeverityHigh:
			score -= penaltyHigh
			anyHigh = true
		case SeverityMedium:
			score -= penaltyMedium
		case SeverityLow:
			score -= penaltyLow
		}
		if f.Blocking {
			score -= penaltyBlocking
			blocking = true
		}
	}
	if score < 0 {
		score = 0
	}

	switch {
	case blocking || anyHigh:
		return RequestChanges, score
	case len(findings) > 0:
		return Comment, score
	default:
		return Approve, score
	}
}
