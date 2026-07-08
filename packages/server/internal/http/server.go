// Package httpapi exposes the backend over a JSON HTTP API. It is the contract
// every client (TUI, SPA, Telegram bot) consumes.
package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/webcloster-dev/ai-reviewer/internal/app/accounts"
	"github.com/webcloster-dev/ai-reviewer/internal/app/providers"
	"github.com/webcloster-dev/ai-reviewer/internal/app/repos"
	"github.com/webcloster-dev/ai-reviewer/internal/app/reviews"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/account"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/job"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/provider"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/repo"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/review"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/secret"
)

// Server holds the application services the handlers call.
type Server struct {
	accounts  *accounts.Service
	providers *providers.Service
	repos     *repos.Service
	reviews   *reviews.Service
}

// NewServer wires a Server.
func NewServer(a *accounts.Service, p *providers.Service, r *repos.Service, rv *reviews.Service) *Server {
	return &Server{accounts: a, providers: p, repos: r, reviews: rv}
}

// Routes returns the HTTP handler with every endpoint registered.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	mux.HandleFunc("POST /accounts", s.createAccount)
	mux.HandleFunc("GET /accounts", s.listAccounts)
	mux.HandleFunc("DELETE /accounts/{id}", s.deleteAccount)

	mux.HandleFunc("POST /providers", s.createProvider)
	mux.HandleFunc("GET /providers", s.listProviders)
	mux.HandleFunc("PATCH /providers/{id}", s.updateProvider)
	mux.HandleFunc("POST /providers/{id}/default", s.setDefaultProvider)
	mux.HandleFunc("DELETE /providers/{id}", s.deleteProvider)

	mux.HandleFunc("POST /repos", s.createRepo)
	mux.HandleFunc("GET /repos", s.listRepos)
	mux.HandleFunc("PATCH /repos/{id}/assign", s.assignRepo)
	mux.HandleFunc("DELETE /repos/{id}", s.deleteRepo)
	mux.HandleFunc("GET /repos/{id}/merge-requests", s.listMergeRequests)
	mux.HandleFunc("GET /repos/{id}/reviews", s.listReviews)

	mux.HandleFunc("POST /reviews", s.createReview)
	mux.HandleFunc("GET /reviews/{id}", s.getReview)
	mux.HandleFunc("POST /reviews/{id}/retry", s.retryReview)
	mux.HandleFunc("POST /reviews/{id}/publish", s.publishReview)

	return mux
}

// --- helpers ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeErr maps not-found sentinels to 404 and everything else to fallback.
func writeErr(w http.ResponseWriter, err error, fallback int) {
	status := fallback
	if isNotFound(err) {
		status = http.StatusNotFound
	}
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func isNotFound(err error) bool {
	return errors.Is(err, account.ErrNotFound) ||
		errors.Is(err, provider.ErrNotFound) ||
		errors.Is(err, repo.ErrNotFound) ||
		errors.Is(err, review.ErrNotFound) ||
		errors.Is(err, job.ErrNotFound) ||
		errors.Is(err, secret.ErrNotFound)
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
