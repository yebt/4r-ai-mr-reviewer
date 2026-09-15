package mergerequests

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/webcloster-dev/ai-reviewer/internal/adapters/crypto"
	"github.com/webcloster-dev/ai-reviewer/internal/adapters/sqlite"
	"github.com/webcloster-dev/ai-reviewer/internal/app/accounts"
	"github.com/webcloster-dev/ai-reviewer/internal/app/providers"
	appRepos "github.com/webcloster-dev/ai-reviewer/internal/app/repos"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/provider"
)

// TestMain points the AI client at its buffered (non-streaming) path for this
// package's tests, whose stubs serve a single JSON completion rather than SSE.
// The streaming transport itself is covered in internal/adapters/ai.
func TestMain(m *testing.M) {
	os.Setenv("AIR_AI_STREAM", "false")
	os.Exit(m.Run())
}

// compareStub serves a canned GitLab branch comparison so Generate has diffs to
// describe. It only answers the /repository/compare endpoint.
func compareStub(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/repository/compare") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"commits": []map[string]any{{"id": "a1", "title": "feat: add", "message": "feat: add"}},
			"diffs": []map[string]any{
				{"old_path": "a.go", "new_path": "a.go", "diff": "@@ -1 +1 @@\n-old\n+new"},
			},
		})
	}))
}

// modelCapturingAIStub serves a canned OpenAI-compatible drafting completion and
// records the model string it received, so a test can assert which model the
// service actually sent to the provider.
func modelCapturingAIStub(t *testing.T, gotHits *int, sink *string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/chat/completions") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		var body struct {
			Model string `json:"model"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		*sink = body.Model
		*gotHits++
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"model":   body.Model,
			"choices": []map[string]any{{"message": map[string]any{"content": "My MR title\n\nA short description."}}},
			"usage":   map[string]any{"prompt_tokens": 1, "completion_tokens": 1},
		})
	}))
}

// mrTestEnv wires a mergerequests.Service against sqlite stores and a GitLab
// compare stub, mirroring the reviews service tests.
type mrTestEnv struct {
	svc  *Service
	repo string // repo ID whose ProviderID points at the "repo" provider
}

func newMRTestEnv(t *testing.T, repoAISrv *httptest.Server, repoProviderModel string) (mrTestEnv, *providers.Service, context.Context) {
	t.Helper()
	ctx := context.Background()

	gl := compareStub(t)
	t.Cleanup(gl.Close)

	db, err := sqlite.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	salt, _ := crypto.NewSalt()
	key, _ := crypto.DeriveKey("pw", salt)
	cipher, _ := crypto.NewCipher(key)
	secrets := sqlite.NewSecretStore(db, cipher)
	accountSvc := accounts.NewService(sqlite.NewAccountRepo(db), secrets)
	providerSvc := providers.NewService(sqlite.NewProviderRepo(db), secrets)
	repoSvc := appRepos.NewService(sqlite.NewRepoStore(db), sqlite.NewAccountRepo(db), sqlite.NewProviderRepo(db))

	acc, _ := accountSvc.Add(ctx, "acc", gl.URL, "token")
	prov, _ := providerSvc.Add(ctx, providers.AddInput{Name: "repo-prov", Kind: provider.KindOpenAICompat, BaseURL: repoAISrv.URL, Model: repoProviderModel, APIKey: "k"})
	rp, _ := repoSvc.Add(ctx, appRepos.AddInput{Name: "web", URL: "https://gitlab.test/group/project", AccountID: acc.ID, ProviderID: prov.ID})

	svc := NewService(sqlite.NewRepoStore(db), accountSvc, providerSvc, sqlite.NewProfileStore(db))
	return mrTestEnv{svc: svc, repo: rp.ID}, providerSvc, ctx
}

// TestGenerateProviderAndModelOverride asserts that an explicit providerId/model
// passed to Generate wins over the repo's resolved provider/model: the drafting
// completion must hit the override provider with the override model, and the
// repo's own provider must not be called at all.
func TestGenerateProviderAndModelOverride(t *testing.T) {
	var repoHits, overrideHits int
	var repoModel, overrideModel string
	repoAISrv := modelCapturingAIStub(t, &repoHits, &repoModel)
	t.Cleanup(repoAISrv.Close)
	overrideAISrv := modelCapturingAIStub(t, &overrideHits, &overrideModel)
	t.Cleanup(overrideAISrv.Close)

	env, providerSvc, ctx := newMRTestEnv(t, repoAISrv, "repo-model")

	// A second provider the caller overrides to; it is NOT the repo's provider.
	overrideProv, err := providerSvc.Add(ctx, providers.AddInput{Name: "override-prov", Kind: provider.KindOpenAICompat, BaseURL: overrideAISrv.URL, Model: "override-prov-model", APIKey: "k"})
	if err != nil {
		t.Fatalf("Add override provider: %v", err)
	}

	gen, err := env.svc.Generate(ctx, env.repo, "feat", "main", "", overrideProv.ID, "override-model")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if gen.Title == "" {
		t.Fatalf("Generate returned empty title")
	}
	if overrideHits != 1 {
		t.Fatalf("override provider hits = %d, want 1 (override provider must be used)", overrideHits)
	}
	if repoHits != 0 {
		t.Fatalf("repo provider hits = %d, want 0 (override provider must win)", repoHits)
	}
	if overrideModel != "override-model" {
		t.Fatalf("model sent = %q, want override-model (request model override must win)", overrideModel)
	}
}

// TestGenerateDefaultResolvesRepoProvider proves the pre-existing behaviour is
// preserved: with empty overrides, Generate resolves the repo's provider and
// falls through to the provider's default model (repo has no model set).
func TestGenerateDefaultResolvesRepoProvider(t *testing.T) {
	var repoHits int
	var repoModel string
	repoAISrv := modelCapturingAIStub(t, &repoHits, &repoModel)
	t.Cleanup(repoAISrv.Close)

	env, _, ctx := newMRTestEnv(t, repoAISrv, "repo-provider-model")

	if _, err := env.svc.Generate(ctx, env.repo, "feat", "main", "", "", ""); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if repoHits != 1 {
		t.Fatalf("repo provider hits = %d, want 1 (empty overrides must resolve the repo provider)", repoHits)
	}
	if repoModel != "repo-provider-model" {
		t.Fatalf("model sent = %q, want repo-provider-model (repo->provider resolution must be preserved)", repoModel)
	}
}
