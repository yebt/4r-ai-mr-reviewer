package humanize

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/webcloster-dev/ai-reviewer/internal/adapters/crypto"
	"github.com/webcloster-dev/ai-reviewer/internal/adapters/sqlite"
	"github.com/webcloster-dev/ai-reviewer/internal/app/accounts"
	"github.com/webcloster-dev/ai-reviewer/internal/app/providers"
	apprepos "github.com/webcloster-dev/ai-reviewer/internal/app/repos"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/profile"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/provider"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/review"
	"github.com/webcloster-dev/ai-reviewer/internal/id"
)

// aiStub serves a canned OpenAI-compatible completion carrying the given text.
func aiStub(t *testing.T, content string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/chat/completions") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"model":   "m",
			"choices": []map[string]any{{"message": map[string]any{"content": content}}},
			"usage":   map[string]any{"prompt_tokens": 10, "completion_tokens": 5},
		})
	}))
}

type fixture struct {
	svc           *Service
	reviews       review.Repository
	profiles      profile.Repository
	humanizations review.HumanizationRepository
	repoID        string
}

// newFixture wires a humanize.Service over real sqlite stores and a real
// providers.Service whose default provider points at aiURL.
func newFixture(t *testing.T, aiURL string) fixture {
	t.Helper()
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	salt, _ := crypto.NewSalt()
	key, _ := crypto.DeriveKey("pw", salt)
	cipher, _ := crypto.NewCipher(key)
	secrets := sqlite.NewSecretStore(db, cipher)
	ctx := context.Background()

	accountSvc := accounts.NewService(sqlite.NewAccountRepo(db), secrets)
	providerSvc := providers.NewService(sqlite.NewProviderRepo(db), secrets)
	repoSvc := apprepos.NewService(sqlite.NewRepoStore(db), sqlite.NewAccountRepo(db), sqlite.NewProviderRepo(db))

	acc, err := accountSvc.Add(ctx, "acc", "https://gitlab.test", "token")
	if err != nil {
		t.Fatalf("add account: %v", err)
	}
	prov, err := providerSvc.Add(ctx, providers.AddInput{
		Name: "p", Kind: provider.KindOpenAICompat, BaseURL: aiURL, Model: "m", APIKey: "k",
	})
	if err != nil {
		t.Fatalf("add provider: %v", err)
	}
	rp, err := repoSvc.Add(ctx, apprepos.AddInput{
		Name: "web", URL: "https://gitlab.test/group/project", AccountID: acc.ID, ProviderID: prov.ID,
	})
	if err != nil {
		t.Fatalf("add repo: %v", err)
	}

	reviewStore := sqlite.NewReviewStore(db)
	profileStore := sqlite.NewProfileStore(db)
	humanizationStore := sqlite.NewHumanizationStore(db)
	svc := NewService(reviewStore, profileStore, humanizationStore, providerSvc, log.New(io.Discard, "", 0))
	return fixture{svc: svc, reviews: reviewStore, profiles: profileStore, humanizations: humanizationStore, repoID: rp.ID}
}

// seedReview inserts a review with the given status and two findings.
func seedReview(t *testing.T, repo review.Repository, repoID string, status review.Status) string {
	t.Helper()
	ctx := context.Background()
	rv := review.Review{ID: id.New(), RepoID: repoID, MRIID: 1, Status: review.StatusPending}
	if err := repo.Create(ctx, rv); err != nil {
		t.Fatalf("create review: %v", err)
	}
	rv.Status = status
	rv.Summary = "Overall solid."
	rv.Findings = []review.Finding{
		{Dimension: review.Risk, Severity: review.SeverityHigh, File: "auth.go", Line: 42, Issue: "hardcoded secret", Why: "leaks", Fix: "use env", Blocking: true},
		{Dimension: review.Readability, Severity: review.SeverityLow, File: "util.go", Issue: "unclear name", Why: "hurts", Fix: "rename"},
	}
	if err := repo.Save(ctx, rv); err != nil {
		t.Fatalf("save review: %v", err)
	}
	return rv.ID
}

// seedProfile inserts a profile and sets its style-guide status.
func seedProfile(t *testing.T, repo profile.Repository, status string) string {
	t.Helper()
	ctx := context.Background()
	p := profile.Profile{ID: id.New(), Name: "voice", Samples: []string{"che"}}
	if err := repo.Create(ctx, p); err != nil {
		t.Fatalf("create profile: %v", err)
	}
	if err := repo.SetStyleGuide(ctx, p.ID, "Voice: warm, uses voseo.", status, ""); err != nil {
		t.Fatalf("set style guide: %v", err)
	}
	return p.ID
}

const findingJSON = `{"issue":"che, acá el token está hardcodeado","why":"filtra credenciales","fix":"leelo del env"}`

const summaryJSON = `{"summary":"che, en gral quedó lindo"}`

func TestHumanizeFindingReturnsStructured(t *testing.T) {
	srv := aiStub(t, findingJSON)
	defer srv.Close()
	f := newFixture(t, srv.URL)

	reviewID := seedReview(t, f.reviews, f.repoID, review.StatusDone)
	profileID := seedProfile(t, f.profiles, profile.StyleStatusReady)

	fh, err := f.svc.HumanizeFinding(context.Background(), reviewID, profileID, 0)
	if err != nil {
		t.Fatalf("HumanizeFinding: %v", err)
	}
	if fh.Issue != "che, acá el token está hardcodeado" {
		t.Fatalf("issue = %q", fh.Issue)
	}
	if fh.Why != "filtra credenciales" || fh.Fix != "leelo del env" {
		t.Fatalf("parts not parsed: %+v", fh)
	}
}

func TestHumanizeSummaryReturnsStructured(t *testing.T) {
	srv := aiStub(t, summaryJSON)
	defer srv.Close()
	f := newFixture(t, srv.URL)

	reviewID := seedReview(t, f.reviews, f.repoID, review.StatusDone)
	profileID := seedProfile(t, f.profiles, profile.StyleStatusReady)

	sh, err := f.svc.HumanizeSummary(context.Background(), reviewID, profileID)
	if err != nil {
		t.Fatalf("HumanizeSummary: %v", err)
	}
	if sh.Summary != "che, en gral quedó lindo" {
		t.Fatalf("summary = %q", sh.Summary)
	}
}

func TestHumanizePersistsRuns(t *testing.T) {
	srv := aiStub(t, findingJSON)
	defer srv.Close()
	f := newFixture(t, srv.URL)

	reviewID := seedReview(t, f.reviews, f.repoID, review.StatusDone)
	profileID := seedProfile(t, f.profiles, profile.StyleStatusReady)
	ctx := context.Background()

	// Two finding runs on index 0 must accumulate as tabs 0 and 1.
	if _, err := f.svc.HumanizeFinding(ctx, reviewID, profileID, 0); err != nil {
		t.Fatalf("HumanizeFinding #1: %v", err)
	}
	if _, err := f.svc.HumanizeFinding(ctx, reviewID, profileID, 0); err != nil {
		t.Fatalf("HumanizeFinding #2: %v", err)
	}

	hs, err := f.humanizations.ListByReview(ctx, reviewID)
	if err != nil {
		t.Fatalf("ListByReview: %v", err)
	}
	if len(hs) != 2 {
		t.Fatalf("persisted runs = %d, want 2", len(hs))
	}
	for i, h := range hs {
		if h.Target != review.HumanizationFinding || h.FindingIndex != 0 {
			t.Fatalf("run %d: target=%q findingIndex=%d", i, h.Target, h.FindingIndex)
		}
		if h.TabIndex != i {
			t.Fatalf("run %d: tabIndex = %d, want %d", i, h.TabIndex, i)
		}
		if h.Issue != "che, acá el token está hardcodeado" {
			t.Fatalf("run %d: issue = %q", i, h.Issue)
		}
	}
}

func TestHumanizeSummaryPersistsRun(t *testing.T) {
	srv := aiStub(t, summaryJSON)
	defer srv.Close()
	f := newFixture(t, srv.URL)

	reviewID := seedReview(t, f.reviews, f.repoID, review.StatusDone)
	profileID := seedProfile(t, f.profiles, profile.StyleStatusReady)
	ctx := context.Background()

	if _, err := f.svc.HumanizeSummary(ctx, reviewID, profileID); err != nil {
		t.Fatalf("HumanizeSummary: %v", err)
	}

	hs, err := f.humanizations.ListByReview(ctx, reviewID)
	if err != nil {
		t.Fatalf("ListByReview: %v", err)
	}
	if len(hs) != 1 {
		t.Fatalf("persisted runs = %d, want 1", len(hs))
	}
	if hs[0].Target != review.HumanizationSummary || hs[0].FindingIndex != review.SummaryFindingIndex {
		t.Fatalf("target=%q findingIndex=%d", hs[0].Target, hs[0].FindingIndex)
	}
	if hs[0].Summary != "che, en gral quedó lindo" {
		t.Fatalf("summary = %q", hs[0].Summary)
	}
}

func TestHumanizeFindingOutOfRange(t *testing.T) {
	srv := aiStub(t, findingJSON)
	defer srv.Close()
	f := newFixture(t, srv.URL)

	reviewID := seedReview(t, f.reviews, f.repoID, review.StatusDone)
	profileID := seedProfile(t, f.profiles, profile.StyleStatusReady)

	if _, err := f.svc.HumanizeFinding(context.Background(), reviewID, profileID, 9); !errors.Is(err, ErrFindingIndexOutOfRange) {
		t.Fatalf("expected ErrFindingIndexOutOfRange, got: %v", err)
	}
	if _, err := f.svc.HumanizeFinding(context.Background(), reviewID, profileID, -1); !errors.Is(err, ErrFindingIndexOutOfRange) {
		t.Fatalf("expected ErrFindingIndexOutOfRange for negative index, got: %v", err)
	}
}

func TestHumanizeRejectsUnfinishedReview(t *testing.T) {
	srv := aiStub(t, findingJSON)
	defer srv.Close()
	f := newFixture(t, srv.URL)

	reviewID := seedReview(t, f.reviews, f.repoID, review.StatusRunning)
	profileID := seedProfile(t, f.profiles, profile.StyleStatusReady)

	if _, err := f.svc.HumanizeFinding(context.Background(), reviewID, profileID, 0); !errors.Is(err, ErrReviewNotDone) {
		t.Fatalf("expected ErrReviewNotDone for HumanizeFinding, got: %v", err)
	}
	if _, err := f.svc.HumanizeSummary(context.Background(), reviewID, profileID); !errors.Is(err, ErrReviewNotDone) {
		t.Fatalf("expected ErrReviewNotDone for HumanizeSummary, got: %v", err)
	}
}

func TestHumanizeRejectsProfileNotReady(t *testing.T) {
	srv := aiStub(t, findingJSON)
	defer srv.Close()
	f := newFixture(t, srv.URL)

	reviewID := seedReview(t, f.reviews, f.repoID, review.StatusDone)
	profileID := seedProfile(t, f.profiles, profile.StyleStatusPending)

	_, err := f.svc.HumanizeFinding(context.Background(), reviewID, profileID, 0)
	if !errors.Is(err, ErrStyleGuideNotReady) {
		t.Fatalf("expected ErrStyleGuideNotReady, got: %v", err)
	}
	if !strings.Contains(err.Error(), "pending") {
		t.Fatalf("error should include the actual status, got: %v", err)
	}
}
