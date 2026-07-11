package humanize

import (
	"context"
	"encoding/json"
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
	svc      *Service
	reviews  review.Repository
	profiles profile.Repository
	repoID   string
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
	svc := NewService(reviewStore, profileStore, providerSvc, log.New(io.Discard, "", 0))
	return fixture{svc: svc, reviews: reviewStore, profiles: profileStore, repoID: rp.ID}
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

const twoVariants = `{"variants":[
	{"summary":"che, quedó lindo","findings":[{"index":0,"text":"ojo con el secreto"},{"index":1,"text":"renombralo"}]},
	{"summary":"buen laburo","findings":[{"index":0,"text":"sacá el secreto"},{"index":1,"text":"mejor nombre"}]}
]}`

func TestHumanizeReturnsVariants(t *testing.T) {
	srv := aiStub(t, twoVariants)
	defer srv.Close()
	f := newFixture(t, srv.URL)

	reviewID := seedReview(t, f.reviews, f.repoID, review.StatusDone)
	profileID := seedProfile(t, f.profiles, profile.StyleStatusReady)

	vs, err := f.svc.Humanize(context.Background(), reviewID, profileID, 2)
	if err != nil {
		t.Fatalf("Humanize: %v", err)
	}
	if len(vs) != 2 {
		t.Fatalf("variants len = %d, want 2", len(vs))
	}
	for i, v := range vs {
		if v.Summary == "" {
			t.Fatalf("variant %d has empty summary", i)
		}
		if len(v.Findings) != 2 {
			t.Fatalf("variant %d findings len = %d, want 2", i, len(v.Findings))
		}
		for _, ft := range v.Findings {
			if ft.Text == "" {
				t.Fatalf("variant %d finding %d has empty text", i, ft.Index)
			}
			if ft.Index < 0 || ft.Index > 1 {
				t.Fatalf("variant %d finding index %d out of range", i, ft.Index)
			}
		}
	}
}

func TestHumanizeRejectsUnfinishedReview(t *testing.T) {
	srv := aiStub(t, twoVariants)
	defer srv.Close()
	f := newFixture(t, srv.URL)

	reviewID := seedReview(t, f.reviews, f.repoID, review.StatusRunning)
	profileID := seedProfile(t, f.profiles, profile.StyleStatusReady)

	if _, err := f.svc.Humanize(context.Background(), reviewID, profileID, 3); err == nil {
		t.Fatal("expected an error for a review that is not done")
	}
}

func TestHumanizeRejectsProfileNotReady(t *testing.T) {
	srv := aiStub(t, twoVariants)
	defer srv.Close()
	f := newFixture(t, srv.URL)

	reviewID := seedReview(t, f.reviews, f.repoID, review.StatusDone)
	profileID := seedProfile(t, f.profiles, profile.StyleStatusPending)

	_, err := f.svc.Humanize(context.Background(), reviewID, profileID, 3)
	if err == nil {
		t.Fatal("expected an error for a profile whose style guide is not ready")
	}
	if !strings.Contains(err.Error(), "pending") {
		t.Fatalf("error should include the actual status, got: %v", err)
	}
}

func TestHumanizeDropsOutOfRangeIndex(t *testing.T) {
	// The stub references index 9, which the review does not have; it must be
	// dropped while the valid index 0 survives.
	const withBadIndex = `{"variants":[
		{"summary":"s","findings":[{"index":0,"text":"ok"},{"index":9,"text":"ghost"}]}
	]}`
	srv := aiStub(t, withBadIndex)
	defer srv.Close()
	f := newFixture(t, srv.URL)

	reviewID := seedReview(t, f.reviews, f.repoID, review.StatusDone)
	profileID := seedProfile(t, f.profiles, profile.StyleStatusReady)

	vs, err := f.svc.Humanize(context.Background(), reviewID, profileID, 1)
	if err != nil {
		t.Fatalf("Humanize: %v", err)
	}
	if len(vs) != 1 {
		t.Fatalf("variants len = %d, want 1", len(vs))
	}
	if len(vs[0].Findings) != 1 {
		t.Fatalf("findings len = %d, want 1 (bad index dropped)", len(vs[0].Findings))
	}
	if vs[0].Findings[0].Index != 0 {
		t.Fatalf("surviving finding index = %d, want 0", vs[0].Findings[0].Index)
	}
}
