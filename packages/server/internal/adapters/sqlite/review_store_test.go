package sqlite

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/webcloster-dev/ai-reviewer/internal/domain/account"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/repo"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/review"
	"github.com/webcloster-dev/ai-reviewer/internal/id"
)

func newReviewStore(t *testing.T) (*ReviewStore, string) {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	ctx := context.Background()
	acc := account.Account{ID: id.New(), Name: "a", BaseURL: "https://gitlab.com", TokenRef: "ref", CreatedAt: time.Now().UTC()}
	if err := NewAccountRepo(db).Create(ctx, acc); err != nil {
		t.Fatalf("seed account: %v", err)
	}
	rp := repo.Repo{ID: id.New(), Name: "web", URL: "u", AccountID: acc.ID, CreatedAt: time.Now().UTC()}
	if err := NewRepoStore(db).Create(ctx, rp); err != nil {
		t.Fatalf("seed repo: %v", err)
	}
	return NewReviewStore(db), rp.ID
}

func TestReviewCreateGet(t *testing.T) {
	ctx := context.Background()
	s, repoID := newReviewStore(t)

	rv := review.Review{ID: id.New(), RepoID: repoID, MRIID: 7, Status: review.StatusPending}
	if err := s.Create(ctx, rv); err != nil {
		t.Fatalf("Create: %v", err)
	}
	got, err := s.Get(ctx, rv.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Status != review.StatusPending || got.MRIID != 7 {
		t.Fatalf("unexpected review: %+v", got)
	}
}

func TestReviewSaveWithFindings(t *testing.T) {
	ctx := context.Background()
	s, repoID := newReviewStore(t)

	rv := review.Review{ID: id.New(), RepoID: repoID, MRIID: 3, Status: review.StatusPending}
	if err := s.Create(ctx, rv); err != nil {
		t.Fatalf("Create: %v", err)
	}

	rv.Status = review.StatusDone
	rv.Summary = "has one blocker"
	rv.Recommendation = review.RequestChanges
	rv.Score = 75
	rv.InputTokens = 100
	rv.OutputTokens = 40
	rv.Findings = []review.Finding{
		{Dimension: review.Risk, Severity: review.SeverityHigh, File: "a.go", Line: 5, Issue: "secret", Blocking: true},
		{Dimension: review.Readability, Severity: review.SeverityLow, File: "b.go", Line: 9, Issue: "naming"},
	}
	if err := s.Save(ctx, rv); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := s.Get(ctx, rv.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Status != review.StatusDone || got.Score != 75 || got.Recommendation != review.RequestChanges {
		t.Fatalf("review not saved: %+v", got)
	}
	if len(got.Findings) != 2 {
		t.Fatalf("len findings = %d, want 2", len(got.Findings))
	}
	// Order must be preserved.
	if got.Findings[0].File != "a.go" || !got.Findings[0].Blocking || got.Findings[1].File != "b.go" {
		t.Fatalf("findings order/content wrong: %+v", got.Findings)
	}
}

func TestReviewSaveReplacesFindings(t *testing.T) {
	ctx := context.Background()
	s, repoID := newReviewStore(t)

	rv := review.Review{ID: id.New(), RepoID: repoID, MRIID: 1, Status: review.StatusPending}
	_ = s.Create(ctx, rv)

	rv.Findings = []review.Finding{{Dimension: review.Risk, Severity: review.SeverityHigh}}
	_ = s.Save(ctx, rv)
	rv.Findings = []review.Finding{{Dimension: review.Readability, Severity: review.SeverityLow}}
	_ = s.Save(ctx, rv)

	got, _ := s.Get(ctx, rv.ID)
	if len(got.Findings) != 1 || got.Findings[0].Dimension != review.Readability {
		t.Fatalf("Save should replace findings, got %+v", got.Findings)
	}
}

func TestReviewSetStatus(t *testing.T) {
	ctx := context.Background()
	s, repoID := newReviewStore(t)
	rv := review.Review{ID: id.New(), RepoID: repoID, MRIID: 1, Status: review.StatusPending}
	_ = s.Create(ctx, rv)

	if err := s.SetStatus(ctx, rv.ID, review.StatusError, "boom"); err != nil {
		t.Fatalf("SetStatus: %v", err)
	}
	got, _ := s.Get(ctx, rv.ID)
	if got.Status != review.StatusError || got.Error != "boom" {
		t.Fatalf("status not updated: %+v", got)
	}
}

func TestReviewSetPhase(t *testing.T) {
	ctx := context.Background()
	s, repoID := newReviewStore(t)
	rv := review.Review{ID: id.New(), RepoID: repoID, MRIID: 1, Status: review.StatusRunning}
	if err := s.Create(ctx, rv); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := s.SetPhase(ctx, rv.ID, "reliability"); err != nil {
		t.Fatalf("SetPhase: %v", err)
	}
	got, _ := s.Get(ctx, rv.ID)
	if got.Phase != "reliability" {
		t.Fatalf("phase = %q, want reliability", got.Phase)
	}
}

func TestReviewSetArchived(t *testing.T) {
	ctx := context.Background()
	s, repoID := newReviewStore(t)
	rv := review.Review{ID: id.New(), RepoID: repoID, MRIID: 1, Status: review.StatusDone}
	if err := s.Create(ctx, rv); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := s.SetArchived(ctx, rv.ID, true); err != nil {
		t.Fatalf("SetArchived: %v", err)
	}
	got, _ := s.Get(ctx, rv.ID)
	if !got.Archived {
		t.Fatalf("Archived = %v, want true", got.Archived)
	}

	if err := s.SetArchived(ctx, rv.ID, false); err != nil {
		t.Fatalf("SetArchived unarchive: %v", err)
	}
	got, _ = s.Get(ctx, rv.ID)
	if got.Archived {
		t.Fatalf("Archived = %v, want false", got.Archived)
	}
}

func TestReviewSetArchivedMissing(t *testing.T) {
	s, _ := newReviewStore(t)
	if err := s.SetArchived(context.Background(), "nope", true); !errors.Is(err, review.ErrNotFound) {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
}

func TestReviewMarkSummaryPublished(t *testing.T) {
	ctx := context.Background()
	s, repoID := newReviewStore(t)
	rv := review.Review{ID: id.New(), RepoID: repoID, MRIID: 1, Status: review.StatusDone}
	if err := s.Create(ctx, rv); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, _ := s.Get(ctx, rv.ID)
	if got.SummaryPublished {
		t.Fatalf("SummaryPublished = %v, want false by default", got.SummaryPublished)
	}

	if err := s.MarkSummaryPublished(ctx, rv.ID); err != nil {
		t.Fatalf("MarkSummaryPublished: %v", err)
	}
	got, _ = s.Get(ctx, rv.ID)
	if !got.SummaryPublished {
		t.Fatalf("SummaryPublished = %v, want true", got.SummaryPublished)
	}
}

func TestReviewMarkSummaryPublishedMissing(t *testing.T) {
	s, _ := newReviewStore(t)
	if err := s.MarkSummaryPublished(context.Background(), "nope"); !errors.Is(err, review.ErrNotFound) {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
}

func TestReviewListArchivedByRepo(t *testing.T) {
	ctx := context.Background()
	s, repoID := newReviewStore(t)

	active := review.Review{ID: id.New(), RepoID: repoID, MRIID: 1, Status: review.StatusDone}
	_ = s.Create(ctx, active)
	archived := review.Review{ID: id.New(), RepoID: repoID, MRIID: 2, Status: review.StatusDone}
	_ = s.Create(ctx, archived)
	if err := s.SetArchived(ctx, archived.ID, true); err != nil {
		t.Fatalf("SetArchived: %v", err)
	}

	activeList, err := s.ListByRepo(ctx, repoID)
	if err != nil {
		t.Fatalf("ListByRepo: %v", err)
	}
	if len(activeList) != 1 || activeList[0].ID != active.ID {
		t.Fatalf("active list should exclude archived, got %+v", activeList)
	}

	archivedList, err := s.ListArchivedByRepo(ctx, repoID)
	if err != nil {
		t.Fatalf("ListArchivedByRepo: %v", err)
	}
	if len(archivedList) != 1 || archivedList[0].ID != archived.ID {
		t.Fatalf("archived list should include only archived, got %+v", archivedList)
	}
}

func TestReviewGetMissing(t *testing.T) {
	s, _ := newReviewStore(t)
	if _, err := s.Get(context.Background(), "nope"); !errors.Is(err, review.ErrNotFound) {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
}

func TestReviewDelete(t *testing.T) {
	ctx := context.Background()
	s, repoID := newReviewStore(t)

	rv := review.Review{ID: id.New(), RepoID: repoID, MRIID: 4, Status: review.StatusDone}
	if err := s.Create(ctx, rv); err != nil {
		t.Fatalf("Create: %v", err)
	}
	rv.Findings = []review.Finding{{Dimension: review.Risk, Severity: review.SeverityHigh, File: "a.go"}}
	if err := s.Save(ctx, rv); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := s.Delete(ctx, rv.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.Get(ctx, rv.ID); !errors.Is(err, review.ErrNotFound) {
		t.Fatalf("Get after delete = %v, want ErrNotFound", err)
	}
}

func TestReviewDeleteMissing(t *testing.T) {
	s, _ := newReviewStore(t)
	if err := s.Delete(context.Background(), "nope"); !errors.Is(err, review.ErrNotFound) {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
}

func TestReviewListByRepoNewestFirst(t *testing.T) {
	ctx := context.Background()
	s, repoID := newReviewStore(t)

	older := review.Review{ID: id.New(), RepoID: repoID, MRIID: 1, Status: review.StatusDone}
	_ = s.Create(ctx, older)
	time.Sleep(2 * time.Millisecond)
	newer := review.Review{ID: id.New(), RepoID: repoID, MRIID: 2, Status: review.StatusDone}
	_ = s.Create(ctx, newer)

	list, err := s.ListByRepo(ctx, repoID)
	if err != nil {
		t.Fatalf("ListByRepo: %v", err)
	}
	if len(list) != 2 || list[0].ID != newer.ID {
		t.Fatalf("expected newest first, got %+v", list)
	}
}
