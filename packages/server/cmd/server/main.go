// Command server is the ai-reviewer backend: the single process that owns all
// state and is consumed by every client (TUI, SPA, Telegram bot).
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/webcloster-dev/ai-reviewer/internal/adapters/crypto"
	"github.com/webcloster-dev/ai-reviewer/internal/adapters/sqlite"
	"github.com/webcloster-dev/ai-reviewer/internal/app/accounts"
	"github.com/webcloster-dev/ai-reviewer/internal/app/providers"
	apprepos "github.com/webcloster-dev/ai-reviewer/internal/app/repos"
	"github.com/webcloster-dev/ai-reviewer/internal/app/reviews"
	"github.com/webcloster-dev/ai-reviewer/internal/app/vault"
	"github.com/webcloster-dev/ai-reviewer/internal/config"
	httpapi "github.com/webcloster-dev/ai-reviewer/internal/http"
	"github.com/webcloster-dev/ai-reviewer/internal/jobs"
	"github.com/webcloster-dev/ai-reviewer/internal/review/engine"
	"github.com/webcloster-dev/ai-reviewer/internal/review/skills"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("ai-reviewer: %v", err)
	}
}

func run() error {
	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := sqlite.Open(cfg.DBPath)
	if err != nil {
		return err
	}
	defer db.Close()

	cipher, err := unlockVault(ctx, sqlite.NewMetaStore(db), cfg)
	if err != nil {
		return err
	}

	// Stores.
	secrets := sqlite.NewSecretStore(db, cipher)
	accountRepo := sqlite.NewAccountRepo(db)
	providerRepo := sqlite.NewProviderRepo(db)
	repoStore := sqlite.NewRepoStore(db)
	reviewStore := sqlite.NewReviewStore(db)
	jobStore := sqlite.NewJobStore(db)

	// Services.
	accountSvc := accounts.NewService(accountRepo, secrets)
	providerSvc := providers.NewService(providerRepo, secrets)
	repoSvc := apprepos.NewService(repoStore, accountRepo, providerRepo)

	ruleSet, err := skills.Load(cfg.SkillsDir)
	if err != nil {
		return err
	}
	reviewSvc := reviews.NewService(reviewStore, repoStore, accountSvc, providerSvc, engine.NewMultiPass(ruleSet))

	runner := jobs.NewRunner(jobStore, reviewSvc.Handle)
	reviewSvc.AttachRunner(runner)
	go runner.Start(ctx)

	api := httpapi.NewServer(accountSvc, providerSvc, repoSvc, reviewSvc, ruleSet)
	srv := &http.Server{Addr: cfg.HTTPAddr, Handler: api.Routes()}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	log.Printf("ai-reviewer: listening on %s", cfg.HTTPAddr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	log.Print("ai-reviewer: shut down cleanly")
	return nil
}

// unlockVault initializes the secret vault on first run, or unlocks it, and
// returns the cipher used to protect stored secrets.
func unlockVault(ctx context.Context, meta *sqlite.MetaStore, cfg config.Config) (*crypto.Cipher, error) {
	v := vault.New(meta, cfg.KeyfilePath)
	status, err := v.Status(ctx)
	if err != nil {
		return nil, err
	}
	if !status.Initialized {
		cipher, err := v.Initialize(ctx, cfg.Password)
		if err != nil {
			return nil, err
		}
		if cfg.Password == "" {
			log.Printf("ai-reviewer: vault initialized (key-file mode at %s)", cfg.KeyfilePath)
		} else {
			log.Print("ai-reviewer: vault initialized (password mode)")
		}
		return cipher, nil
	}
	return v.Unlock(ctx, cfg.Password)
}
