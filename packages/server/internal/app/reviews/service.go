// Package reviews orchestrates a review end to end: it creates the review,
// enqueues it, and — as the job handler — resolves the account, provider and
// model, gathers context, runs the engine, and persists the result.
package reviews

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/webcloster-dev/ai-reviewer/internal/adapters/ai"
	"github.com/webcloster-dev/ai-reviewer/internal/adapters/gitlab"
	"github.com/webcloster-dev/ai-reviewer/internal/app/accounts"
	"github.com/webcloster-dev/ai-reviewer/internal/app/providers"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/provider"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/repo"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/review"
	"github.com/webcloster-dev/ai-reviewer/internal/id"
	"github.com/webcloster-dev/ai-reviewer/internal/jobs"
	reviewctx "github.com/webcloster-dev/ai-reviewer/internal/review/context"
	"github.com/webcloster-dev/ai-reviewer/internal/review/engine"
)

// ErrNotCancelable is returned when a review is already in a terminal state and
// therefore cannot be cancelled. The HTTP layer maps it to 409 Conflict.
var ErrNotCancelable = errors.New("reviews: review is not cancelable")

// Service orchestrates reviews.
type Service struct {
	reviews   review.Repository
	repos     repo.Repository
	accounts  *accounts.Service
	providers *providers.Service
	strategy  engine.Strategy
	runner    *jobs.Runner

	// Cancellation state. Jobs run sequentially on the runner's shared context,
	// so each running review gets its own cancelable context registered here;
	// an HTTP-triggered Cancel (on another goroutine) fires it by id.
	mu        sync.Mutex
	cancels   map[string]context.CancelFunc // reviewID -> cancel of the running job's ctx
	requested map[string]bool               // reviewID -> cancel requested (covers pending / pre-registration)
}

// NewService wires the orchestrator. Call AttachRunner before creating reviews.
func NewService(
	reviews review.Repository,
	repos repo.Repository,
	accounts *accounts.Service,
	providers *providers.Service,
	strategy engine.Strategy,
) *Service {
	return &Service{
		reviews:   reviews,
		repos:     repos,
		accounts:  accounts,
		providers: providers,
		strategy:  strategy,
		cancels:   make(map[string]context.CancelFunc),
		requested: make(map[string]bool),
	}
}

// markRequested records a cancellation request and, if the job is already
// running, fires its context immediately.
func (s *Service) markRequested(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requested[id] = true
	if cancel := s.cancels[id]; cancel != nil {
		cancel()
	}
}

// isRequested reports whether a cancellation was requested for id.
func (s *Service) isRequested(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.requested[id]
}

// registerCancel stores the running job's cancel func. If a cancel was already
// requested before registration (the race where Cancel arrives first), it fires
// immediately.
func (s *Service) registerCancel(id string, cancel context.CancelFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cancels[id] = cancel
	if s.requested[id] {
		cancel()
	}
}

// clearCancel drops all cancellation state for a finished job.
func (s *Service) clearCancel(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.cancels, id)
	delete(s.requested, id)
}

// Cancel requests cooperative cancellation of a pending or running review. It
// never writes the terminal status itself — Handle is the sole terminal-status
// writer — so callers observe the flip to "cancelled" via polling. A review in
// a terminal state returns ErrNotCancelable.
func (s *Service) Cancel(ctx context.Context, reviewID string) error {
	rv, err := s.reviews.Get(ctx, reviewID)
	if err != nil {
		return err
	}
	if rv.Status.Terminal() {
		return ErrNotCancelable
	}
	s.markRequested(reviewID)
	return nil
}

// AttachRunner sets the job runner used to enqueue reviews. This breaks the
// construction cycle (the runner's handler is this service's Handle).
func (s *Service) AttachRunner(r *jobs.Runner) { s.runner = r }

// Create records a pending review and enqueues it. An empty mode defaults to fast.
func (s *Service) Create(ctx context.Context, repoID string, mrIID int, mode review.ContextMode) (review.Review, error) {
	if _, err := s.repos.Get(ctx, repoID); err != nil {
		return review.Review{}, err
	}
	if mode == "" {
		mode = review.ModeFast
	}
	rv := review.Review{
		ID:          id.New(),
		RepoID:      repoID,
		MRIID:       mrIID,
		ContextMode: mode,
		Status:      review.StatusPending,
	}
	if err := s.reviews.Create(ctx, rv); err != nil {
		return review.Review{}, err
	}
	if _, err := s.runner.Enqueue(ctx, rv.ID); err != nil {
		return review.Review{}, err
	}
	return rv, nil
}

// List returns a repo's active (non-archived) reviews (without findings),
// newest first.
func (s *Service) List(ctx context.Context, repoID string) ([]review.Review, error) {
	return s.reviews.ListByRepo(ctx, repoID)
}

// ListArchived returns a repo's archived reviews (without findings), newest first.
func (s *Service) ListArchived(ctx context.Context, repoID string) ([]review.Review, error) {
	return s.reviews.ListArchivedByRepo(ctx, repoID)
}

// Get returns a review with its findings.
func (s *Service) Get(ctx context.Context, reviewID string) (review.Review, error) {
	return s.reviews.Get(ctx, reviewID)
}

// Delete hard-removes a review and its findings.
func (s *Service) Delete(ctx context.Context, reviewID string) error {
	return s.reviews.Delete(ctx, reviewID)
}

// Archive soft-hides a review from the active list, keeping its history.
func (s *Service) Archive(ctx context.Context, reviewID string) error {
	return s.reviews.SetArchived(ctx, reviewID, true)
}

// Unarchive restores an archived review to the active list.
func (s *Service) Unarchive(ctx context.Context, reviewID string) error {
	return s.reviews.SetArchived(ctx, reviewID, false)
}

// ListOpenMergeRequests lists the open MRs of a repo's GitLab project.
func (s *Service) ListOpenMergeRequests(ctx context.Context, repoID string) ([]gitlab.MergeRequest, error) {
	rp, err := s.repos.Get(ctx, repoID)
	if err != nil {
		return nil, err
	}
	gl, projectID, _, err := s.gitlabFor(ctx, rp)
	if err != nil {
		return nil, err
	}
	return gl.ListOpenMergeRequests(ctx, projectID)
}

// Retry clones an existing review's configuration into a fresh pending review
// and enqueues it. The original (errored) review is kept for history.
func (s *Service) Retry(ctx context.Context, reviewID string) (review.Review, error) {
	old, err := s.reviews.Get(ctx, reviewID)
	if err != nil {
		return review.Review{}, err
	}
	return s.Create(ctx, old.RepoID, old.MRIID, old.ContextMode)
}

// Handle is the job handler: it executes one review by id and persists the
// outcome. It returns an error only when the job itself should be marked failed.
func (s *Service) Handle(ctx context.Context, reviewID string) error {
	// Fast path: cancel requested before we even start (e.g. a pending job).
	if s.isRequested(reviewID) {
		_ = s.reviews.SetStatus(ctx, reviewID, review.StatusCancelled, "")
		_ = s.reviews.SetPhase(ctx, reviewID, "")
		s.clearCancel(reviewID)
		return nil // job done — do NOT fail/retry
	}

	// Clear cancellation bookkeeping on every exit below, including the early
	// error/terminal returns before registerCancel runs (a racing Cancel may have
	// set requested[id]); otherwise those entries would leak for the process life.
	defer s.clearCancel(reviewID)

	rv, err := s.reviews.Get(ctx, reviewID)
	if err != nil {
		return err
	}
	// Crash-recovery guard: RequeueRunning may re-queue a job whose review
	// already reached a terminal state (a crash between persisting the review and
	// completing the job). Re-running would re-charge the LLM and wipe published
	// state, so treat the job as already done.
	if rv.Status.Terminal() {
		return nil
	}
	if err := s.reviews.SetStatus(ctx, reviewID, review.StatusRunning, ""); err != nil {
		return err
	}

	// Per-review cancelable context so an HTTP Cancel(id) can abort this job
	// without touching the runner's shared, long-lived ctx.
	cctx, cancel := context.WithCancel(ctx)
	s.registerCancel(reviewID, cancel)
	defer cancel()

	result, err := s.execute(cctx, rv)
	if err != nil {
		// Distinguish a cancellation from a genuine failure.
		if cctx.Err() == context.Canceled || s.isRequested(reviewID) {
			// Use the PARENT ctx for these writes — cctx is already cancelled.
			_ = s.reviews.SetStatus(ctx, reviewID, review.StatusCancelled, "")
			_ = s.reviews.SetPhase(ctx, reviewID, "")
			return nil // job completes; no retry
		}
		// Record the failure on the review; the job also fails so it can retry.
		_ = s.reviews.SetStatus(ctx, reviewID, review.StatusError, err.Error())
		return err
	}

	result.ID = rv.ID
	result.RepoID = rv.RepoID
	result.MRIID = rv.MRIID
	if err := s.reviews.Save(ctx, result); err != nil {
		// The review succeeded but persisting the result failed; don't leave it
		// stuck in "running" forever (RequeueRunning wouldn't recover it once the
		// job is terminal). Mark it errored so it surfaces and can be retried.
		_ = s.reviews.SetStatus(ctx, reviewID, review.StatusError, err.Error())
		return err
	}
	return nil
}

func (s *Service) execute(ctx context.Context, rv review.Review) (review.Review, error) {
	rp, err := s.repos.Get(ctx, rv.RepoID)
	if err != nil {
		return review.Review{}, err
	}

	prov, err := s.resolveProvider(ctx, rp.ProviderID)
	if err != nil {
		return review.Review{}, err
	}
	apiKey, err := s.providers.APIKey(ctx, prov.ID)
	if err != nil {
		return review.Review{}, err
	}
	model := rp.Model
	if model == "" {
		model = prov.Model
	}
	if model == "" {
		return review.Review{}, fmt.Errorf("reviews: no model set on repo or provider %q", prov.Name)
	}
	aiClient, err := ai.New(prov, apiKey)
	if err != nil {
		return review.Review{}, err
	}

	gl, projectID, token, err := s.gitlabFor(ctx, rp)
	if err != nil {
		return review.Review{}, err
	}

	strat := s.contextStrategy(rv.ContextMode, gl, token)
	in, cleanup, err := strat.Build(ctx, reviewctx.Target{RepoURL: rp.URL, ProjectID: projectID, MRIID: rv.MRIID})
	defer cleanup()
	if err != nil {
		return review.Review{}, err
	}
	in.RepoID = rv.RepoID

	onPhase := func(phase string) { _ = s.reviews.SetPhase(ctx, rv.ID, phase) }
	return s.strategy.Run(ctx, aiClient, model, prov.Temperature, in, onPhase)
}

func (s *Service) resolveProvider(ctx context.Context, providerID string) (provider.Provider, error) {
	if providerID != "" {
		return s.providers.Get(ctx, providerID)
	}
	return s.providers.Default(ctx)
}

func (s *Service) contextStrategy(mode review.ContextMode, gl *gitlab.Client, token string) reviewctx.Strategy {
	if mode == review.ModeDeep {
		return reviewctx.NewDeepStrategy(gl, gitlab.NewCloner(token))
	}
	return reviewctx.NewFastStrategy(gl)
}
