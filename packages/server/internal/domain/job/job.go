// Package job models the async work queue that drives review execution.
package job

import (
	"context"
	"errors"
	"time"
)

// ErrNotFound is returned when a job does not exist.
var ErrNotFound = errors.New("job: not found")

// Status is a job's lifecycle state.
type Status string

const (
	StatusPending Status = "pending"
	StatusRunning Status = "running"
	StatusDone    Status = "done"
	StatusError   Status = "error"
)

// Job is one unit of async work: run the review it points at.
type Job struct {
	ID        string
	ReviewID  string
	Status    Status
	Attempts  int
	LastError string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Store persists jobs and hands them out for execution.
type Store interface {
	// Enqueue creates a pending job for a review.
	Enqueue(ctx context.Context, reviewID string) (Job, error)
	// Claim atomically takes the oldest pending job, marking it running and
	// incrementing its attempt count. ok is false when nothing is pending.
	Claim(ctx context.Context) (j Job, ok bool, err error)
	// Complete marks a job done.
	Complete(ctx context.Context, jobID string) error
	// Fail marks a job errored with a message.
	Fail(ctx context.Context, jobID, errMsg string) error
	// RequeueRunning resets running jobs back to pending; used at startup to
	// recover work that was interrupted by a crash. Returns how many it reset.
	RequeueRunning(ctx context.Context) (int, error)
	// Get returns a job by id.
	Get(ctx context.Context, jobID string) (Job, error)
}
