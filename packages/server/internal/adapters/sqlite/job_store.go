package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/webcloster-dev/ai-reviewer/internal/domain/job"
	"github.com/webcloster-dev/ai-reviewer/internal/id"
)

// JobStore is the SQLite-backed implementation of job.Store.
type JobStore struct {
	db *sql.DB
}

// NewJobStore wires a JobStore over a database.
func NewJobStore(db *sql.DB) *JobStore {
	return &JobStore{db: db}
}

var _ job.Store = (*JobStore)(nil)

const jobCols = `id, review_id, status, attempts, last_error, created_at, updated_at`

// Enqueue creates a pending job for a review.
func (s *JobStore) Enqueue(ctx context.Context, reviewID string) (job.Job, error) {
	now := time.Now().UTC()
	j := job.Job{
		ID:        id.New(),
		ReviewID:  reviewID,
		Status:    job.StatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO jobs(`+jobCols+`) VALUES(?,?,?,?,?,?,?)`,
		j.ID, j.ReviewID, string(j.Status), j.Attempts, j.LastError, formatTime(now), formatTime(now))
	if err != nil {
		return job.Job{}, fmt.Errorf("job store: enqueue: %w", err)
	}
	return j, nil
}

// Claim atomically takes the oldest pending job. The UPDATE guarded by
// status='pending' ensures two workers can never claim the same job.
func (s *JobStore) Claim(ctx context.Context) (job.Job, bool, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return job.Job{}, false, fmt.Errorf("job store: claim: begin: %w", err)
	}
	defer tx.Rollback()

	var jobID string
	err = tx.QueryRowContext(ctx,
		`SELECT id FROM jobs WHERE status = ? ORDER BY created_at LIMIT 1`, string(job.StatusPending),
	).Scan(&jobID)
	if errors.Is(err, sql.ErrNoRows) {
		return job.Job{}, false, nil
	}
	if err != nil {
		return job.Job{}, false, fmt.Errorf("job store: claim: select: %w", err)
	}

	res, err := tx.ExecContext(ctx,
		`UPDATE jobs SET status = ?, attempts = attempts + 1, updated_at = ? WHERE id = ? AND status = ?`,
		string(job.StatusRunning), formatTime(time.Now().UTC()), jobID, string(job.StatusPending))
	if err != nil {
		return job.Job{}, false, fmt.Errorf("job store: claim: update: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		// Lost the race to another claimer; nothing taken this round.
		return job.Job{}, false, nil
	}

	j, err := getJobTx(ctx, tx, jobID)
	if err != nil {
		return job.Job{}, false, err
	}
	if err := tx.Commit(); err != nil {
		return job.Job{}, false, fmt.Errorf("job store: claim: commit: %w", err)
	}
	return j, true, nil
}

// Complete marks a job done.
func (s *JobStore) Complete(ctx context.Context, jobID string) error {
	return s.setStatus(ctx, jobID, job.StatusDone, "")
}

// Fail marks a job errored with a message.
func (s *JobStore) Fail(ctx context.Context, jobID, errMsg string) error {
	return s.setStatus(ctx, jobID, job.StatusError, errMsg)
}

func (s *JobStore) setStatus(ctx context.Context, jobID string, status job.Status, errMsg string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE jobs SET status = ?, last_error = ?, updated_at = ? WHERE id = ?`,
		string(status), errMsg, formatTime(time.Now().UTC()), jobID)
	if err != nil {
		return fmt.Errorf("job store: set status: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return job.ErrNotFound
	}
	return nil
}

// RequeueRunning resets running jobs back to pending (startup crash recovery).
func (s *JobStore) RequeueRunning(ctx context.Context) (int, error) {
	res, err := s.db.ExecContext(ctx,
		`UPDATE jobs SET status = ?, updated_at = ? WHERE status = ?`,
		string(job.StatusPending), formatTime(time.Now().UTC()), string(job.StatusRunning))
	if err != nil {
		return 0, fmt.Errorf("job store: requeue running: %w", err)
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

// Get returns a job by id.
func (s *JobStore) Get(ctx context.Context, jobID string) (job.Job, error) {
	j, err := getJobTx(ctx, s.db, jobID)
	if errors.Is(err, sql.ErrNoRows) {
		return job.Job{}, job.ErrNotFound
	}
	if err != nil {
		return job.Job{}, fmt.Errorf("job store: get: %w", err)
	}
	return j, nil
}

// querier is satisfied by both *sql.DB and *sql.Tx.
type querier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func getJobTx(ctx context.Context, q querier, jobID string) (job.Job, error) {
	var (
		j                  job.Job
		status             string
		createdAt, updated string
	)
	err := q.QueryRowContext(ctx, `SELECT `+jobCols+` FROM jobs WHERE id = ?`, jobID).
		Scan(&j.ID, &j.ReviewID, &status, &j.Attempts, &j.LastError, &createdAt, &updated)
	if err != nil {
		return job.Job{}, err
	}
	j.Status = job.Status(status)
	if j.CreatedAt, err = parseTime(createdAt); err != nil {
		return job.Job{}, err
	}
	if j.UpdatedAt, err = parseTime(updated); err != nil {
		return job.Job{}, err
	}
	return j, nil
}
