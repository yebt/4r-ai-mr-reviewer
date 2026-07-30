package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/webcloster-dev/ai-reviewer/internal/domain/routine"
)

// RoutineRunStore is the SQLite-backed implementation of routine.RunStore.
type RoutineRunStore struct {
	db *sql.DB
}

// NewRoutineRunStore wires a RoutineRunStore over a database.
func NewRoutineRunStore(db *sql.DB) *RoutineRunStore {
	return &RoutineRunStore{db: db}
}

var _ routine.RunStore = (*RoutineRunStore)(nil)

const routineRunCols = `id, kind, repo_id, mr_iid, status, params_json, state_json, steps_json, last_error, created_at, updated_at`

// Create inserts a new routine run.
func (s *RoutineRunStore) Create(ctx context.Context, run routine.Run) error {
	steps, err := json.Marshal(run.Steps)
	if err != nil {
		return fmt.Errorf("routine run store: create: marshal steps: %w", err)
	}
	now := formatTime(time.Now().UTC())
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO routine_run(`+routineRunCols+`) VALUES(?,?,?,?,?,?,?,?,?,?,?)`,
		run.ID, string(run.Kind), run.RepoID, mrIIDArg(run.MRIID), string(run.Status),
		rawJSONOr(run.Params, "{}"), rawJSONOr(run.State, "{}"), string(steps), run.LastError, now, now)
	if err != nil {
		return fmt.Errorf("routine run store: create: %w", err)
	}
	return nil
}

// Get returns a routine run by id.
func (s *RoutineRunStore) Get(ctx context.Context, id string) (routine.Run, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+routineRunCols+` FROM routine_run WHERE id = ?`, id)
	run, err := scanRoutineRun(row)
	if errors.Is(err, sql.ErrNoRows) {
		return routine.Run{}, routine.ErrRunNotFound
	}
	if err != nil {
		return routine.Run{}, fmt.Errorf("routine run store: get: %w", err)
	}
	return run, nil
}

// Delete removes a routine run by id, mapping a missing row to
// routine.ErrRunNotFound.
func (s *RoutineRunStore) Delete(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM routine_run WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("routine run store: delete: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return routine.ErrRunNotFound
	}
	return nil
}

// ListByRepo returns a repo's runs, newest first.
func (s *RoutineRunStore) ListByRepo(ctx context.Context, repoID string) ([]routine.Run, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+routineRunCols+` FROM routine_run WHERE repo_id = ? ORDER BY created_at DESC`, repoID)
	if err != nil {
		return nil, fmt.Errorf("routine run store: list: %w", err)
	}
	defer rows.Close()

	var out []routine.Run
	for rows.Next() {
		run, err := scanRoutineRun(rows)
		if err != nil {
			return nil, fmt.Errorf("routine run store: scan: %w", err)
		}
		out = append(out, run)
	}
	return out, rows.Err()
}

// ListRecent returns the most recent runs across all repos, newest first, capped
// at limit. It mirrors ListByRepo but is not scoped to a single repo, so the
// global "recent runs" view can list activity across every tracked repo.
func (s *RoutineRunStore) ListRecent(ctx context.Context, limit int) ([]routine.Run, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+routineRunCols+` FROM routine_run ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("routine run store: list recent: %w", err)
	}
	defer rows.Close()

	var out []routine.Run
	for rows.Next() {
		run, err := scanRoutineRun(rows)
		if err != nil {
			return nil, fmt.Errorf("routine run store: scan: %w", err)
		}
		out = append(out, run)
	}
	return out, rows.Err()
}

// Save persists the run's status, steps, state, last_error and updated_at. Params
// are immutable and never rewritten.
func (s *RoutineRunStore) Save(ctx context.Context, run routine.Run) error {
	steps, err := json.Marshal(run.Steps)
	if err != nil {
		return fmt.Errorf("routine run store: save: marshal steps: %w", err)
	}
	res, err := s.db.ExecContext(ctx, `
		UPDATE routine_run SET
			status = ?, state_json = ?, steps_json = ?, last_error = ?, updated_at = ?
		WHERE id = ?`,
		string(run.Status), rawJSONOr(run.State, "{}"), string(steps), run.LastError,
		formatTime(time.Now().UTC()), run.ID)
	if err != nil {
		return fmt.Errorf("routine run store: save: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return routine.ErrRunNotFound
	}
	return nil
}

// ClaimPending atomically takes the oldest pending run. The UPDATE guarded by
// status='pending' ensures two workers can never claim the same run.
func (s *RoutineRunStore) ClaimPending(ctx context.Context) (routine.Run, bool, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return routine.Run{}, false, fmt.Errorf("routine run store: claim: begin: %w", err)
	}
	defer tx.Rollback()

	var runID string
	err = tx.QueryRowContext(ctx,
		`SELECT id FROM routine_run WHERE status = ? ORDER BY created_at LIMIT 1`, string(routine.RunPending),
	).Scan(&runID)
	if errors.Is(err, sql.ErrNoRows) {
		return routine.Run{}, false, nil
	}
	if err != nil {
		return routine.Run{}, false, fmt.Errorf("routine run store: claim: select: %w", err)
	}

	res, err := tx.ExecContext(ctx,
		`UPDATE routine_run SET status = ?, updated_at = ? WHERE id = ? AND status = ?`,
		string(routine.RunRunning), formatTime(time.Now().UTC()), runID, string(routine.RunPending))
	if err != nil {
		return routine.Run{}, false, fmt.Errorf("routine run store: claim: update: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		// Lost the race to another claimer; nothing taken this round.
		return routine.Run{}, false, nil
	}

	run, err := scanRoutineRun(tx.QueryRowContext(ctx, `SELECT `+routineRunCols+` FROM routine_run WHERE id = ?`, runID))
	if err != nil {
		return routine.Run{}, false, fmt.Errorf("routine run store: claim: load: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return routine.Run{}, false, fmt.Errorf("routine run store: claim: commit: %w", err)
	}
	return run, true, nil
}

// RequeueRunning resets running runs back to pending (startup crash recovery).
// It is safe because each step is checkpointed, so a resumed run skips the work
// already done.
func (s *RoutineRunStore) RequeueRunning(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE routine_run SET status = ?, updated_at = ? WHERE status = ?`,
		string(routine.RunPending), formatTime(time.Now().UTC()), string(routine.RunRunning))
	if err != nil {
		return fmt.Errorf("routine run store: requeue running: %w", err)
	}
	return nil
}

func scanRoutineRun(sc scanner) (routine.Run, error) {
	var (
		run                              routine.Run
		kind, status                     string
		mrIID                            sql.NullInt64
		paramsJSON, stateJSON, stepsJSON string
		createdAt, updatedAt             string
	)
	if err := sc.Scan(&run.ID, &kind, &run.RepoID, &mrIID, &status,
		&paramsJSON, &stateJSON, &stepsJSON, &run.LastError, &createdAt, &updatedAt); err != nil {
		return routine.Run{}, err
	}
	run.Kind = routine.Kind(kind)
	run.Status = routine.RunStatus(status)
	if mrIID.Valid {
		run.MRIID = int(mrIID.Int64)
	}
	run.Params = json.RawMessage(paramsJSON)
	run.State = json.RawMessage(stateJSON)
	if err := json.Unmarshal([]byte(stepsJSON), &run.Steps); err != nil {
		return routine.Run{}, err
	}

	ct, err := parseTime(createdAt)
	if err != nil {
		return routine.Run{}, err
	}
	ut, err := parseTime(updatedAt)
	if err != nil {
		return routine.Run{}, err
	}
	run.CreatedAt = ct
	run.UpdatedAt = ut
	return run, nil
}

// mrIIDArg stores a 0 MR IID as NULL (the column is nullable) and any other
// value as itself.
func mrIIDArg(iid int) any {
	if iid == 0 {
		return nil
	}
	return iid
}

// rawJSONOr returns r as a string, or def when r is empty, so the JSON columns
// never hold an empty (invalid) value.
func rawJSONOr(r json.RawMessage, def string) string {
	if len(r) == 0 {
		return def
	}
	return string(r)
}
