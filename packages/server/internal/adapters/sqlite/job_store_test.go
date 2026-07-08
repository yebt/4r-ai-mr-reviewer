package sqlite

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/webcloster-dev/ai-reviewer/internal/domain/account"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/job"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/repo"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/review"
	"github.com/webcloster-dev/ai-reviewer/internal/id"
)

func newJobStore(t *testing.T) (*JobStore, string) {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	ctx := context.Background()
	acc := account.Account{ID: id.New(), Name: "a", BaseURL: "u", TokenRef: "r", CreatedAt: time.Now().UTC()}
	_ = NewAccountRepo(db).Create(ctx, acc)
	rp := repo.Repo{ID: id.New(), Name: "web", URL: "u", AccountID: acc.ID, CreatedAt: time.Now().UTC()}
	_ = NewRepoStore(db).Create(ctx, rp)
	rv := review.Review{ID: id.New(), RepoID: rp.ID, MRIID: 1, Status: review.StatusPending}
	if err := NewReviewStore(db).Create(ctx, rv); err != nil {
		t.Fatalf("seed review: %v", err)
	}
	return NewJobStore(db), rv.ID
}

func TestJobEnqueueClaim(t *testing.T) {
	ctx := context.Background()
	s, reviewID := newJobStore(t)

	enq, err := s.Enqueue(ctx, reviewID)
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	j, ok, err := s.Claim(ctx)
	if err != nil || !ok {
		t.Fatalf("Claim: ok=%v err=%v", ok, err)
	}
	if j.ID != enq.ID || j.Status != job.StatusRunning || j.Attempts != 1 {
		t.Fatalf("claimed job wrong: %+v", j)
	}

	// Nothing left to claim.
	if _, ok, _ := s.Claim(ctx); ok {
		t.Fatal("expected no claimable job after the only one was taken")
	}
}

func TestJobClaimOldestFirst(t *testing.T) {
	ctx := context.Background()
	s, reviewID := newJobStore(t)

	first, _ := s.Enqueue(ctx, reviewID)
	time.Sleep(2 * time.Millisecond)
	second, _ := s.Enqueue(ctx, reviewID)

	j1, _, _ := s.Claim(ctx)
	j2, _, _ := s.Claim(ctx)
	if j1.ID != first.ID || j2.ID != second.ID {
		t.Fatalf("claim order wrong: got %s,%s want %s,%s", j1.ID, j2.ID, first.ID, second.ID)
	}
}

func TestJobCompleteAndFail(t *testing.T) {
	ctx := context.Background()
	s, reviewID := newJobStore(t)

	a, _ := s.Enqueue(ctx, reviewID)
	if err := s.Complete(ctx, a.ID); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if got, _ := s.Get(ctx, a.ID); got.Status != job.StatusDone {
		t.Fatalf("status = %s, want done", got.Status)
	}

	b, _ := s.Enqueue(ctx, reviewID)
	if err := s.Fail(ctx, b.ID, "boom"); err != nil {
		t.Fatalf("Fail: %v", err)
	}
	if got, _ := s.Get(ctx, b.ID); got.Status != job.StatusError || got.LastError != "boom" {
		t.Fatalf("fail not recorded: %+v", got)
	}
}

func TestJobRequeueRunning(t *testing.T) {
	ctx := context.Background()
	s, reviewID := newJobStore(t)

	_, _ = s.Enqueue(ctx, reviewID)
	if _, ok, _ := s.Claim(ctx); !ok {
		t.Fatal("expected a claim")
	}
	// Simulate a crash: the job is stuck 'running'. Recovery should requeue it.
	n, err := s.RequeueRunning(ctx)
	if err != nil || n != 1 {
		t.Fatalf("RequeueRunning: n=%d err=%v", n, err)
	}
	if _, ok, _ := s.Claim(ctx); !ok {
		t.Fatal("requeued job should be claimable again")
	}
}

func TestJobGetMissing(t *testing.T) {
	s, _ := newJobStore(t)
	if _, err := s.Get(context.Background(), "nope"); !errors.Is(err, job.ErrNotFound) {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
}
