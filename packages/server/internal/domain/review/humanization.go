package review

import (
	"context"
	"time"
)

// HumanizationTarget names which part of a review a humanization rewrites.
type HumanizationTarget string

const (
	// HumanizationSummary marks a rewrite of the review summary.
	HumanizationSummary HumanizationTarget = "summary"
	// HumanizationFinding marks a rewrite of a single finding.
	HumanizationFinding HumanizationTarget = "finding"
)

// SummaryFindingIndex is the sentinel finding index stored for summary
// humanizations (a summary is not tied to any finding position).
const SummaryFindingIndex = -1

// Humanization is one persisted humanize run: either a summary rewrite or a
// single finding rewrite, kept so the paid LLM output survives a page reload.
//
// FindingIndex is stable for the life of a humanization: a humanization always
// belongs to one ReviewID, humanize only runs on a review whose status is done
// (see the humanize service), and a done review's findings never change in place
// — a retry FORKS a brand-new review with its own id and its own (empty)
// humanization set (see reviews.Service.Retry / TestRetryClonesReview) rather
// than regenerating this review's findings. So a stored FindingIndex can never
// come to point at a different finding, and no reconciliation is needed.
type Humanization struct {
	ID        string
	ReviewID  string
	ProfileID string
	Target    HumanizationTarget
	// FindingIndex is the finding position for finding rewrites, or
	// SummaryFindingIndex for summary rewrites.
	FindingIndex int
	// TabIndex is the 0-based order of this run within its (review, target,
	// finding_index) group, assigned by the store on Add.
	TabIndex  int
	Summary   string // set for summary rewrites
	Issue     string // set for finding rewrites
	Why       string // set for finding rewrites
	Fix       string // set for finding rewrites
	CreatedAt time.Time
}

// HumanizationRepository persists humanize runs and lists them for rehydration.
type HumanizationRepository interface {
	// Add assigns the next tab_index within the run's (review, target,
	// finding_index) group and returns the stored record.
	Add(ctx context.Context, h Humanization) (Humanization, error)
	// ListByReview returns a review's humanizations ordered by target,
	// finding_index, then tab_index.
	ListByReview(ctx context.Context, reviewID string) ([]Humanization, error)
}
