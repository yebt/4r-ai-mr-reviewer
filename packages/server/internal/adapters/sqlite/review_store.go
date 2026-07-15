package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/webcloster-dev/ai-reviewer/internal/domain/review"
	"github.com/webcloster-dev/ai-reviewer/internal/id"
)

// ReviewStore is the SQLite-backed implementation of review.Repository.
type ReviewStore struct {
	db *sql.DB
}

// NewReviewStore wires a ReviewStore over a database.
func NewReviewStore(db *sql.DB) *ReviewStore {
	return &ReviewStore{db: db}
}

var _ review.Repository = (*ReviewStore)(nil)

const reviewCols = `id, repo_id, mr_iid, context_mode, status, phase, archived, summary_published, summary, recommendation, score, error, input_tokens, output_tokens, created_at, updated_at`

// Create inserts a new review row (findings, if any, are written too).
func (r *ReviewStore) Create(ctx context.Context, rv review.Review) error {
	now := formatTime(time.Now().UTC())
	if rv.ContextMode == "" {
		rv.ContextMode = review.ModeFast
	}
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO reviews(`+reviewCols+`) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		rv.ID, rv.RepoID, rv.MRIID, string(rv.ContextMode), string(rv.Status), rv.Phase, boolToInt(rv.Archived), boolToInt(rv.SummaryPublished), rv.Summary, string(rv.Recommendation),
		rv.Score, rv.Error, rv.InputTokens, rv.OutputTokens, now, now)
	if err != nil {
		return fmt.Errorf("review store: create: %w", err)
	}
	return nil
}

// Save updates the review row and replaces its findings in one transaction.
func (r *ReviewStore) Save(ctx context.Context, rv review.Review) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("review store: save: begin: %w", err)
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, `
		UPDATE reviews SET
			status = ?, phase = ?, summary = ?, recommendation = ?, score = ?, error = ?,
			input_tokens = ?, output_tokens = ?, updated_at = ?
		WHERE id = ?`,
		string(rv.Status), rv.Phase, rv.Summary, string(rv.Recommendation), rv.Score, rv.Error,
		rv.InputTokens, rv.OutputTokens, formatTime(time.Now().UTC()), rv.ID)
	if err != nil {
		return fmt.Errorf("review store: save: update: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return review.ErrNotFound
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM review_findings WHERE review_id = ?`, rv.ID); err != nil {
		return fmt.Errorf("review store: save: clear findings: %w", err)
	}
	for i, f := range rv.Findings {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO review_findings(id, review_id, position, dimension, severity, file, line, issue, why, fix, blocking, published)
			VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`,
			id.New(), rv.ID, i, string(f.Dimension), string(f.Severity), f.File, f.Line, f.Issue, f.Why, f.Fix, boolToInt(f.Blocking), boolToInt(f.Published)); err != nil {
			return fmt.Errorf("review store: save: insert finding: %w", err)
		}
	}
	return tx.Commit()
}

// Get returns a review with its findings ordered by position.
func (r *ReviewStore) Get(ctx context.Context, id string) (review.Review, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+reviewCols+` FROM reviews WHERE id = ?`, id)
	rv, err := scanReview(row)
	if errors.Is(err, sql.ErrNoRows) {
		return review.Review{}, review.ErrNotFound
	}
	if err != nil {
		return review.Review{}, fmt.Errorf("review store: get: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT dimension, severity, file, line, issue, why, fix, blocking, published
		FROM review_findings WHERE review_id = ? ORDER BY position`, id)
	if err != nil {
		return review.Review{}, fmt.Errorf("review store: get findings: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			f                   review.Finding
			dim, sev            string
			blocking, published int
		)
		if err := rows.Scan(&dim, &sev, &f.File, &f.Line, &f.Issue, &f.Why, &f.Fix, &blocking, &published); err != nil {
			return review.Review{}, fmt.Errorf("review store: scan finding: %w", err)
		}
		f.Dimension = review.Dimension(dim)
		f.Severity = review.Severity(sev)
		f.Blocking = blocking != 0
		f.Published = published != 0
		rv.Findings = append(rv.Findings, f)
	}
	return rv, rows.Err()
}

// ListByRepo returns a repo's active (non-archived) reviews without findings,
// newest first.
func (r *ReviewStore) ListByRepo(ctx context.Context, repoID string) ([]review.Review, error) {
	return r.listByRepo(ctx, repoID, false)
}

// ListArchivedByRepo returns a repo's archived reviews without findings,
// newest first.
func (r *ReviewStore) ListArchivedByRepo(ctx context.Context, repoID string) ([]review.Review, error) {
	return r.listByRepo(ctx, repoID, true)
}

func (r *ReviewStore) listByRepo(ctx context.Context, repoID string, archived bool) ([]review.Review, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+reviewCols+` FROM reviews WHERE repo_id = ? AND archived = ? ORDER BY created_at DESC`,
		repoID, boolToInt(archived))
	if err != nil {
		return nil, fmt.Errorf("review store: list: %w", err)
	}
	defer rows.Close()

	var out []review.Review
	for rows.Next() {
		rv, err := scanReview(rows)
		if err != nil {
			return nil, fmt.Errorf("review store: scan: %w", err)
		}
		out = append(out, rv)
	}
	return out, rows.Err()
}

// MarkFindingsPublished flags the findings at the given positions as published.
func (r *ReviewStore) MarkFindingsPublished(ctx context.Context, reviewID string, positions []int) error {
	if len(positions) == 0 {
		return nil
	}
	args := make([]any, 0, len(positions)+1)
	args = append(args, reviewID)
	placeholders := make([]string, len(positions))
	for i, p := range positions {
		placeholders[i] = "?"
		args = append(args, p)
	}
	q := `UPDATE review_findings SET published = 1 WHERE review_id = ? AND position IN (` +
		strings.Join(placeholders, ",") + `)`
	if _, err := r.db.ExecContext(ctx, q, args...); err != nil {
		return fmt.Errorf("review store: mark published: %w", err)
	}
	return nil
}

// MarkSummaryPublished flags the review's summary as posted to the platform.
func (r *ReviewStore) MarkSummaryPublished(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE reviews SET summary_published = 1, updated_at = ? WHERE id = ?`,
		formatTime(time.Now().UTC()), id)
	if err != nil {
		return fmt.Errorf("review store: mark summary published: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return review.ErrNotFound
	}
	return nil
}

// SetPhase updates only the progress phase.
func (r *ReviewStore) SetPhase(ctx context.Context, id string, phase string) error {
	if _, err := r.db.ExecContext(ctx,
		`UPDATE reviews SET phase = ?, updated_at = ? WHERE id = ?`,
		phase, formatTime(time.Now().UTC()), id); err != nil {
		return fmt.Errorf("review store: set phase: %w", err)
	}
	return nil
}

// SetArchived updates only the archived flag.
func (r *ReviewStore) SetArchived(ctx context.Context, id string, archived bool) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE reviews SET archived = ?, updated_at = ? WHERE id = ?`,
		boolToInt(archived), formatTime(time.Now().UTC()), id)
	if err != nil {
		return fmt.Errorf("review store: set archived: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return review.ErrNotFound
	}
	return nil
}

// SetStatus updates only status and error.
func (r *ReviewStore) SetStatus(ctx context.Context, id string, status review.Status, errMsg string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE reviews SET status = ?, error = ?, updated_at = ? WHERE id = ?`,
		string(status), errMsg, formatTime(time.Now().UTC()), id)
	if err != nil {
		return fmt.Errorf("review store: set status: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return review.ErrNotFound
	}
	return nil
}

// Delete hard-removes a review and its findings in one transaction.
func (r *ReviewStore) Delete(ctx context.Context, id string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("review store: delete: begin: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM review_findings WHERE review_id = ?`, id); err != nil {
		return fmt.Errorf("review store: delete: clear findings: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM review_humanizations WHERE review_id = ?`, id); err != nil {
		return fmt.Errorf("review store: delete: clear humanizations: %w", err)
	}
	res, err := tx.ExecContext(ctx, `DELETE FROM reviews WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("review store: delete: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return review.ErrNotFound
	}
	return tx.Commit()
}

func scanReview(s scanner) (review.Review, error) {
	var (
		rv                         review.Review
		mode, status, rec          string
		archived, summaryPublished int
		createdAt, updated         string
	)
	if err := s.Scan(&rv.ID, &rv.RepoID, &rv.MRIID, &mode, &status, &rv.Phase, &archived, &summaryPublished, &rv.Summary, &rec,
		&rv.Score, &rv.Error, &rv.InputTokens, &rv.OutputTokens, &createdAt, &updated); err != nil {
		return review.Review{}, err
	}
	rv.ContextMode = review.ContextMode(mode)
	rv.Status = review.Status(status)
	rv.Recommendation = review.Recommendation(rec)
	rv.Archived = archived != 0
	rv.SummaryPublished = summaryPublished != 0

	ct, err := parseTime(createdAt)
	if err != nil {
		return review.Review{}, err
	}
	ut, err := parseTime(updated)
	if err != nil {
		return review.Review{}, err
	}
	rv.CreatedAt = ct
	rv.UpdatedAt = ut
	return rv, nil
}
