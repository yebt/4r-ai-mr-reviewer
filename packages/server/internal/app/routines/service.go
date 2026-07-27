// Package routines coordinates merge-request routines. Its first responsibility
// is the preflight: reporting which routine actions a repo's token and access
// level allow, before any routine tries to run them.
package routines

import (
	"context"
	"log"

	"github.com/webcloster-dev/ai-reviewer/internal/adapters/gitlab"
	"github.com/webcloster-dev/ai-reviewer/internal/app/accounts"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/repo"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/routine"
)

// Service computes routine preflights for tracked repos.
type Service struct {
	repos    repo.Repository
	accounts *accounts.Service
}

// NewService wires the routines service.
func NewService(repos repo.Repository, accounts *accounts.Service) *Service {
	return &Service{repos: repos, accounts: accounts}
}

// Preflight probes a repo's GitLab project and reports which routine actions the
// account's token and access level permit. It returns repo.ErrNotFound when the
// repo is unknown.
//
// TokenSelf, ProtectedBranches, and ProtectedTags are best-effort: a token type
// that cannot introspect scopes, or a project without protection rules, must not
// fail the whole preflight — only Project is required (it yields the access
// level and default branch the checks hinge on).
func (s *Service) Preflight(ctx context.Context, repoID string) (routine.Preflight, error) {
	rp, err := s.repos.Get(ctx, repoID)
	if err != nil {
		return routine.Preflight{}, err
	}

	gl, projectID, err := s.gitlabFor(ctx, rp)
	if err != nil {
		return routine.Preflight{}, err
	}

	in := routine.PreflightInput{}

	// Token scopes are unknown for OAuth/job/deploy tokens; tolerate the error.
	if info, err := gl.TokenSelf(ctx); err != nil {
		in.ScopesKnown = false
		in.Scopes = nil
	} else {
		in.ScopesKnown = true
		in.Scopes = info.Scopes
	}

	// Project is required: it carries the effective access level and default branch.
	project, err := gl.Project(ctx, projectID)
	if err != nil {
		return routine.Preflight{}, err
	}
	in.AccessLevel = effectiveAccessLevel(project.Permissions)
	in.DefaultBranch = project.DefaultBranch

	// Branch/tag protection is best-effort, but fail-closed: on failure the
	// *Known flag stays false so the preflight reports "unknown" rather than
	// silently assuming the branch/tag is unprotected.
	if branches, err := gl.ProtectedBranches(ctx, projectID); err != nil {
		log.Printf("routines: preflight: protected branches for repo %s: %v", repoID, err)
	} else {
		in.ProtectedBranches = protectedBranchLevels(branches)
		in.BranchProtectionKnown = true
	}
	if tags, err := gl.ProtectedTags(ctx, projectID); err != nil {
		log.Printf("routines: preflight: protected tags for repo %s: %v", repoID, err)
	} else {
		in.ProtectedTagsMinLevel = protectedTagsMinLevel(tags)
		in.TagProtectionKnown = true
	}

	return routine.Evaluate(in), nil
}

// gitlabFor builds a GitLab client for a repo's account and returns the client
// and encoded project path. This mirrors reviews.gitlabFor, which is unexported
// in the reviews package, so the ~5 lines are duplicated deliberately.
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

// effectiveAccessLevel is the highest of the caller's direct project access and
// inherited group access (either may be nil).
func effectiveAccessLevel(p gitlab.ProjectPermissions) int {
	level := 0
	if p.ProjectAccess != nil && p.ProjectAccess.AccessLevel > level {
		level = p.ProjectAccess.AccessLevel
	}
	if p.GroupAccess != nil && p.GroupAccess.AccessLevel > level {
		level = p.GroupAccess.AccessLevel
	}
	return level
}

// protectedBranchLevels maps each protected branch to the highest access level
// required to merge into it.
func protectedBranchLevels(branches []gitlab.ProtectedBranch) map[string]int {
	out := make(map[string]int, len(branches))
	for _, b := range branches {
		highest := 0
		for _, rule := range b.MergeAccessLevels {
			if rule.AccessLevel > highest {
				highest = rule.AccessLevel
			}
		}
		out[b.Name] = highest
	}
	return out
}

// protectedTagsMinLevel is the highest create access level across all protected
// tag rules (0 when there are no protected tags).
func protectedTagsMinLevel(tags []gitlab.ProtectedTag) int {
	highest := 0
	for _, t := range tags {
		for _, rule := range t.CreateAccessLevels {
			if rule.AccessLevel > highest {
				highest = rule.AccessLevel
			}
		}
	}
	return highest
}
