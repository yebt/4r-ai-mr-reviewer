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

const routineRunCols = `id, kind, repo_id, mr_iid, status, params_json, state_json, steps_json, last_error, created_at, updated_at, archived`

// Create inserts a new routine run.
func (s *RoutineRunStore) Create(ctx context.Context, run routine.Run) error {
	steps, err := json.Marshal(run.Steps)
	if err != nil {
		return fmt.Errorf("routine run store: create: marshal steps: %w", err)
	}
	now := formatTime(time.Now().UTC())
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO routine_run(`+routineRunCols+`) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`,
		run.ID, string(run.Kind), run.RepoID, mrIIDArg(run.MRIID), string(run.Status),
		rawJSONOr(run.Params, "{}"), rawJSONOr(run.State, "{}"), string(steps), run.LastError, now, now,
		boolToInt(run.Archived))
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
// queryRuns runs a routineRunCols SELECT and scans every row into a slice.
func (s *RoutineRunStore) queryRuns(ctx context.Context, query string, args ...any) ([]routine.Run, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
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

// ListByRepo returns a repo's active (non-archived) runs, newest first.
func (s *RoutineRunStore) ListByRepo(ctx context.Context, repoID string) ([]routine.Run, error) {
	return s.queryRuns(ctx,
		`SELECT `+routineRunCols+` FROM routine_run WHERE repo_id = ? AND archived = 0 ORDER BY created_at DESC`,
		repoID)
}

// ListArchivedByRepo returns a repo's archived runs, newest first.
func (s *RoutineRunStore) ListArchivedByRepo(ctx context.Context, repoID string) ([]routine.Run, error) {
	return s.queryRuns(ctx,
		`SELECT `+routineRunCols+` FROM routine_run WHERE repo_id = ? AND archived = 1 ORDER BY created_at DESC`,
		repoID)
}

// ListRecent returns the most recent active (non-archived) runs across all repos,
// newest first, capped at limit.
func (s *RoutineRunStore) ListRecent(ctx context.Context, limit int) ([]routine.Run, error) {
	return s.queryRuns(ctx,
		`SELECT `+routineRunCols+` FROM routine_run WHERE archived = 0 ORDER BY created_at DESC LIMIT ?`,
		limit)
}

// ListRecentArchived returns the most recent archived runs across all repos,
// newest first, capped at limit.
func (s *RoutineRunStore) ListRecentArchived(ctx context.Context, limit int) ([]routine.Run, error) {
	return s.queryRuns(ctx,
		`SELECT `+routineRunCols+` FROM routine_run WHERE archived = 1 ORDER BY created_at DESC LIMIT ?`,
		limit)
}

// SetArchived flips only the archived flag, mapping a missing row to
// ErrRunNotFound.
func (s *RoutineRunStore) SetArchived(ctx context.Context, id string, archived bool) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE routine_run SET archived = ?, updated_at = ? WHERE id = ?`,
		boolToInt(archived), formatTime(time.Now().UTC()), id)
	if err != nil {
		return fmt.Errorf("routine run store: set archived: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return routine.ErrRunNotFound
	}
	return nil
}

// Save persists the run's status, steps, state, last_error and updated_at. Params
// are immutable and never rewritten.
//
// It is a compare-and-set on status: the UPDATE is guarded by
// "status NOT IN ('cancelled','done')", so the DB row is the synchronization
// point between the worker's per-step Saves and an out-of-band Cancel. Once a row
// is terminal, no further Save can revive it — a worker writing running/done from
// its in-memory copy matches zero rows and gets ErrRunFinalized. The legitimate
// running->done transition still succeeds because the row is running (not
// terminal) when execute writes done: only an ALREADY-terminal row blocks the
// write. On a zero-row update a follow-up SELECT disambiguates a missing row
// (ErrRunNotFound) from an already-finalized one (ErrRunFinalized).
func (s *RoutineRunStore) Save(ctx context.Context, run routine.Run) error {
	steps, err := json.Marshal(run.Steps)
	if err != nil {
		return fmt.Errorf("routine run store: save: marshal steps: %w", err)
	}
	res, err := s.db.ExecContext(ctx, `
		UPDATE routine_run SET
			status = ?, state_json = ?, steps_json = ?, last_error = ?, updated_at = ?
		WHERE id = ? AND status NOT IN ('cancelled','done')`,
		string(run.Status), rawJSONOr(run.State, "{}"), string(steps), run.LastError,
		formatTime(time.Now().UTC()), run.ID)
	if err != nil {
		return fmt.Errorf("routine run store: save: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		// Zero rows means the row is either gone or already terminal. Disambiguate
		// so callers can tell a missing run from a finalized one.
		var status string
		serr := s.db.QueryRowContext(ctx, `SELECT status FROM routine_run WHERE id = ?`, run.ID).Scan(&status)
		if errors.Is(serr, sql.ErrNoRows) {
			return routine.ErrRunNotFound
		}
		if serr != nil {
			return fmt.Errorf("routine run store: save: disambiguate: %w", serr)
		}
		return routine.ErrRunFinalized
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
		archived                         int
	)
	if err := sc.Scan(&run.ID, &kind, &run.RepoID, &mrIID, &status,
		&paramsJSON, &stateJSON, &stepsJSON, &run.LastError, &createdAt, &updatedAt, &archived); err != nil {
		return routine.Run{}, err
	}
	run.Kind = routine.Kind(kind)
	run.Status = routine.RunStatus(status)
	run.Archived = archived != 0
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
