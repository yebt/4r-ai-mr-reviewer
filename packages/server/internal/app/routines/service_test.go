package routines

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/webcloster-dev/ai-reviewer/internal/adapters/crypto"
	"github.com/webcloster-dev/ai-reviewer/internal/adapters/sqlite"
	"github.com/webcloster-dev/ai-reviewer/internal/app/accounts"
	appRepos "github.com/webcloster-dev/ai-reviewer/internal/app/repos"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/repo"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/routine"
)

func intPtr(v int) *int { return &v }

// fakeConfig configures the canned GitLab responses per endpoint. A zero status
// means 200; a non-2xx status makes that endpoint fail so the best-effort /
// fail-closed branches can be exercised.
type fakeConfig struct {
	tokenStatus   int
	tokenScopes   []string
	defaultBranch string
	projectAccess *int
	groupAccess   *int
	branchStatus  int
	branches      []map[string]any
	tagStatus     int
	tags          []map[string]any
}

// newFakeGitLab serves the four endpoints the preflight probes, routed by path.
func newFakeGitLab(t *testing.T, cfg fakeConfig) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/personal_access_tokens/self"):
			if cfg.tokenStatus != 0 && cfg.tokenStatus != http.StatusOK {
				w.WriteHeader(cfg.tokenStatus)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"active": true, "scopes": cfg.tokenScopes})
		case strings.HasSuffix(r.URL.Path, "/protected_branches"):
			if cfg.branchStatus != 0 && cfg.branchStatus != http.StatusOK {
				w.WriteHeader(cfg.branchStatus)
				return
			}
			_ = json.NewEncoder(w).Encode(cfg.branches)
		case strings.HasSuffix(r.URL.Path, "/protected_tags"):
			if cfg.tagStatus != 0 && cfg.tagStatus != http.StatusOK {
				w.WriteHeader(cfg.tagStatus)
				return
			}
			_ = json.NewEncoder(w).Encode(cfg.tags)
		case strings.Contains(r.URL.Path, "/projects/"):
			perms := map[string]any{}
			if cfg.projectAccess != nil {
				perms["project_access"] = map[string]any{"access_level": *cfg.projectAccess}
			}
			if cfg.groupAccess != nil {
				perms["group_access"] = map[string]any{"access_level": *cfg.groupAccess}
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"default_branch": cfg.defaultBranch,
				"permissions":    perms,
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// setupRoutinesTest wires a routines.Service whose single repo's account points
// at baseURL, returning the service and the repo id.
func setupRoutinesTest(t *testing.T, baseURL string) (context.Context, *Service, string) {
	t.Helper()
	ctx := context.Background()

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
	repoSvc := appRepos.NewService(sqlite.NewRepoStore(db), sqlite.NewAccountRepo(db), sqlite.NewProviderRepo(db))

	acc, err := accountSvc.Add(ctx, "acc", baseURL, "token")
	if err != nil {
		t.Fatalf("account Add: %v", err)
	}
	rp, err := repoSvc.Add(ctx, appRepos.AddInput{
		Name: "web", URL: "https://gitlab.test/group/project", AccountID: acc.ID,
	})
	if err != nil {
		t.Fatalf("repo Add: %v", err)
	}

	svc := NewService(sqlite.NewRepoStore(db), accountSvc)
	return ctx, svc, rp.ID
}

// findCheck returns the Check for a capability, failing the test if absent.
func findCheck(t *testing.T, pf routine.Preflight, cap routine.Capability) routine.Check {
	t.Helper()
	for _, c := range pf.Checks {
		if c.Capability == cap {
			return c
		}
	}
	t.Fatalf("no check for capability %q", cap)
	return routine.Check{}
}

func TestServicePreflight(t *testing.T) {
	tests := []struct {
		name  string
		cfg   fakeConfig
		check func(t *testing.T, pf routine.Preflight)
	}{
		{
			name: "happy: api scope, Maintainer, protected main → merge ok",
			cfg: fakeConfig{
				tokenScopes:   []string{"api"},
				defaultBranch: "main",
				projectAccess: intPtr(40),
				branches: []map[string]any{
					{"name": "main", "merge_access_levels": []map[string]any{{"access_level": 40}}},
				},
				tags: []map[string]any{},
			},
			check: func(t *testing.T, pf routine.Preflight) {
				if !pf.ScopesKnown {
					t.Errorf("ScopesKnown = false, want true")
				}
				if pf.AccessLevel != 40 {
					t.Errorf("AccessLevel = %d, want 40", pf.AccessLevel)
				}
				if c := findCheck(t, pf, routine.CapMergeMR); c.Status != routine.StatusOK {
					t.Errorf("merge_mr status = %q, want ok (detail %q)", c.Status, c.Detail)
				}
			},
		},
		{
			name: "group access outranks project access",
			cfg: fakeConfig{
				tokenScopes:   []string{"api"},
				defaultBranch: "main",
				projectAccess: intPtr(30),
				groupAccess:   intPtr(40),
			},
			check: func(t *testing.T, pf routine.Preflight) {
				if pf.AccessLevel != 40 {
					t.Errorf("AccessLevel = %d, want 40 (group outranks project)", pf.AccessLevel)
				}
				if pf.AccessLevelName != "Maintainer" {
					t.Errorf("AccessLevelName = %q, want Maintainer", pf.AccessLevelName)
				}
			},
		},
		{
			name: "no membership at any level → None",
			cfg: fakeConfig{
				tokenScopes:   []string{"api"},
				defaultBranch: "main",
			},
			check: func(t *testing.T, pf routine.Preflight) {
				if pf.AccessLevel != 0 {
					t.Errorf("AccessLevel = %d, want 0", pf.AccessLevel)
				}
				if pf.AccessLevelName != "None" {
					t.Errorf("AccessLevelName = %q, want None", pf.AccessLevelName)
				}
			},
		},
		{
			name: "protected-branches lookup fails → merge unknown (fail-closed)",
			cfg: fakeConfig{
				tokenScopes:   []string{"api"},
				defaultBranch: "main",
				projectAccess: intPtr(40),
				branchStatus:  http.StatusForbidden,
			},
			check: func(t *testing.T, pf routine.Preflight) {
				if c := findCheck(t, pf, routine.CapMergeMR); c.Status != routine.StatusUnknown {
					t.Errorf("merge_mr status = %q, want unknown when protection unreadable (detail %q)", c.Status, c.Detail)
				}
			},
		},
		{
			name: "token-self fails → scopes unknown",
			cfg: fakeConfig{
				tokenStatus:   http.StatusUnauthorized,
				defaultBranch: "main",
				projectAccess: intPtr(40),
			},
			check: func(t *testing.T, pf routine.Preflight) {
				if pf.ScopesKnown {
					t.Errorf("ScopesKnown = true, want false when token-self fails")
				}
				if c := findCheck(t, pf, routine.CapComment); c.Status != routine.StatusUnknown {
					t.Errorf("comment status = %q, want unknown when scopes unverified", c.Status)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newFakeGitLab(t, tt.cfg)
			ctx, svc, repoID := setupRoutinesTest(t, srv.URL)
			pf, err := svc.Preflight(ctx, repoID)
			if err != nil {
				t.Fatalf("Preflight: %v", err)
			}
			tt.check(t, pf)
		})
	}
}

func TestServicePreflightUnknownRepo(t *testing.T) {
	srv := newFakeGitLab(t, fakeConfig{tokenScopes: []string{"api"}, defaultBranch: "main"})
	ctx, svc, _ := setupRoutinesTest(t, srv.URL)
	if _, err := svc.Preflight(ctx, "does-not-exist"); !errors.Is(err, repo.ErrNotFound) {
		t.Fatalf("Preflight(unknown) error = %v, want repo.ErrNotFound", err)
	}
}
