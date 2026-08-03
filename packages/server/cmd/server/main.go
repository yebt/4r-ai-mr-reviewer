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

	"github.com/joho/godotenv"
	"github.com/webcloster-dev/ai-reviewer/internal/adapters/crypto"
	"github.com/webcloster-dev/ai-reviewer/internal/adapters/sqlite"
	"github.com/webcloster-dev/ai-reviewer/internal/app/accounts"
	"github.com/webcloster-dev/ai-reviewer/internal/app/bot"
	apphumanize "github.com/webcloster-dev/ai-reviewer/internal/app/humanize"
	"github.com/webcloster-dev/ai-reviewer/internal/app/notifications"
	"github.com/webcloster-dev/ai-reviewer/internal/app/profiles"
	"github.com/webcloster-dev/ai-reviewer/internal/app/providers"
	apprepos "github.com/webcloster-dev/ai-reviewer/internal/app/repos"
	"github.com/webcloster-dev/ai-reviewer/internal/app/reviews"
	"github.com/webcloster-dev/ai-reviewer/internal/app/routines"
	apptelegram "github.com/webcloster-dev/ai-reviewer/internal/app/telegram"
	"github.com/webcloster-dev/ai-reviewer/internal/app/vault"
	"github.com/webcloster-dev/ai-reviewer/internal/auth"
	"github.com/webcloster-dev/ai-reviewer/internal/config"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/notification"
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
	// Load a local .env into the process environment if present. Real environment
	// variables always win — godotenv.Load never overrides an already-set var — so
	// production/CI that inject real env vars are unaffected. A missing file is a
	// silent no-op; the file is read relative to the working directory (the server
	// runs from packages/server, so packages/server/.env is picked up).
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.Printf("ai-reviewer: .env not loaded: %v", err)
	}

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
	profileStore := sqlite.NewProfileStore(db)
	repoStore := sqlite.NewRepoStore(db)
	reviewStore := sqlite.NewReviewStore(db)
	humanizationStore := sqlite.NewHumanizationStore(db)
	telegramStore := sqlite.NewTelegramStore(db)
	notificationRuleStore := sqlite.NewNotificationRuleStore(db)
	jobStore := sqlite.NewJobStore(db)
	routineRunStore := sqlite.NewRoutineRunStore(db)

	// Services.
	accountSvc := accounts.NewService(accountRepo, secrets)
	providerSvc := providers.NewService(providerRepo, secrets)
	profileSvc := profiles.NewService(profileStore, providerSvc, nil)
	repoSvc := apprepos.NewService(repoStore, accountRepo, providerRepo)

	ruleSet, err := skills.Load(cfg.SkillsDir)
	if err != nil {
		return err
	}
	reviewSvc := reviews.NewService(reviewStore, repoStore, accountSvc, providerSvc, engine.NewMultiPass(ruleSet), cfg.ReasoningBudget)
	humanizeSvc := apphumanize.NewService(reviewStore, profileStore, humanizationStore, providerSvc, nil)
	telegramSvc := apptelegram.NewService(telegramStore, secrets)
	notificationsSvc := notifications.NewService(notificationRuleStore, telegramSvc)
	// The release routine notifies through the shared notifications fan-out via a
	// thin adapter, so app/routines stays decoupled from app/notifications (the
	// composition wiring lives here at the root).
	routinesSvc := routines.NewService(repoStore, accountSvc, routineRunStore, cfg.MergeWaitTimeout, releaseNotifier{notificationsSvc}, nil)
	botSvc := bot.NewService(bot.NewAPIClient(), telegramSvc, reviewSvc, repoSvc)

	runner := jobs.NewRunner(jobStore, reviewSvc.Handle, jobs.WithConcurrency(cfg.ReviewConcurrency))
	reviewSvc.AttachRunner(runner)
	reviewSvc.AttachNotifier(notificationsSvc)
	go runner.Start(ctx)
	go routinesSvc.Start(ctx)

	// Re-trigger any style-guide distillations left pending by a prior crash.
	go profileSvc.RecoverPending(context.Background())

	authMgr := auth.NewManager(cfg.AuthPassword, time.Duration(cfg.AuthSessionHours)*time.Hour)
	if !authMgr.Enabled() {
		log.Print("ai-reviewer: WARNING — API authentication is DISABLED; set AIR_AUTH_PASSWORD to require a password")
	}

	api := httpapi.NewServer(accountSvc, providerSvc, profileSvc, repoSvc, reviewSvc, routinesSvc, humanizeSvc, telegramSvc, notificationsSvc, ruleSet, botSvc, cfg.TelegramWebhookSecret, authMgr, cfg.TrustProxyHeaders)
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

// releaseNotifier adapts the notifications fan-out service to the
// routines.ReleaseNotifier interface, delivering a finished-release summary as
// the release.finished event. It keeps app/routines from importing
// app/notifications: the composition root owns this wiring.
type releaseNotifier struct {
	svc *notifications.Service
}

// Notify fans the release summary out to the rules subscribed to
// release.finished for repoID (a repo's own rules override the global ones).
func (n releaseNotifier) Notify(ctx context.Context, repoID, text string) error {
	return n.svc.Notify(ctx, notification.EventReleaseFinished, repoID, text)
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
