package sqlite

import (
	"context"
	"encoding/json"
	"errors"
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
