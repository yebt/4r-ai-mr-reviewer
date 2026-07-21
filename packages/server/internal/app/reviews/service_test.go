package reviews

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

	rv, err := svc.Create(ctx, rp.ID, 7, review.ModeFast, "", "")
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

	rv, err := svc.Create(ctx, rp.ID, 7, review.ModeFast, "", "")
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

	rv, err := svc.Create(ctx, rp.ID, 7, review.ModeFast, "", "")
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

// TestArchiveRunningRejected asserts a non-terminal review cannot be archived
// (ErrNotArchivable), while a terminal one archives fine. The review row is
// inserted directly so the status is fully deterministic (no runner race).
func TestArchiveRunningRejected(t *testing.T) {
	ctx := context.Background()

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
	set, _ := skills.Load("")
	svc := NewService(reviewStore, sqlite.NewRepoStore(db), accountSvc, providerSvc, engine.NewMultiPass(set))

	acc, _ := accountSvc.Add(ctx, "acc", "https://gitlab.test", "token")
	prov, _ := providerSvc.Add(ctx, providers.AddInput{Name: "p", Kind: provider.KindOpenAICompat, BaseURL: "https://ai.test", Model: "m", APIKey: "k"})
	rp, _ := repoSvc.Add(ctx, appRepos.AddInput{Name: "web", URL: "https://gitlab.test/group/project", AccountID: acc.ID, ProviderID: prov.ID})

	// Both non-terminal states must be rejected: pending (job not started yet)
	// and running (job in flight). Each uses its own review row.
	for i, status := range []review.Status{review.StatusPending, review.StatusRunning} {
		rv := review.Review{
			ID:          fmt.Sprintf("rev-nonterminal-%d", i),
			RepoID:      rp.ID,
			MRIID:       9 + i,
			ContextMode: review.ModeFast,
			Status:      status,
		}
		if err := reviewStore.Create(ctx, rv); err != nil {
			t.Fatalf("Create(%s): %v", status, err)
		}
		if err := svc.Archive(ctx, rv.ID); !errors.Is(err, ErrNotArchivable) {
			t.Fatalf("Archive on %s = %v, want ErrNotArchivable", status, err)
		}
	}

	// A terminal review archives without complaint.
	done := review.Review{
		ID:          "rev-terminal",
		RepoID:      rp.ID,
		MRIID:       42,
		ContextMode: review.ModeFast,
		Status:      review.StatusDone,
	}
	if err := reviewStore.Create(ctx, done); err != nil {
		t.Fatalf("Create(done): %v", err)
	}
	if err := svc.Archive(ctx, done.ID); err != nil {
		t.Fatalf("Archive on terminal = %v, want nil", err)
	}
	got, err := reviewStore.Get(ctx, done.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !got.Archived {
		t.Fatalf("Archived = false, want true after archiving a terminal review")
	}
}

// modelCapturingAIStub serves a canned review and records the model field of
// the last /chat/completions request, so a test can assert which model the
// resolution precedence selected.
func modelCapturingAIStub(t *testing.T, reviewJSON string, got *string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/chat/completions") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		var body struct {
			Model string `json:"model"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		*got = body.Model
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"model":   body.Model,
			"choices": []map[string]any{{"message": map[string]any{"content": reviewJSON}}},
			"usage":   map[string]any{"prompt_tokens": 1, "completion_tokens": 1},
		})
	}))
}

// TestReviewModelOverride asserts a per-review model override takes precedence
// over the repo's and provider's model when resolving the run.
func TestReviewModelOverride(t *testing.T) {
	ctx := context.Background()

	gl := gitlabStub(t)
	defer gl.Close()
	var usedModel string
	aiSrv := modelCapturingAIStub(t, `{"summary":"clean","findings":[]}`, &usedModel)
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
	prov, _ := providerSvc.Add(ctx, providers.AddInput{Name: "p", Kind: provider.KindOpenAICompat, BaseURL: aiSrv.URL, Model: "provider-model", APIKey: "k"})
	rp, _ := repoSvc.Add(ctx, appRepos.AddInput{Name: "web", URL: "https://gitlab.test/group/project", AccountID: acc.ID, ProviderID: prov.ID})

	set, _ := skills.Load("")
	svc := NewService(reviewStore, sqlite.NewRepoStore(db), accountSvc, providerSvc, engine.New(set))
	runner := jobs.NewRunner(sqlite.NewJobStore(db), svc.Handle, jobs.WithLogger(log.New(io.Discard, "", 0)))
	svc.AttachRunner(runner)

	if _, err := svc.Create(ctx, rp.ID, 7, review.ModeFast, "", "override-model"); err != nil {
		t.Fatalf("Create: %v", err)
	}
	runner.Drain(ctx)

	if usedModel != "override-model" {
		t.Fatalf("model = %q, want override-model (per-review override must win)", usedModel)
	}
}

// TestRetryPreservesModelOverride guards the contract that Retry reloads the
// persisted per-review override rather than resetting it: a future Retry that
// called Create with empty overrides would fail here. It asserts both at the
// struct level and by observing the model the AI stub receives on the retried
// run.
func TestRetryPreservesModelOverride(t *testing.T) {
	ctx := context.Background()

	gl := gitlabStub(t)
	defer gl.Close()
	var usedModel string
	aiSrv := modelCapturingAIStub(t, `{"summary":"clean","findings":[]}`, &usedModel)
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
	prov, _ := providerSvc.Add(ctx, providers.AddInput{Name: "p", Kind: provider.KindOpenAICompat, BaseURL: aiSrv.URL, Model: "provider-model", APIKey: "k"})
	rp, _ := repoSvc.Add(ctx, appRepos.AddInput{Name: "web", URL: "https://gitlab.test/group/project", AccountID: acc.ID, ProviderID: prov.ID})

	set, _ := skills.Load("")
	svc := NewService(reviewStore, sqlite.NewRepoStore(db), accountSvc, providerSvc, engine.New(set))
	runner := jobs.NewRunner(sqlite.NewJobStore(db), svc.Handle, jobs.WithLogger(log.New(io.Discard, "", 0)))
	svc.AttachRunner(runner)

	original, err := svc.Create(ctx, rp.ID, 7, review.ModeFast, prov.ID, "override-model")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	runner.Drain(ctx)

	// Retry must carry the persisted override into the clone.
	clone, err := svc.Retry(ctx, original.ID)
	if err != nil {
		t.Fatalf("Retry: %v", err)
	}
	if clone.ProviderID != prov.ID || clone.Model != "override-model" {
		t.Fatalf("clone override = %q/%q, want %q/override-model", clone.ProviderID, clone.Model, prov.ID)
	}

	// Observe the model the retried run actually sends to the provider.
	usedModel = ""
	runner.Drain(ctx)
	if usedModel != "override-model" {
		t.Fatalf("retried run model = %q, want override-model (persisted override must survive retry)", usedModel)
	}
}

// fakeNotifier records every Notify call over a buffered channel so tests can
// observe the async, fire-and-forget notification without racing.
type fakeNotifier struct{ calls chan string }

func newFakeNotifier() *fakeNotifier { return &fakeNotifier{calls: make(chan string, 8)} }

func (f *fakeNotifier) Notify(_ context.Context, text string) error {
	f.calls <- text
	return nil
}

// recvNotification waits up to timeout for one notification. It returns ("",
// false) if none arrives — notifyFinished spawns a goroutine, so callers must
// wait rather than read synchronously.
func recvNotification(t *testing.T, f *fakeNotifier, timeout time.Duration) (string, bool) {
	t.Helper()
	select {
	case msg := <-f.calls:
		return msg, true
	case <-time.After(timeout):
		return "", false
	}
}

// TestNotifyFinishedDoneAndError covers the two message shapes directly. The
// failure text must never carry the raw error, which is sent to an external
// third party (Telegram).
func TestNotifyFinishedDoneAndError(t *testing.T) {
	fake := newFakeNotifier()
	svc := &Service{}
	svc.AttachNotifier(fake)

	rv := review.Review{
		MRIID: 42,
		Findings: []review.Finding{
			{Dimension: review.Risk, Severity: review.SeverityHigh},
			{Dimension: review.Readability, Severity: review.SeverityLow},
		},
	}

	svc.notifyFinished(rv, review.StatusDone, "")
	msg, ok := recvNotification(t, fake, time.Second)
	if !ok {
		t.Fatal("done: expected a notification, got none")
	}
	if !strings.Contains(msg, "finished") || !strings.Contains(msg, "2 finding(s)") {
		t.Fatalf("done text = %q, want it to mention 'finished' and '2 finding(s)'", msg)
	}
	// Exactly one message for one finished review.
	if _, extra := recvNotification(t, fake, 100*time.Millisecond); extra {
		t.Fatal("done: expected exactly one notification, got more")
	}

	svc.notifyFinished(rv, review.StatusError, "some secret detail")
	msg, ok = recvNotification(t, fake, time.Second)
	if !ok {
		t.Fatal("error: expected a notification, got none")
	}
	if msg != "Review !42 failed." {
		t.Fatalf("error text = %q, want %q", msg, "Review !42 failed.")
	}
	if strings.Contains(msg, "some secret detail") {
		t.Fatalf("error text leaked the raw error: %q", msg)
	}
}

// TestNotifyFinishedNilNotifier asserts that with no notifier attached the call
// is a silent no-op — it must not panic and must send nothing.
func TestNotifyFinishedNilNotifier(t *testing.T) {
	svc := &Service{} // notifier is nil
	// Must not panic.
	svc.notifyFinished(review.Review{MRIID: 1}, review.StatusDone, "")
	svc.notifyFinished(review.Review{MRIID: 1}, review.StatusError, "boom")
}

// TestHandleDoneFiresNotification drives one review to completion through the
// real runner and asserts exactly one notification fires on the DONE path.
//
// The cancel paths intentionally do NOT call notifyFinished (see Handle): that
// is enforced by code structure and covered by the cancel tests above, not by a
// racy runtime assertion here.
func TestHandleDoneFiresNotification(t *testing.T) {
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
	fake := newFakeNotifier()
	svc.AttachNotifier(fake)

	rv, err := svc.Create(ctx, rp.ID, 7, review.ModeFast, "", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	runner.Drain(ctx)

	got, err := reviewStore.Get(ctx, rv.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Status != review.StatusDone {
		t.Fatalf("status = %s (err=%q), want done", got.Status, got.Error)
	}

	msg, ok := recvNotification(t, fake, 2*time.Second)
	if !ok {
		t.Fatal("expected a notification on the done path, got none")
	}
	if !strings.Contains(msg, "finished") {
		t.Fatalf("notification text = %q, want it to mention 'finished'", msg)
	}
	if _, extra := recvNotification(t, fake, 150*time.Millisecond); extra {
		t.Fatal("expected exactly one notification for one finished review")
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

	original, _ := svc.Create(ctx, rp.ID, 7, review.ModeFast, "", "")
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
