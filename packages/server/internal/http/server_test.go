package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/webcloster-dev/ai-reviewer/internal/adapters/crypto"
	"github.com/webcloster-dev/ai-reviewer/internal/adapters/sqlite"
	"github.com/webcloster-dev/ai-reviewer/internal/app/accounts"
	"github.com/webcloster-dev/ai-reviewer/internal/app/profiles"
	"github.com/webcloster-dev/ai-reviewer/internal/app/providers"
	apprepos "github.com/webcloster-dev/ai-reviewer/internal/app/repos"
	"github.com/webcloster-dev/ai-reviewer/internal/app/reviews"
	"github.com/webcloster-dev/ai-reviewer/internal/jobs"
	"github.com/webcloster-dev/ai-reviewer/internal/review/engine"
	"github.com/webcloster-dev/ai-reviewer/internal/review/skills"
)

func newTestServer(t *testing.T) *httptest.Server {
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

	accountSvc := accounts.NewService(sqlite.NewAccountRepo(db), secrets)
	providerSvc := providers.NewService(sqlite.NewProviderRepo(db), secrets)
	profileSvc := profiles.NewService(sqlite.NewProfileStore(db))
	repoSvc := apprepos.NewService(sqlite.NewRepoStore(db), sqlite.NewAccountRepo(db), sqlite.NewProviderRepo(db))
	set, _ := skills.Load("")
	reviewSvc := reviews.NewService(sqlite.NewReviewStore(db), sqlite.NewRepoStore(db), accountSvc, providerSvc, engine.New(set))
	runner := jobs.NewRunner(sqlite.NewJobStore(db), reviewSvc.Handle, jobs.WithLogger(log.New(io.Discard, "", 0)))
	reviewSvc.AttachRunner(runner)

	srv := httptest.NewServer(NewServer(accountSvc, providerSvc, profileSvc, repoSvc, reviewSvc, set).Routes())
	t.Cleanup(srv.Close)
	return srv
}

func postJSON(t *testing.T, url string, body any) *http.Response {
	t.Helper()
	b, _ := json.Marshal(body)
	resp, err := http.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	return resp
}

func decodeBody(t *testing.T, resp *http.Response, dst any) {
	t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		t.Fatalf("decode: %v", err)
	}
}

func TestHealth(t *testing.T) {
	srv := newTestServer(t)
	resp, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

func TestAccountLifecycleOverHTTP(t *testing.T) {
	srv := newTestServer(t)

	resp := postJSON(t, srv.URL+"/accounts", map[string]any{"name": "acc", "baseUrl": "https://gitlab.com", "token": "glpat"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create account status = %d, want 201", resp.StatusCode)
	}
	var created struct{ ID string }
	decodeBody(t, resp, &created)
	if created.ID == "" {
		t.Fatal("created account has no id")
	}

	listResp, _ := http.Get(srv.URL + "/accounts")
	var list []map[string]any
	decodeBody(t, listResp, &list)
	if len(list) != 1 {
		t.Fatalf("account list len = %d, want 1", len(list))
	}

	// The token must never be exposed in the API response.
	if _, leaked := list[0]["token"]; leaked {
		t.Fatal("account response leaked the token")
	}
}

func TestCreateReviewOverHTTP(t *testing.T) {
	srv := newTestServer(t)

	acctResp := postJSON(t, srv.URL+"/accounts", map[string]any{"name": "a", "baseUrl": "https://gitlab.com", "token": "t"})
	var acct struct{ ID string }
	decodeBody(t, acctResp, &acct)

	provResp := postJSON(t, srv.URL+"/providers", map[string]any{"name": "p", "kind": "openai-compat", "model": "m", "apiKey": "k"})
	var prov struct{ ID string }
	decodeBody(t, provResp, &prov)

	repoResp := postJSON(t, srv.URL+"/repos", map[string]any{"name": "web", "url": "https://gitlab.com/g/p", "accountId": acct.ID, "providerId": prov.ID})
	if repoResp.StatusCode != http.StatusCreated {
		t.Fatalf("create repo status = %d", repoResp.StatusCode)
	}
	var repoObj struct{ ID string }
	decodeBody(t, repoResp, &repoObj)

	revResp := postJSON(t, srv.URL+"/reviews", map[string]any{"repoId": repoObj.ID, "mrIid": 7, "mode": "fast"})
	if revResp.StatusCode != http.StatusCreated {
		t.Fatalf("create review status = %d, want 201", revResp.StatusCode)
	}
	var rev struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	decodeBody(t, revResp, &rev)
	if rev.Status != "pending" {
		t.Fatalf("review status = %q, want pending", rev.Status)
	}

	// It should show up under the repo's reviews.
	listResp, _ := http.Get(srv.URL + "/repos/" + repoObj.ID + "/reviews")
	var reviewsList []map[string]any
	decodeBody(t, listResp, &reviewsList)
	if len(reviewsList) != 1 {
		t.Fatalf("repo reviews len = %d, want 1", len(reviewsList))
	}
}

func TestCreateRepoRejectsUnknownAccount(t *testing.T) {
	srv := newTestServer(t)
	resp := postJSON(t, srv.URL+"/repos", map[string]any{"name": "web", "url": "https://gitlab.com/g/p", "accountId": "nope"})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 for unknown account", resp.StatusCode)
	}
}
