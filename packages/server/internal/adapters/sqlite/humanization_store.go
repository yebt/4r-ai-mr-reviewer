package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/webcloster-dev/ai-reviewer/internal/domain/review"
	"github.com/webcloster-dev/ai-reviewer/internal/id"
)

// HumanizationStore is the SQLite-backed implementation of
// review.HumanizationRepository.
type HumanizationStore struct {
	db *sql.DB
}

// NewHumanizationStore wires a HumanizationStore over a database.
func NewHumanizationStore(db *sql.DB) *HumanizationStore {
	return &HumanizationStore{db: db}
}

var _ review.HumanizationRepository = (*HumanizationStore)(nil)

const humanizationCols = `id, review_id, profile_id, target, finding_index, tab_index, summary, issue, why, fix, created_at`

// Add assigns the next tab_index for the run's (review, target, finding_index)
// group and inserts it. The COUNT and INSERT run in one transaction so
// concurrent Adds to the same group cannot collide on tab_index.
func (s *HumanizationStore) Add(ctx context.Context, h review.Humanization) (review.Humanization, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return review.Humanization{}, fmt.Errorf("humanization store: add: begin: %w", err)
	}
	defer tx.Rollback()

	var count int
	if err := tx.QueryRowContext(ctx,
		`SELECT COUNT(1) FROM review_humanizations WHERE review_id = ? AND target = ? AND finding_index = ?`,
		h.ReviewID, string(h.Target), h.FindingIndex).Scan(&count); err != nil {
		return review.Humanization{}, fmt.Errorf("humanization store: add: count: %w", err)
	}

	h.ID = id.New()
	h.TabIndex = count
	h.CreatedAt = time.Now().UTC()

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO review_humanizations(`+humanizationCols+`) VALUES(?,?,?,?,?,?,?,?,?,?,?)`,
		h.ID, h.ReviewID, h.ProfileID, string(h.Target), h.FindingIndex, h.TabIndex,
		h.Summary, h.Issue, h.Why, h.Fix, formatTime(h.CreatedAt)); err != nil {
		return review.Humanization{}, fmt.Errorf("humanization store: add: insert: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return review.Humanization{}, fmt.Errorf("humanization store: add: commit: %w", err)
	}
	return h, nil
}

// ListByReview returns a review's humanizations ordered by target, finding_index
// and tab_index so callers can group them while preserving tab order.
func (s *HumanizationStore) ListByReview(ctx context.Context, reviewID string) ([]review.Humanization, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+humanizationCols+` FROM review_humanizations WHERE review_id = ?
		 ORDER BY target, finding_index, tab_index`, reviewID)
	if err != nil {
		return nil, fmt.Errorf("humanization store: list: %w", err)
	}
	defer rows.Close()

	var out []review.Humanization
	for rows.Next() {
		h, err := scanHumanization(rows)
		if err != nil {
			return nil, fmt.Errorf("humanization store: scan: %w", err)
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

func scanHumanization(s scanner) (review.Humanization, error) {
	var (
		h         review.Humanization
		target    string
		createdAt string
	)
	if err := s.Scan(&h.ID, &h.ReviewID, &h.ProfileID, &target, &h.FindingIndex, &h.TabIndex,
		&h.Summary, &h.Issue, &h.Why, &h.Fix, &createdAt); err != nil {
		return review.Humanization{}, err
	}
	h.Target = review.HumanizationTarget(target)
	ct, err := parseTime(createdAt)
	if err != nil {
		return review.Humanization{}, err
	}
	h.CreatedAt = ct
	return h, nil
}
