// Package mergerequests drafts and opens GitLab merge requests: it compares two
// existing branches, asks the LLM to write a title and description from the
// diff (optionally in a profile author's voice), and — as a separate step —
// opens the merge request with the (possibly edited) text. Branch creation and
// commits are out of scope: both branches must already exist.
package mergerequests

import (
	"context"
	"errors"
	"fmt"

	"github.com/webcloster-dev/ai-reviewer/internal/adapters/ai"
	"github.com/webcloster-dev/ai-reviewer/internal/adapters/gitlab"
	"github.com/webcloster-dev/ai-reviewer/internal/app/accounts"
	"github.com/webcloster-dev/ai-reviewer/internal/app/providers"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/llm"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/profile"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/provider"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/repo"
	"github.com/webcloster-dev/ai-reviewer/internal/mergerequest"
)

// ErrSameBranch is returned when source and target are the same branch. The
// HTTP layer maps it to 400 Bad Request.
var ErrSameBranch = errors.New("mergerequests: source and target branch must differ")

// ErrNoChanges is returned when the two branches have no differences to
// describe. The HTTP layer maps it to 400 Bad Request.
var ErrNoChanges = errors.New("mergerequests: no differences between the branches")

// ErrStyleGuideNotReady is returned when the chosen profile's style guide has
// not been distilled yet. The HTTP layer maps it to 409 Conflict.
var ErrStyleGuideNotReady = errors.New("mergerequests: profile style guide not ready")

// maxDiffChars caps how much diff text is sent to the model, so a huge branch
// comparison cannot blow the context window. The commit list is always sent in
// full; only the per-file diffs are truncated (with a flag the prompt shows).
const maxDiffChars = 60000

// Service drafts and opens merge requests.
type Service struct {
	repos     repo.Repository
	accounts  *accounts.Service
	providers *providers.Service
	profiles  profile.Repository
}

// NewService wires the merge-request service. accounts/providers are the app
// services (they decrypt the account token and provider API key); repos and
// profiles are the domain repositories, mirroring how reviews.Service is wired.
func NewService(repos repo.Repository, accounts *accounts.Service, providers *providers.Service, profiles profile.Repository) *Service {
	return &Service{repos: repos, accounts: accounts, providers: providers, profiles: profiles}
}

// Generate drafts a title and description for a merge request from source into
// target, using the diff between the branches. profileID is optional: when set
// and its style guide is ready, the description is written in that author's
// voice; when empty, it is written in plain English. It does NOT open the merge
// request — the caller reviews/edits the text and then calls Create.
func (s *Service) Generate(ctx context.Context, repoID, source, target, profileID string) (mergerequest.Generated, error) {
	if source == "" || target == "" {
		return mergerequest.Generated{}, fmt.Errorf("mergerequests: source and target branch are required")
	}
	if source == target {
		return mergerequest.Generated{}, ErrSameBranch
	}

	rp, err := s.repos.Get(ctx, repoID)
	if err != nil {
		return mergerequest.Generated{}, err
	}
	gl, projectID, err := s.gitlabFor(ctx, rp)
	if err != nil {
		return mergerequest.Generated{}, err
	}

	styleGuide, err := s.styleGuide(ctx, profileID)
	if err != nil {
		return mergerequest.Generated{}, err
	}

	// Commits/diffs reachable from source but not target = what the MR introduces
	// (GitLab compare from=target to=source, three-dot / merge-base semantics).
	cmp, err := gl.CompareRefs(ctx, projectID, target, source)
	if err != nil {
		return mergerequest.Generated{}, fmt.Errorf("compare branches: %w", err)
	}
	if len(cmp.Commits) == 0 && len(cmp.Diffs) == 0 {
		return mergerequest.Generated{}, ErrNoChanges
	}

	content, err := s.complete(ctx, rp, mergerequest.BuildMessages(styleGuide, buildDiffInput(source, target, cmp)))
	if err != nil {
		return mergerequest.Generated{}, err
	}
	return mergerequest.Parse(content)
}

// Create opens a merge request from source into target with the given title and
// description (the text the user reviewed after Generate, edited or not).
func (s *Service) Create(ctx context.Context, repoID, source, target, title, description string) (gitlab.MergeRequest, error) {
	if source == "" || target == "" {
		return gitlab.MergeRequest{}, fmt.Errorf("mergerequests: source and target branch are required")
	}
	if source == target {
		return gitlab.MergeRequest{}, ErrSameBranch
	}
	if title == "" {
		return gitlab.MergeRequest{}, fmt.Errorf("mergerequests: title is required")
	}

	rp, err := s.repos.Get(ctx, repoID)
	if err != nil {
		return gitlab.MergeRequest{}, err
	}
	gl, projectID, err := s.gitlabFor(ctx, rp)
	if err != nil {
		return gitlab.MergeRequest{}, err
	}
	return gl.CreateMergeRequest(ctx, projectID, source, target, title, description)
}

// styleGuide returns the distilled style guide for profileID, or "" when no
// profile was chosen. It errors when the profile is unknown or its guide is not
// ready yet.
func (s *Service) styleGuide(ctx context.Context, profileID string) (string, error) {
	if profileID == "" {
		return "", nil
	}
	p, err := s.profiles.Get(ctx, profileID)
	if err != nil {
		return "", err
	}
	if p.StyleGuideStatus != profile.StyleStatusReady {
		return "", ErrStyleGuideNotReady
	}
	return p.StyleGuide, nil
}

// complete resolves the repo's provider/model (repo override, then default
// provider), builds the AI client and runs the single drafting completion.
func (s *Service) complete(ctx context.Context, rp repo.Repo, msgs []llm.Message) (string, error) {
	prov, err := s.resolveProvider(ctx, rp.ProviderID)
	if err != nil {
		return "", err
	}
	apiKey, err := s.providers.APIKey(ctx, prov.ID)
	if err != nil {
		return "", err
	}
	model := firstNonEmpty(rp.Model, prov.Model)
	if model == "" {
		return "", fmt.Errorf("mergerequests: no model set on repo or provider %q", prov.Name)
	}
	client, err := ai.New(prov, apiKey)
	if err != nil {
		return "", err
	}
	resp, err := client.Complete(ctx, llm.Request{
		Model:       model,
		Messages:    msgs,
		Temperature: prov.Temperature,
	})
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

func (s *Service) resolveProvider(ctx context.Context, providerID string) (provider.Provider, error) {
	if providerID != "" {
		return s.providers.Get(ctx, providerID)
	}
	return s.providers.Default(ctx)
}

// gitlabFor builds a GitLab client for a repo's account and returns the client
// and the encoded project path. Mirrors routines.gitlabFor / reviews.gitlabFor,
// which are unexported in their packages, so the few lines are duplicated.
func (s *Service) gitlabFor(ctx context.Context, rp repo.Repo) (*gitlab.Client, string, error) {
	acc, err := s.accounts.Get(ctx, rp.AccountID)
	if err != nil {
		return nil, "", err
	}
	token, err := s.accounts.Token(ctx, rp.AccountID)
	if err != nil {
		return nil, "", err
	}
	projectID, err := gitlab.ProjectPath(rp.URL)
	if err != nil {
		return nil, "", err
	}
	return gitlab.NewClient(acc.BaseURL, token), projectID, nil
}

// buildDiffInput turns a GitLab comparison into the prompt's DiffInput,
// truncating the per-file diffs once their combined size passes maxDiffChars.
func buildDiffInput(source, target string, cmp gitlab.Compare) mergerequest.DiffInput {
	in := mergerequest.DiffInput{SourceBranch: source, TargetBranch: target}
	for _, c := range cmp.Commits {
		in.Commits = append(in.Commits, c.Title)
	}

	total := 0
	for _, f := range cmp.Diffs {
		if total >= maxDiffChars {
			in.Truncated = true
			break
		}
		diff := f.Diff
		if total+len(diff) > maxDiffChars {
			diff = diff[:maxDiffChars-total]
			in.Truncated = true
		}
		total += len(diff)
		in.Files = append(in.Files, mergerequest.FileDiff{
			Path:   filePath(f),
			Status: fileStatus(f),
			Diff:   diff,
		})
	}
	return in
}

// filePath is the file's new path, falling back to the old path for a deletion.
func filePath(f gitlab.FileChange) string {
	if f.NewPath != "" {
		return f.NewPath
	}
	return f.OldPath
}

// fileStatus renders a FileChange's change kind for the prompt.
func fileStatus(f gitlab.FileChange) string {
	switch {
	case f.NewFile:
		return "added"
	case f.DeletedFile:
		return "deleted"
	case f.RenamedFile:
		return "renamed"
	default:
		return "modified"
	}
}

// firstNonEmpty returns the first non-empty string, or "" if all are empty.
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
