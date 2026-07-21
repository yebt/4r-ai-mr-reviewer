// Package httpapi exposes the backend over a JSON HTTP API. It is the contract
// every client (TUI, SPA, Telegram bot) consumes.
package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/webcloster-dev/ai-reviewer/internal/app/accounts"
	apphumanize "github.com/webcloster-dev/ai-reviewer/internal/app/humanize"
	"github.com/webcloster-dev/ai-reviewer/internal/app/profiles"
	"github.com/webcloster-dev/ai-reviewer/internal/app/providers"
	"github.com/webcloster-dev/ai-reviewer/internal/app/repos"
	"github.com/webcloster-dev/ai-reviewer/internal/app/reviews"
	apptelegram "github.com/webcloster-dev/ai-reviewer/internal/app/telegram"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/account"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/job"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/profile"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/provider"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/repo"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/review"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/secret"
	tgdomain "github.com/webcloster-dev/ai-reviewer/internal/domain/telegram"
	"github.com/webcloster-dev/ai-reviewer/internal/review/skills"
)

// Server holds the application services the handlers call.
type Server struct {
	accounts  *accounts.Service
	providers *providers.Service
	profiles  *profiles.Service
	repos     *repos.Service
	reviews   *reviews.Service
	humanize  *apphumanize.Service
	telegram  *apptelegram.Service
	skills    skills.Set
}

// NewServer wires a Server.
func NewServer(a *accounts.Service, p *providers.Service, pr *profiles.Service, r *repos.Service, rv *reviews.Service, hz *apphumanize.Service, tg *apptelegram.Service, sk skills.Set) *Server {
	return &Server{accounts: a, providers: p, profiles: pr, repos: r, reviews: rv, humanize: hz, telegram: tg, skills: sk}
}

// Routes returns the HTTP handler with every endpoint registered.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	mux.HandleFunc("GET /skills", s.getSkills)

	mux.HandleFunc("POST /accounts", s.createAccount)
	mux.HandleFunc("GET /accounts", s.listAccounts)
	mux.HandleFunc("DELETE /accounts/{id}", s.deleteAccount)

	mux.HandleFunc("POST /providers", s.createProvider)
	mux.HandleFunc("GET /providers", s.listProviders)
	mux.HandleFunc("PATCH /providers/{id}", s.updateProvider)
	mux.HandleFunc("POST /providers/{id}/default", s.setDefaultProvider)
	mux.HandleFunc("DELETE /providers/{id}", s.deleteProvider)

	mux.HandleFunc("POST /telegram", s.createTelegram)
	mux.HandleFunc("POST /telegram/resolve", s.resolveTelegram)
	mux.HandleFunc("GET /telegram", s.listTelegram)
	mux.HandleFunc("DELETE /telegram/{id}", s.deleteTelegram)
	mux.HandleFunc("POST /telegram/{id}/default", s.setDefaultTelegram)
	mux.HandleFunc("POST /telegram/{id}/test", s.testTelegram)

	mux.HandleFunc("POST /profiles", s.createProfile)
	mux.HandleFunc("GET /profiles", s.listProfiles)
	mux.HandleFunc("GET /profiles/{id}", s.getProfile)
	mux.HandleFunc("PATCH /profiles/{id}", s.updateProfile)
	mux.HandleFunc("DELETE /profiles/{id}", s.deleteProfile)
	mux.HandleFunc("POST /profiles/{id}/redistill", s.redistillProfile)

	mux.HandleFunc("POST /repos", s.createRepo)
	mux.HandleFunc("GET /repos", s.listRepos)
	mux.HandleFunc("PATCH /repos/{id}/assign", s.assignRepo)
	mux.HandleFunc("DELETE /repos/{id}", s.deleteRepo)
	mux.HandleFunc("GET /repos/{id}/merge-requests", s.listMergeRequests)
	mux.HandleFunc("GET /repos/{id}/reviews", s.listReviews)

	mux.HandleFunc("POST /reviews", s.createReview)
	mux.HandleFunc("GET /reviews/{id}", s.getReview)
	mux.HandleFunc("DELETE /reviews/{id}", s.deleteReview)
	mux.HandleFunc("POST /reviews/{id}/retry", s.retryReview)
	mux.HandleFunc("POST /reviews/{id}/publish", s.publishReview)
	mux.HandleFunc("POST /reviews/{id}/cancel", s.cancelReview)
	mux.HandleFunc("POST /reviews/{id}/archive", s.archiveReview)
	mux.HandleFunc("POST /reviews/{id}/unarchive", s.unarchiveReview)
	mux.HandleFunc("POST /reviews/{id}/humanize", s.humanizeReview)
	mux.HandleFunc("GET /reviews/{id}/humanizations", s.getHumanizations)

	return mux
}

// --- helpers ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeErr maps not-found sentinels to 404 and everything else to fallback.
//
// Client-error statuses (4xx) return the error message, which for this codebase
// is app-authored validation text safe to show. Server errors (5xx) wrap internal
// or upstream detail — DB operation names, provider/GitLab response bodies,
// filesystem paths — so those are logged server-side and the client gets only a
// generic status message.
func writeErr(w http.ResponseWriter, err error, fallback int) {
	status := fallback
	if isNotFound(err) {
		status = http.StatusNotFound
	}
	msg := err.Error()
	if status >= 500 {
		log.Printf("http %d: %v", status, err)
		msg = http.StatusText(status)
	}
	writeJSON(w, status, map[string]string{"error": msg})
}

func isNotFound(err error) bool {
	return errors.Is(err, account.ErrNotFound) ||
		errors.Is(err, provider.ErrNotFound) ||
		errors.Is(err, profile.ErrNotFound) ||
		errors.Is(err, repo.ErrNotFound) ||
		errors.Is(err, review.ErrNotFound) ||
		errors.Is(err, job.ErrNotFound) ||
		errors.Is(err, secret.ErrNotFound) ||
		errors.Is(err, tgdomain.ErrNotFound)
}

// decode reads a JSON body into dst, rejecting empty or malformed input.
func decode(r *http.Request, dst any) error {
	if r.Body == nil {
		return errors.New("empty request body")
	}
	err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(dst)
	if errors.Is(err, io.EOF) {
		return errors.New("empty request body")
	}
	return err
}
