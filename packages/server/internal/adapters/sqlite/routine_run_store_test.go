package sqlite

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/webcloster-dev/ai-reviewer/internal/domain/account"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/repo"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/routine"
	"github.com/webcloster-dev/ai-reviewer/internal/id"
)

func newRoutineRunStore(t *testing.T) (*RoutineRunStore, string) {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	ctx := context.Background()
	acc := account.Account{ID: id.New(), Name: "a", BaseURL: "u", TokenRef: "r", CreatedAt: time.Now().UTC()}
	if err := NewAccountRepo(db).Create(ctx, acc); err != nil {
		t.Fatalf("seed account: %v", err)
	}
	rp := repo.Repo{ID: id.New(), Name: "web", URL: "u", AccountID: acc.ID, CreatedAt: time.Now().UTC()}
	if err := NewRepoStore(db).Create(ctx, rp); err != nil {
		t.Fatalf("seed repo: %v", err)
	}
	return NewRoutineRunStore(db), rp.ID
}

// newRoutineRunStoreMultiRepo seeds one account and repoCount repos, returning
// the store and the repo ids so cross-repo listing can be exercised.
func newRoutineRunStoreMultiRepo(t *testing.T, repoCount int) (*RoutineRunStore, []string) {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	ctx := context.Background()
	acc := account.Account{ID: id.New(), Name: "a", BaseURL: "u", TokenRef: "r", CreatedAt: time.Now().UTC()}
	if err := NewAccountRepo(db).Create(ctx, acc); err != nil {
		t.Fatalf("seed account: %v", err)
	}
	ids := make([]string, 0, repoCount)
	for i := 0; i < repoCount; i++ {
		rp := repo.Repo{ID: id.New(), Name: fmt.Sprintf("repo-%d", i), URL: "u", AccountID: acc.ID, CreatedAt: time.Now().UTC()}
		if err := NewRepoStore(db).Create(ctx, rp); err != nil {
			t.Fatalf("seed repo %d: %v", i, err)
		}
		ids = append(ids, rp.ID)
	}
	return NewRoutineRunStore(db), ids
}

func newRun(repoID string) routine.Run {
	now := time.Now().UTC()
	return routine.Run{
		ID:     id.New(),
		Kind:   routine.KindApproveAndTag,
		RepoID: repoID,
		MRIID:  7,
		Status: routine.RunPending,
		Params: json.RawMessage(`{"emojis":["thumbsup"],"comment":"LGFM","bump":"patch"}`),
		State:  json.RawMessage(`{}`),
		Steps: []routine.Step{
			{Name: "react", Status: routine.StepPending},
			{Name: "comment", Status: routine.StepPending},
			{Name: "tag", Status: routine.StepPending},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func TestRoutineRunCreateGet(t *testing.T) {
	ctx := context.Background()
	s, repoID := newRoutineRunStore(t)

	run := newRun(repoID)
	if err := s.Create(ctx, run); err != nil {
		t.Fatalf("Create: %v", err)
	}
	got, err := s.Get(ctx, run.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Kind != routine.KindApproveAndTag || got.MRIID != 7 || got.Status != routine.RunPending {
		t.Fatalf("unexpected run: %+v", got)
	}
	if len(got.Steps) != 3 || got.Steps[0].Name != "react" {
		t.Fatalf("steps not round-tripped: %+v", got.Steps)
	}
	var params map[string]any
	if err := json.Unmarshal(got.Params, &params); err != nil {
		t.Fatalf("params not valid JSON: %v", err)
	}
	if params["comment"] != "LGFM" {
		t.Fatalf("params not round-tripped: %+v", params)
	}
}

func TestRoutineRunGetMissing(t *testing.T) {
	s, _ := newRoutineRunStore(t)
	if _, err := s.Get(context.Background(), "nope"); !errors.Is(err, routine.ErrRunNotFound) {
		t.Fatalf("got %v, want ErrRunNotFound", err)
	}
}

func TestRoutineRunSave(t *testing.T) {
	ctx := context.Background()
	s, repoID := newRoutineRunStore(t)

	run := newRun(repoID)
	if err := s.Create(ctx, run); err != nil {
		t.Fatalf("Create: %v", err)
	}

	run.Status = routine.RunBlocked
	run.LastError = "boom"
	run.State = json.RawMessage(`{"nextTag":"1.2.4"}`)
	run.Steps[0].Status = routine.StepDone
	run.Steps[2].Status = routine.StepFailed
	if err := s.Save(ctx, run); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := s.Get(ctx, run.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Status != routine.RunBlocked || got.LastError != "boom" {
		t.Fatalf("status/last_error not saved: %+v", got)
	}
	if got.Steps[0].Status != routine.StepDone || got.Steps[2].Status != routine.StepFailed {
		t.Fatalf("steps not saved: %+v", got.Steps)
	}
	var state map[string]any
	if err := json.Unmarshal(got.State, &state); err != nil {
		t.Fatalf("state not valid JSON: %v", err)
	}
	if state["nextTag"] != "1.2.4" {
		t.Fatalf("state not saved: %+v", state)
	}
}

func TestRoutineRunSaveMissing(t *testing.T) {
	s, _ := newRoutineRunStore(t)
	if err := s.Save(context.Background(), newRun("nope")); !errors.Is(err, routine.ErrRunNotFound) {
		t.Fatalf("Save missing = %v, want ErrRunNotFound", err)
	}
}

func TestRoutineRunListByRepoNewestFirst(t *testing.T) {
	ctx := context.Background()
	s, repoID := newRoutineRunStore(t)

	older := newRun(repoID)
	if err := s.Create(ctx, older); err != nil {
		t.Fatalf("Create older: %v", err)
	}
	time.Sleep(2 * time.Millisecond)
	newer := newRun(repoID)
	if err := s.Create(ctx, newer); err != nil {
		t.Fatalf("Create newer: %v", err)
	}

	list, err := s.ListByRepo(ctx, repoID)
	if err != nil {
		t.Fatalf("ListByRepo: %v", err)
	}
	if len(list) != 2 || list[0].ID != newer.ID {
		t.Fatalf("expected newest first, got %+v", list)
	}
}

func TestRoutineRunListRecentAcrossReposNewestFirst(t *testing.T) {
	ctx := context.Background()
	s, repoIDs := newRoutineRunStoreMultiRepo(t, 2)

	// Create three runs, interleaving the two repos, oldest to newest.
	order := []string{repoIDs[0], repoIDs[1], repoIDs[0]}
	created := make([]routine.Run, 0, len(order))
	for i, repoID := range order {
		run := newRun(repoID)
		if err := s.Create(ctx, run); err != nil {
			t.Fatalf("Create run %d: %v", i, err)
		}
		created = append(created, run)
		time.Sleep(2 * time.Millisecond)
	}

	// ListRecent spans both repos, newest first.
	list, err := s.ListRecent(ctx, 10)
	if err != nil {
		t.Fatalf("ListRecent: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("ListRecent len = %d, want 3", len(list))
	}
	if list[0].ID != created[2].ID || list[1].ID != created[1].ID || list[2].ID != created[0].ID {
		t.Fatalf("expected newest first across repos, got %s,%s,%s", list[0].ID, list[1].ID, list[2].ID)
	}
	// It genuinely spans both repos (not scoped to one).
	if list[1].RepoID != repoIDs[1] {
		t.Fatalf("second run repo = %q, want the other repo %q", list[1].RepoID, repoIDs[1])
	}

	// The limit caps the result to the newest N.
	limited, err := s.ListRecent(ctx, 2)
	if err != nil {
		t.Fatalf("ListRecent(limit 2): %v", err)
	}
	if len(limited) != 2 || limited[0].ID != created[2].ID || limited[1].ID != created[1].ID {
		t.Fatalf("limit=2 should return the two newest, got %+v", limited)
	}
}

func TestRoutineRunClaimPendingAtomic(t *testing.T) {
	ctx := context.Background()
	s, repoID := newRoutineRunStore(t)

	run := newRun(repoID)
	if err := s.Create(ctx, run); err != nil {
		t.Fatalf("Create: %v", err)
	}

	claimed, ok, err := s.ClaimPending(ctx)
	if err != nil || !ok {
		t.Fatalf("ClaimPending: ok=%v err=%v", ok, err)
	}
	if claimed.ID != run.ID || claimed.Status != routine.RunRunning {
		t.Fatalf("claimed run wrong: %+v", claimed)
	}

	// The only pending run was taken; a second claim finds nothing.
	if _, ok, _ := s.ClaimPending(ctx); ok {
		t.Fatal("expected no claimable run after the only one was taken")
	}
}

func TestRoutineRunClaimOldestFirst(t *testing.T) {
	ctx := context.Background()
	s, repoID := newRoutineRunStore(t)

	first := newRun(repoID)
	if err := s.Create(ctx, first); err != nil {
		t.Fatalf("Create first: %v", err)
	}
	time.Sleep(2 * time.Millisecond)
	second := newRun(repoID)
	if err := s.Create(ctx, second); err != nil {
		t.Fatalf("Create second: %v", err)
	}

	c1, _, _ := s.ClaimPending(ctx)
	c2, _, _ := s.ClaimPending(ctx)
	if c1.ID != first.ID || c2.ID != second.ID {
		t.Fatalf("claim order wrong: got %s,%s want %s,%s", c1.ID, c2.ID, first.ID, second.ID)
	}
}

// An awaiting_confirmation run is paused on the interactive gate, not queued and
// not running: ClaimPending must not take it and RequeueRunning must not touch
// it. Only pending runs are claimable and only running runs are requeued.
func TestRoutineRunAwaitingConfirmationNotClaimedOrRequeued(t *testing.T) {
	ctx := context.Background()
	s, repoID := newRoutineRunStore(t)

	run := newRun(repoID)
	run.Status = routine.RunAwaitingConfirmation
	if err := s.Create(ctx, run); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// ClaimPending only claims 'pending' runs, so it must find nothing.
	if _, ok, err := s.ClaimPending(ctx); err != nil || ok {
		t.Fatalf("ClaimPending: ok=%v err=%v, want ok=false (awaiting must not be claimed)", ok, err)
	}
	// RequeueRunning only resets 'running' runs, so the awaiting run is untouched.
	if err := s.RequeueRunning(ctx); err != nil {
		t.Fatalf("RequeueRunning: %v", err)
	}

	got, err := s.Get(ctx, run.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Status != routine.RunAwaitingConfirmation {
		t.Fatalf("status = %q, want awaiting_confirmation (untouched)", got.Status)
	}
}

func TestRoutineRunRequeueRunning(t *testing.T) {
	ctx := context.Background()
	s, repoID := newRoutineRunStore(t)

	run := newRun(repoID)
	if err := s.Create(ctx, run); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, ok, _ := s.ClaimPending(ctx); !ok {
		t.Fatal("expected a claim")
	}
	// Simulate a crash: the run is stuck 'running'. Recovery should requeue it.
	if err := s.RequeueRunning(ctx); err != nil {
		t.Fatalf("RequeueRunning: %v", err)
	}
	if _, ok, _ := s.ClaimPending(ctx); !ok {
		t.Fatal("requeued run should be claimable again")
	}
}
