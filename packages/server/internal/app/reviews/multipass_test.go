package reviews

import (
	"context"
	"io"
	"log"
	"path/filepath"
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

// TestReviewMultiPassEndToEnd runs the orchestrator with the multi-pass strategy
// and asserts one focused call per 4R dimension aggregated into the review.
func TestReviewMultiPassEndToEnd(t *testing.T) {
	ctx := context.Background()

	gl := gitlabStub(t)
	defer gl.Close()
	// Each dimension pass gets this findings-only response.
	aiSrv := aiStub(t, `{"findings":[{"severity":"low","file":"a.go","line":1,"issue":"x"}]}`)
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

	rv, _ := svc.Create(ctx, rp.ID, 7, review.ModeFast, "", "")
	runner.Drain(ctx)

	got, err := reviewStore.Get(ctx, rv.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Status != review.StatusDone {
		t.Fatalf("status = %s (err=%q), want done", got.Status, got.Error)
	}
	if len(got.Findings) != 4 {
		t.Fatalf("findings = %d, want 4 (one per dimension)", len(got.Findings))
	}
	seen := map[review.Dimension]bool{}
	for _, f := range got.Findings {
		seen[f.Dimension] = true
	}
	for _, d := range []review.Dimension{review.Risk, review.Readability, review.Reliability, review.Resilience} {
		if !seen[d] {
			t.Fatalf("missing dimension %s in findings: %+v", d, got.Findings)
		}
	}
	// On completion the phase is cleared.
	if got.Phase != "" {
		t.Fatalf("phase = %q, want empty on done", got.Phase)
	}
}
