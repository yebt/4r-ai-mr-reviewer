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

// TestReviewAwaitingApprovalPersistsAndIsActive asserts a held (awaiting_approval)
// review round-trips through the store and counts as in-flight for the webhook
// duplicate guard, so a repeated delivery does not pile up a second held review.
func TestReviewAwaitingApprovalPersistsAndIsActive(t *testing.T) {
	ctx := context.Background()
	s, repoID := newReviewStore(t)

	rv := review.Review{ID: id.New(), RepoID: repoID, MRIID: 7, Status: review.StatusAwaitingApproval}
	if err := s.Create(ctx, rv); err != nil {
		t.Fatalf("Create: %v", err)
	}
	got, err := s.Get(ctx, rv.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Status != review.StatusAwaitingApproval {
		t.Fatalf("status = %s, want awaiting_approval", got.Status)
	}

	// A held review makes the MR active (guards against duplicate held reviews).
	if active, err := s.HasActiveForMR(ctx, repoID, 7); err != nil || !active {
		t.Fatalf("HasActiveForMR (awaiting_approval) = (%v, %v), want (true, nil)", active, err)
	}
}

// TestReviewHasActiveForMR asserts a pending or running review makes the MR
// "active", while a terminal (done/error/cancelled) review or no review at all
// does not.
func TestReviewHasActiveForMR(t *testing.T) {
	ctx := context.Background()
	s, repoID := newReviewStore(t)

	// No review yet → not active.
	if active, err := s.HasActiveForMR(ctx, repoID, 7); err != nil || active {
		t.Fatalf("HasActiveForMR (none) = (%v, %v), want (false, nil)", active, err)
	}

	// Pending → active.
	pending := review.Review{ID: id.New(), RepoID: repoID, MRIID: 7, Status: review.StatusPending}
	if err := s.Create(ctx, pending); err != nil {
		t.Fatalf("Create pending: %v", err)
	}
	if active, err := s.HasActiveForMR(ctx, repoID, 7); err != nil || !active {
		t.Fatalf("HasActiveForMR (pending) = (%v, %v), want (true, nil)", active, err)
	}

	// A different MR is unaffected.
	if active, err := s.HasActiveForMR(ctx, repoID, 8); err != nil || active {
		t.Fatalf("HasActiveForMR (other MR) = (%v, %v), want (false, nil)", active, err)
	}

	// Running → active.
	if err := s.SetStatus(ctx, pending.ID, review.StatusRunning, ""); err != nil {
		t.Fatalf("SetStatus running: %v", err)
	}
	if active, err := s.HasActiveForMR(ctx, repoID, 7); err != nil || !active {
		t.Fatalf("HasActiveForMR (running) = (%v, %v), want (true, nil)", active, err)
	}

	// Terminal (done) → not active.
	if err := s.SetStatus(ctx, pending.ID, review.StatusDone, ""); err != nil {
		t.Fatalf("SetStatus done: %v", err)
	}
	if active, err := s.HasActiveForMR(ctx, repoID, 7); err != nil || active {
		t.Fatalf("HasActiveForMR (done) = (%v, %v), want (false, nil)", active, err)
	}
}

func TestReviewProviderModelRoundTrip(t *testing.T) {
	ctx := context.Background()
	s, repoID := newReviewStore(t)

	rv := review.Review{ID: id.New(), RepoID: repoID, MRIID: 7, Status: review.StatusPending, ProviderID: "prov-1", Model: "gpt-x"}
	if err := s.Create(ctx, rv); err != nil {
		t.Fatalf("Create: %v", err)
	}
	got, err := s.Get(ctx, rv.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ProviderID != "prov-1" || got.Model != "gpt-x" {
		t.Fatalf("provider/model not persisted: %+v", got)
	}
}

func TestReviewBranchRoundTrip(t *testing.T) {
	ctx := context.Background()
	s, repoID := newReviewStore(t)

	// Create defaults to empty branches (a review only learns them once it runs).
	rv := review.Review{ID: id.New(), RepoID: repoID, MRIID: 7, Status: review.StatusPending}
	if err := s.Create(ctx, rv); err != nil {
		t.Fatalf("Create: %v", err)
	}
	got, err := s.Get(ctx, rv.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.SourceBranch != "" || got.TargetBranch != "" {
		t.Fatalf("branches should default empty: %+v", got)
	}

	// Save persists the branches captured at run time.
	rv.Status = review.StatusDone
	rv.SourceBranch = "feature/login"
	rv.TargetBranch = "main"
	if err := s.Save(ctx, rv); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err = s.Get(ctx, rv.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.SourceBranch != "feature/login" || got.TargetBranch != "main" {
		t.Fatalf("branches not persisted: source=%q target=%q", got.SourceBranch, got.TargetBranch)
	}
}

func TestReviewSetBranches(t *testing.T) {
	ctx := context.Background()
	s, repoID := newReviewStore(t)

	rv := review.Review{ID: id.New(), RepoID: repoID, MRIID: 9, Status: review.StatusRunning}
	if err := s.Create(ctx, rv); err != nil {
		t.Fatalf("Create: %v", err)
	}
	// SetBranches records the branch before the review completes, so it survives
	// a later failure (which never reaches the success Save).
	if err := s.SetBranches(ctx, rv.ID, "fix/bug", "develop"); err != nil {
		t.Fatalf("SetBranches: %v", err)
	}
	got, err := s.Get(ctx, rv.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.SourceBranch != "fix/bug" || got.TargetBranch != "develop" {
		t.Fatalf("SetBranches not persisted: source=%q target=%q", got.SourceBranch, got.TargetBranch)
	}
	if err := s.SetBranches(ctx, "nope", "a", "b"); !errors.Is(err, review.ErrNotFound) {
		t.Fatalf("SetBranches(unknown) = %v, want ErrNotFound", err)
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

func TestReviewSetReasoningUpsert(t *testing.T) {
	ctx := context.Background()
	s, repoID := newReviewStore(t)
	rv := review.Review{ID: id.New(), RepoID: repoID, MRIID: 1, Status: review.StatusRunning}
	if err := s.Create(ctx, rv); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := s.SetReasoning(ctx, rv.ID, "risk", "first thoughts"); err != nil {
		t.Fatalf("SetReasoning insert: %v", err)
	}
	if err := s.SetReasoning(ctx, rv.ID, "risk", "revised thoughts"); err != nil {
		t.Fatalf("SetReasoning update: %v", err)
	}
	if err := s.SetReasoning(ctx, rv.ID, "readability", "naming notes"); err != nil {
		t.Fatalf("SetReasoning second phase: %v", err)
	}

	got, err := s.Get(ctx, rv.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	// Upsert: one row per phase; the re-run replaces content, so two phases total.
	if len(got.Reasonings) != 2 {
		t.Fatalf("reasonings = %d, want 2: %+v", len(got.Reasonings), got.Reasonings)
	}
	byPhase := map[string]string{}
	for _, r := range got.Reasonings {
		byPhase[r.Phase] = r.Content
	}
	if byPhase["risk"] != "revised thoughts" {
		t.Fatalf("risk reasoning = %q, want revised thoughts", byPhase["risk"])
	}
	if byPhase["readability"] != "naming notes" {
		t.Fatalf("readability reasoning = %q, want naming notes", byPhase["readability"])
	}
}

func TestReviewDeleteCascadesReasonings(t *testing.T) {
	ctx := context.Background()
	s, repoID := newReviewStore(t)
	rv := review.Review{ID: id.New(), RepoID: repoID, MRIID: 1, Status: review.StatusDone}
	if err := s.Create(ctx, rv); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := s.SetReasoning(ctx, rv.ID, "risk", "thoughts"); err != nil {
		t.Fatalf("SetReasoning: %v", err)
	}

	if err := s.Delete(ctx, rv.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	var n int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM review_reasonings WHERE review_id = ?`, rv.ID).Scan(&n); err != nil {
		t.Fatalf("count reasonings: %v", err)
	}
	if n != 0 {
		t.Fatalf("reasonings after delete = %d, want 0", n)
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

// TestReviewSetRawOutput asserts the captured raw model output round-trips
// through Get, and that setting it on a missing review returns ErrNotFound.
func TestReviewSetRawOutput(t *testing.T) {
	ctx := context.Background()
	s, repoID := newReviewStore(t)

	rv := review.Review{ID: id.New(), RepoID: repoID, MRIID: 7, Status: review.StatusError}
	if err := s.Create(ctx, rv); err != nil {
		t.Fatalf("Create: %v", err)
	}

	const raw = "I could not produce JSON, sorry."
	if err := s.SetRawOutput(ctx, rv.ID, raw); err != nil {
		t.Fatalf("SetRawOutput: %v", err)
	}
	got, err := s.Get(ctx, rv.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.RawOutput != raw {
		t.Fatalf("RawOutput = %q, want %q", got.RawOutput, raw)
	}

	if err := s.SetRawOutput(ctx, "missing-id", raw); !errors.Is(err, review.ErrNotFound) {
		t.Fatalf("SetRawOutput(missing) = %v, want ErrNotFound", err)
	}
}

// TestReviewSetFailure asserts SetFailure persists the error, raw output, and
// token counts in one write, all round-tripping through Get, and that setting it
// on a missing review returns ErrNotFound.
func TestReviewSetFailure(t *testing.T) {
	ctx := context.Background()
	s, repoID := newReviewStore(t)

	rv := review.Review{ID: id.New(), RepoID: repoID, MRIID: 7, Status: review.StatusRunning}
	if err := s.Create(ctx, rv); err != nil {
		t.Fatalf("Create: %v", err)
	}

	const (
		errMsg = "engine: readability pass: the model returned an empty response"
		raw    = ""
	)
	if err := s.SetFailure(ctx, rv.ID, errMsg, raw, 1200, 0); err != nil {
		t.Fatalf("SetFailure: %v", err)
	}
	got, err := s.Get(ctx, rv.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Status != review.StatusError {
		t.Fatalf("status = %s, want error", got.Status)
	}
	if got.Error != errMsg {
		t.Fatalf("error = %q, want %q", got.Error, errMsg)
	}
	if got.RawOutput != raw {
		t.Fatalf("RawOutput = %q, want %q", got.RawOutput, raw)
	}
	if got.InputTokens != 1200 || got.OutputTokens != 0 {
		t.Fatalf("tokens = %d/%d, want 1200/0", got.InputTokens, got.OutputTokens)
	}

	if err := s.SetFailure(ctx, "missing-id", errMsg, raw, 0, 0); !errors.Is(err, review.ErrNotFound) {
		t.Fatalf("SetFailure(missing) = %v, want ErrNotFound", err)
	}
}
