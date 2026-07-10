// Package reviews orchestrates a review end to end: it creates the review,
// enqueues it, and — as the job handler — resolves the account, provider and
// model, gathers context, runs the engine, and persists the result.
package reviews

import (
	"context"
	"fmt"

	"github.com/webcloster-dev/ai-reviewer/internal/adapters/ai"
	"github.com/webcloster-dev/ai-reviewer/internal/adapters/gitlab"
	"github.com/webcloster-dev/ai-reviewer/internal/app/accounts"
	"github.com/webcloster-dev/ai-reviewer/internal/app/providers"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/provider"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/repo"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/review"
	"github.com/webcloster-dev/ai-reviewer/internal/id"
	"github.com/webcloster-dev/ai-reviewer/internal/jobs"
	"github.com/webcloster-dev/ai-reviewer/internal/review/engine"
	reviewctx "github.com/webcloster-dev/ai-reviewer/internal/review/context"
)

// Service orchestrates reviews.
type Service struct {
	reviews   review.Repository
	repos     repo.Repository
	accounts  *accounts.Service
	providers *providers.Service
	strategy  engine.Strategy
	runner    *jobs.Runner
}

// NewService wires the orchestrator. Call AttachRunner before creating reviews.
func NewService(
	reviews review.Repository,
	repos repo.Repository,
	accounts *accounts.Service,
	providers *providers.Service,
	strategy engine.Strategy,
) *Service {
	return &Service{reviews: reviews, repos: repos, accounts: accounts, providers: providers, strategy: strategy}
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

// List returns a repo's reviews (without findings), newest first.
func (s *Service) List(ctx context.Context, repoID string) ([]review.Review, error) {
	return s.reviews.ListByRepo(ctx, repoID)
}

// Get returns a review with its findings.
func (s *Service) Get(ctx context.Context, reviewID string) (review.Review, error) {
	return s.reviews.Get(ctx, reviewID)
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
	rv, err := s.reviews.Get(ctx, reviewID)
	if err != nil {
		return err
	}
	if err := s.reviews.SetStatus(ctx, reviewID, review.StatusRunning, ""); err != nil {
		return err
	}

	result, err := s.execute(ctx, rv)
	if err != nil {
		// Record the failure on the review; the job also fails so it can retry.
		_ = s.reviews.SetStatus(ctx, reviewID, review.StatusError, err.Error())
		return err
	}

	result.ID = rv.ID
	result.RepoID = rv.RepoID
	result.MRIID = rv.MRIID
	return s.reviews.Save(ctx, result)
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
