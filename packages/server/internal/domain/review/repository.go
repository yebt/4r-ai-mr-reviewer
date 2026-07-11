package review

import (
	"context"
	"errors"
)

// ErrNotFound is returned when a review does not exist.
var ErrNotFound = errors.New("review: not found")

// Repository persists reviews and their findings.
type Repository interface {
	// Create inserts a new review (typically in pending status, no findings).
	Create(ctx context.Context, r Review) error
	// Save persists the full review including its findings (used on completion).
	Save(ctx context.Context, r Review) error
	// Get returns a review with its findings.
	Get(ctx context.Context, id string) (Review, error)
	// ListByRepo returns a repo's active (non-archived) reviews (without
	// findings) newest first.
	ListByRepo(ctx context.Context, repoID string) ([]Review, error)
	// ListArchivedByRepo returns a repo's archived reviews (without findings)
	// newest first.
	ListArchivedByRepo(ctx context.Context, repoID string) ([]Review, error)
	// SetStatus updates only the status and error fields.
	SetStatus(ctx context.Context, id string, status Status, errMsg string) error
	// SetPhase updates only the progress phase (called frequently while running).
	SetPhase(ctx context.Context, id string, phase string) error
	// SetArchived updates only the archived flag. Returns ErrNotFound if the
	// review does not exist.
	SetArchived(ctx context.Context, id string, archived bool) error
	// MarkFindingsPublished flags the findings at the given positions (0-based,
	// in stored order) as published to the platform.
	MarkFindingsPublished(ctx context.Context, reviewID string, positions []int) error
	// Delete hard-removes a review and its findings. Returns ErrNotFound if the
	// review does not exist.
	Delete(ctx context.Context, id string) error
}
