package reviews

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
	appRepos "github.com/webcloster-dev/ai-reviewer/internal/app/repos"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/provider"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/review"
	"github.com/webcloster-dev/ai-reviewer/internal/jobs"
	"github.com/webcloster-dev/ai-reviewer/internal/review/engine"
	"github.com/webcloster-dev/ai-reviewer/internal/review/skills"
)

// gitlabStub serves canned MR changes.
func gitlabStub(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/changes") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"iid":           7,
			"title":         "Add login",
			"source_branch": "feat",
			"changes": []map[string]any{
				{"old_path": "a.go", "new_path": "a.go", "diff": "@@ -1 +1 @@\n-old\n+new"},
			},
		})
	}))
}

// aiStub serves a canned OpenAI-compatible completion carrying a JSON review.
func aiStub(t *testing.T, reviewJSON string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/chat/completions") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"model":   "m",
			"choices": []map[string]any{{"message": map[string]any{"content": reviewJSON}}},
			"usage":   map[string]any{"prompt_tokens": 50, "completion_tokens": 20},
		})
	}))
}

func TestReviewEndToEnd(t *testing.T) {
	ctx := context.Background()

	gl := gitlabStub(t)
	defer gl.Close()
	reviewJSON := `{"summary":"one blocker","findings":[{"dimension":"risk","severity":"high","file":"a.go","line":3,"issue":"hardcoded secret","why":"leak","fix":"use env","blocking":true}]}`
	aiSrv := aiStub(t, reviewJSON)
	defer aiSrv.Close()

	db, err := sqlite.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	salt, _ := crypto.NewSalt()
	key, _ := crypto.DeriveKey("pw", salt)
	cipher, _ := crypto.NewCipher(key)
	secrets := sqlite.NewSecretStore(db, cipher)

	accountSvc := accounts.NewService(sqlite.NewAccountRepo(db), secrets)
	providerSvc := providers.NewService(sqlite.NewProviderRepo(db), secrets)
	repoSvc := appRepos.NewService(sqlite.NewRepoStore(db), sqlite.NewAccountRepo(db), sqlite.NewProviderRepo(db))
	reviewStore := sqlite.NewReviewStore(db)

	acc, err := accountSvc.Add(ctx, "acc", gl.URL, "token")
	if err != nil {
		t.Fatalf("add account: %v", err)
	}
	prov, err := providerSvc.Add(ctx, providers.AddInput{
		Name: "p", Kind: provider.KindOpenAICompat, BaseURL: aiSrv.URL, Model: "m", APIKey: "k",
	})
	if err != nil {
		t.Fatalf("add provider: %v", err)
	}
	rp, err := repoSvc.Add(ctx, appRepos.AddInput{
		Name: "web", URL: "https://gitlab.test/group/project", AccountID: acc.ID, ProviderID: prov.ID,
	})
	if err != nil {
		t.Fatalf("add repo: %v", err)
	}

	set, _ := skills.Load("")
	svc := NewService(reviewStore, sqlite.NewRepoStore(db), accountSvc, providerSvc, engine.New(set))
	runner := jobs.NewRunner(sqlite.NewJobStore(db), svc.Handle, jobs.WithLogger(log.New(io.Discard, "", 0)))
	svc.AttachRunner(runner)

	rv, err := svc.Create(ctx, rp.ID, 7, review.ModeFast)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Drain claims the enqueued job and runs the handler synchronously.
	runner.Drain(ctx)

	got, err := reviewStore.Get(ctx, rv.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Status != review.StatusDone {
		t.Fatalf("status = %s (err=%q), want done", got.Status, got.Error)
	}
	if got.Recommendation != review.RequestChanges {
		t.Fatalf("recommendation = %s, want request_changes", got.Recommendation)
	}
	if len(got.Findings) != 1 || got.Findings[0].File != "a.go" || !got.Findings[0].Blocking {
		t.Fatalf("findings not persisted correctly: %+v", got.Findings)
	}
	if got.InputTokens != 50 || got.OutputTokens != 20 {
		t.Fatalf("tokens = %d/%d, want 50/20", got.InputTokens, got.OutputTokens)
	}
}

// TestCancelPendingBeforeRun cancels a pending review, then runs the handler
// directly: the fast path must mark it cancelled and return nil (no retry),
// without ever reaching the engine.
func TestCancelPendingBeforeRun(t *testing.T) {
	ctx := context.Background()

	gl := gitlabStub(t)
	defer gl.Close()
	aiSrv := aiStub(t, `{"findings":[]}`)
	defer aiSrv.Close()

	db, err := sqlite.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	salt, _ := crypto.NewSalt()
	key, _ := crypto.DeriveKey("pw", salt)
	cipher, _ := crypto.NewCipher(key)
	secrets := sqlite.NewSecretStore(db, cipher)
	accountSvc := accounts.NewService(sqlite.NewAccountRepo(db), secrets)
	providerSvc := providers.NewService(sqlite.NewProviderRepo(db), secrets)
	repoSvc := appRepos.NewService(sqlite.NewRepoStore(db), sqlite.NewAccountRepo(db), sqlite.NewProviderRepo(db))
	reviewStore := sqlite.NewReviewStore(db)

	acc, _ := accountSvc.Add(ctx, "acc", gl.URL, "token")
	prov, _ := providerSvc.Add(ctx, providers.AddInput{Name: "p", Kind: provider.KindOpenAICompat, BaseURL: aiSrv.URL, Model: "m", APIKey: "k"})
	rp, _ := repoSvc.Add(ctx, appRepos.AddInput{Name: "web", URL: "https://gitlab.test/group/project", AccountID: acc.ID, ProviderID: prov.ID})

	set, _ := skills.Load("")
	svc := NewService(reviewStore, sqlite.NewRepoStore(db), accountSvc, providerSvc, engine.NewMultiPass(set))
	runner := jobs.NewRunner(sqlite.NewJobStore(db), svc.Handle, jobs.WithLogger(log.New(io.Discard, "", 0)))
	svc.AttachRunner(runner)

	rv, err := svc.Create(ctx, rp.ID, 7, review.ModeFast)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := svc.Cancel(ctx, rv.ID); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	if err := svc.Handle(ctx, rv.ID); err != nil {
		t.Fatalf("Handle after cancel = %v, want nil (no retry)", err)
	}

	got, err := reviewStore.Get(ctx, rv.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Status != review.StatusCancelled {
		t.Fatalf("status = %s, want cancelled", got.Status)
	}
	if got.Phase != "" {
		t.Fatalf("phase = %q, want empty", got.Phase)
	}
}

// TestCancelTerminalRejected asserts Cancel refuses a review already in a
// terminal state.
func TestCancelTerminalRejected(t *testing.T) {
	ctx := context.Background()

	gl := gitlabStub(t)
	defer gl.Close()
	aiSrv := aiStub(t, `{"findings":[]}`)
	defer aiSrv.Close()

	db, err := sqlite.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	salt, _ := crypto.NewSalt()
	key, _ := crypto.DeriveKey("pw", salt)
	cipher, _ := crypto.NewCipher(key)
	secrets := sqlite.NewSecretStore(db, cipher)
	accountSvc := accounts.NewService(sqlite.NewAccountRepo(db), secrets)
	providerSvc := providers.NewService(sqlite.NewProviderRepo(db), secrets)
	repoSvc := appRepos.NewService(sqlite.NewRepoStore(db), sqlite.NewAccountRepo(db), sqlite.NewProviderRepo(db))
	reviewStore := sqlite.NewReviewStore(db)

	acc, _ := accountSvc.Add(ctx, "acc", gl.URL, "token")
	prov, _ := providerSvc.Add(ctx, providers.AddInput{Name: "p", Kind: provider.KindOpenAICompat, BaseURL: aiSrv.URL, Model: "m", APIKey: "k"})
	rp, _ := repoSvc.Add(ctx, appRepos.AddInput{Name: "web", URL: "https://gitlab.test/group/project", AccountID: acc.ID, ProviderID: prov.ID})

	set, _ := skills.Load("")
	svc := NewService(reviewStore, sqlite.NewRepoStore(db), accountSvc, providerSvc, engine.NewMultiPass(set))
	runner := jobs.NewRunner(sqlite.NewJobStore(db), svc.Handle, jobs.WithLogger(log.New(io.Discard, "", 0)))
	svc.AttachRunner(runner)

	rv, err := svc.Create(ctx, rp.ID, 7, review.ModeFast)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	// Force a terminal state directly on the store.
	if err := reviewStore.SetStatus(ctx, rv.ID, review.StatusDone, ""); err != nil {
		t.Fatalf("SetStatus: %v", err)
	}

	if err := svc.Cancel(ctx, rv.ID); !errors.Is(err, ErrNotCancelable) {
		t.Fatalf("Cancel on terminal = %v, want ErrNotCancelable", err)
	}
}

func TestRetryClonesReview(t *testing.T) {
	ctx := context.Background()

	gl := gitlabStub(t)
	defer gl.Close()
	aiSrv := aiStub(t, `{"summary":"clean","findings":[]}`)
	defer aiSrv.Close()

	db, err := sqlite.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	salt, _ := crypto.NewSalt()
	key, _ := crypto.DeriveKey("pw", salt)
	cipher, _ := crypto.NewCipher(key)
	secrets := sqlite.NewSecretStore(db, cipher)
	accountSvc := accounts.NewService(sqlite.NewAccountRepo(db), secrets)
	providerSvc := providers.NewService(sqlite.NewProviderRepo(db), secrets)
	repoSvc := appRepos.NewService(sqlite.NewRepoStore(db), sqlite.NewAccountRepo(db), sqlite.NewProviderRepo(db))
	reviewStore := sqlite.NewReviewStore(db)

	acc, _ := accountSvc.Add(ctx, "acc", gl.URL, "token")
	prov, _ := providerSvc.Add(ctx, providers.AddInput{Name: "p", Kind: provider.KindOpenAICompat, BaseURL: aiSrv.URL, Model: "m", APIKey: "k"})
	rp, _ := repoSvc.Add(ctx, appRepos.AddInput{Name: "web", URL: "https://gitlab.test/group/project", AccountID: acc.ID, ProviderID: prov.ID})

	set, _ := skills.Load("")
	svc := NewService(reviewStore, sqlite.NewRepoStore(db), accountSvc, providerSvc, engine.New(set))
	runner := jobs.NewRunner(sqlite.NewJobStore(db), svc.Handle, jobs.WithLogger(log.New(io.Discard, "", 0)))
	svc.AttachRunner(runner)

	original, _ := svc.Create(ctx, rp.ID, 7, review.ModeFast)
	clone, err := svc.Retry(ctx, original.ID)
	if err != nil {
		t.Fatalf("Retry: %v", err)
	}
	if clone.ID == original.ID {
		t.Fatal("retry must create a new review, not reuse the id")
	}

	list, _ := reviewStore.ListByRepo(ctx, rp.ID)
	if len(list) != 2 {
		t.Fatalf("expected original + clone = 2 reviews, got %d", len(list))
	}
}
