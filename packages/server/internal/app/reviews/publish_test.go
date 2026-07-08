package reviews

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
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

// recordingGitLab serves /changes (with diff_refs) and counts posted notes and
// inline discussions.
type recordingGitLab struct {
	*httptest.Server
	mu          sync.Mutex
	notes       int
	discussions int
}

func newRecordingGitLab(t *testing.T) *recordingGitLab {
	g := &recordingGitLab{}
	g.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/changes"):
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"iid":           7,
				"title":         "Add login",
				"source_branch": "feat",
				"diff_refs":     map[string]any{"base_sha": "b", "start_sha": "s", "head_sha": "h"},
				"changes":       []map[string]any{{"old_path": "a.go", "new_path": "a.go", "diff": "@@ @@"}},
			})
		case strings.HasSuffix(r.URL.Path, "/discussions"):
			g.mu.Lock()
			g.discussions++
			g.mu.Unlock()
			w.WriteHeader(http.StatusCreated)
		case strings.HasSuffix(r.URL.Path, "/notes"):
			g.mu.Lock()
			g.notes++
			g.mu.Unlock()
			w.WriteHeader(http.StatusCreated)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(g.Close)
	return g
}

func TestPublishSelectedFindings(t *testing.T) {
	ctx := context.Background()

	gl := newRecordingGitLab(t)
	// One inline finding (file+line) and one general finding (no line).
	reviewJSON := `{"summary":"issues","findings":[
		{"dimension":"risk","severity":"high","file":"a.go","line":3,"issue":"secret","blocking":true},
		{"dimension":"readability","severity":"low","file":"","line":0,"issue":"vague description"}
	]}`
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

	acc, _ := accountSvc.Add(ctx, "acc", gl.URL, "token")
	prov, _ := providerSvc.Add(ctx, providers.AddInput{Name: "p", Kind: provider.KindOpenAICompat, BaseURL: aiSrv.URL, Model: "m", APIKey: "k"})
	rp, _ := repoSvc.Add(ctx, appRepos.AddInput{Name: "web", URL: "https://gitlab.test/group/project", AccountID: acc.ID, ProviderID: prov.ID})

	set, _ := skills.Load("")
	svc := NewService(reviewStore, sqlite.NewRepoStore(db), accountSvc, providerSvc, engine.New(set))
	runner := jobs.NewRunner(sqlite.NewJobStore(db), svc.Handle, jobs.WithLogger(log.New(io.Discard, "", 0)))
	svc.AttachRunner(runner)

	rv, _ := svc.Create(ctx, rp.ID, 7, review.ModeFast)
	runner.Drain(ctx)

	if err := svc.Publish(ctx, rv.ID, Selection{All: true}); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	gl.mu.Lock()
	notes, discussions := gl.notes, gl.discussions
	gl.mu.Unlock()

	// 1 summary note + 1 note for the line-less finding = 2 notes; 1 inline.
	if notes != 2 || discussions != 1 {
		t.Fatalf("posts = %d notes / %d discussions, want 2 / 1", notes, discussions)
	}

	got, _ := reviewStore.Get(ctx, rv.ID)
	for i, f := range got.Findings {
		if !f.Published {
			t.Fatalf("finding %d not marked published", i)
		}
	}
}

func TestPublishRejectsUnfinishedReview(t *testing.T) {
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
	reviewStore := sqlite.NewReviewStore(db)

	// Seed an account + repo + a pending review directly.
	acc, _ := accountSvc.Add(ctx, "acc", "https://gitlab.test", "t")
	repoSvc := appRepos.NewService(sqlite.NewRepoStore(db), sqlite.NewAccountRepo(db), sqlite.NewProviderRepo(db))
	prov, _ := providerSvc.Add(ctx, providers.AddInput{Name: "p", Kind: provider.KindOpenAICompat, Model: "m", APIKey: "k"})
	rp, _ := repoSvc.Add(ctx, appRepos.AddInput{Name: "web", URL: "https://gitlab.test/g/p", AccountID: acc.ID, ProviderID: prov.ID})

	pending := review.Review{ID: "rv1", RepoID: rp.ID, MRIID: 1, Status: review.StatusPending}
	if err := reviewStore.Create(ctx, pending); err != nil {
		t.Fatalf("seed review: %v", err)
	}

	set, _ := skills.Load("")
	svc := NewService(reviewStore, sqlite.NewRepoStore(db), accountSvc, providerSvc, engine.New(set))
	if err := svc.Publish(ctx, "rv1", Selection{All: true}); err == nil {
		t.Fatal("expected error publishing a non-done review")
	}
}
